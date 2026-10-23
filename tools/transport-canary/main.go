// Temporary sustained diagnostic; never changes host routes or writes supplied configuration.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/amnezia-vpn/amneziawg-go/v3/conn"
	"github.com/amnezia-vpn/amneziawg-go/v3/device"
	"github.com/amnezia-vpn/amneziawg-go/v3/tun/netstack"
	"io"
	"net/http"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

const inputLimit = 65536

type parsed struct {
	addresses, dns []netip.Addr
	mtu            int
	uapi           string
	awgFields      int
	hasIPv6        bool
	endpoint       netip.AddrPort
}

func parse(raw []byte) (parsed, error) {
	p := parsed{mtu: 1420}
	bad := errors.New("invalid-config")
	if len(raw) == 0 || len(raw) > inputLimit || strings.ContainsRune(string(raw), 0) {
		return p, bad
	}
	section := ""
	seen := map[string]bool{}
	iface := map[string]string{}
	peer := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			if line != "[Interface]" && line != "[Peer]" || seen[line] {
				return p, bad
			}
			section = line
			seen[line] = true
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if !ok || section == "" || val == "" {
			return p, bad
		}
		m := iface
		if section == "[Peer]" {
			m = peer
		}
		if _, ok := m[key]; ok {
			return p, bad
		}
		m[key] = val
	}
	if scanner.Err() != nil || len(seen) != 2 {
		return p, bad
	}
	keyHex := func(s string) (string, error) {
		b, e := base64.StdEncoding.DecodeString(s)
		if e != nil || len(b) != 32 {
			return "", bad
		}
		return hex.EncodeToString(b), nil
	}
	private, e := keyHex(iface["PrivateKey"])
	if e != nil {
		return p, bad
	}
	pub, e := keyHex(peer["PublicKey"])
	if e != nil {
		return p, bad
	}
	var out strings.Builder
	fmt.Fprintf(&out, "private_key=%s\n", private)
	for key, val := range iface {
		switch key {
		case "PrivateKey":
		case "Address":
			for _, s := range strings.Split(val, ",") {
				prefix, e := netip.ParsePrefix(strings.TrimSpace(s))
				if e != nil {
					return p, bad
				}
				p.addresses = append(p.addresses, prefix.Addr())
				p.hasIPv6 = p.hasIPv6 || prefix.Addr().Is6()
			}
		case "DNS":
			for _, s := range strings.Split(val, ",") {
				addr, e := netip.ParseAddr(strings.TrimSpace(s))
				if e != nil {
					return p, bad
				}
				p.dns = append(p.dns, addr)
			}
		case "MTU":
			n, e := strconv.Atoi(val)
			if e != nil || n < 1280 || n > 1500 {
				return p, bad
			}
			p.mtu = n
		case "ListenPort": // Dynamic ephemeral UDP port; never bind the source config port.
		case "Jc", "Jmin", "Jmax", "S1", "S2", "S3", "S4", "H1", "H2", "H3", "H4", "I1", "I2", "I3", "I4", "I5":
			if strings.ContainsAny(val, "\r\n") {
				return p, bad
			}
			fmt.Fprintf(&out, "%s=%s\n", strings.ToLower(key), val)
			p.awgFields++
		default:
			return p, bad
		}
	}
	if len(p.addresses) == 0 || len(p.dns) == 0 {
		return p, bad
	}
	// Endpoint must be an IP literal. Avoid resolving VPN endpoint with host DNS.
	endpoint, e := netip.ParseAddrPort(peer["Endpoint"])
	if e != nil || endpoint.Port() == 0 {
		return p, bad
	}
	p.endpoint = endpoint
	fmt.Fprintf(&out, "replace_peers=true\npublic_key=%s\nendpoint=%s\nreplace_allowed_ips=true\n", pub, endpoint.String())
	for key, val := range peer {
		switch key {
		case "PublicKey", "Endpoint":
		case "PresharedKey":
			psk, e := keyHex(val)
			if e != nil {
				return p, bad
			}
			fmt.Fprintf(&out, "preshared_key=%s\n", psk)
		case "AllowedIPs":
			for _, s := range strings.Split(val, ",") {
				prefix, e := netip.ParsePrefix(strings.TrimSpace(s))
				if e != nil {
					return p, bad
				}
				fmt.Fprintf(&out, "allowed_ip=%s\n", prefix.String())
			}
		case "PersistentKeepalive":
			n, e := strconv.Atoi(val)
			if e != nil || n < 0 || n > 65535 {
				return p, bad
			}
			fmt.Fprintf(&out, "persistent_keepalive_interval=%d\n", n)
		default:
			return p, bad
		}
	}
	if peer["AllowedIPs"] == "" {
		return p, bad
	}
	p.uapi = out.String()
	return p, nil
}

type reading struct {
	HandshakeNS int64
	TX          int64
	RX          int64
	Port        int
}
type handshakeEvent struct {
	ObservedAtMS  int64 `json:"observedAtMs"`
	HandshakeAtMS int64 `json:"handshakeAtMs"`
}
type sample struct {
	Services    []serviceResult `json:"services"`
	AtMS        int64           `json:"atMs"`
	DurationMS  int64           `json:"durationMs"`
	DNS         bool            `json:"dns"`
	HTTPSChecks int             `json:"httpsChecks"`
	OK          bool            `json:"ok"`
	Code        string          `json:"code,omitempty"`
	TX          int64           `json:"txBytes"`
	RX          int64           `json:"rxBytes"`
}
type outcome struct {
	RequestTimeoutSeconds      int              `json:"requestTimeoutSeconds"`
	DialMode                   string           `json:"dialMode"`
	OK                         bool             `json:"ok"`
	Stage                      string           `json:"stage"`
	Code                       string           `json:"code,omitempty"`
	Mode                       string           `json:"mode"`
	Handshake                  bool             `json:"handshake"`
	TX                         int64            `json:"txBytes"`
	RX                         int64            `json:"rxBytes"`
	DNS                        bool             `json:"dnsViaTunnel"`
	HTTPS                      bool             `json:"httpsViaTunnel"`
	Egress                     string           `json:"egressIPv4,omitempty"`
	EgressMatches              bool             `json:"egressMatchesExpected"`
	HTTPSChecks                int              `json:"httpsChecks"`
	DNSChecks                  int              `json:"dnsChecks"`
	FailedSamples              int              `json:"failedSamples"`
	SuccessfulSamples          int              `json:"successfulSamples"`
	Samples                    []sample         `json:"samples"`
	Handshakes                 []handshakeEvent `json:"handshakes"`
	HandshakeCount             int              `json:"handshakeCount"`
	RekeyObserved              bool             `json:"rekeyObserved"`
	PostRekeySuccessfulSamples int              `json:"postRekeySuccessfulSamples"`
	RekeyAdvanceMS             int64            `json:"rekeyAdvanceMs"`
	FirstSuccessMS             int64            `json:"firstSuccessMs"`
	LastSuccessMS              int64            `json:"lastSuccessMs"`
	SuccessfulSpanMS           int64            `json:"successfulSpanMs"`
	LongestSuccessGapMS        int64            `json:"longestSuccessGapMs"`
	ContinuousDeviceUpMS       int64            `json:"continuousDeviceUpMs"`
	RequestedSoakSeconds       int              `json:"requestedSoakSeconds"`
	SourceUDPPort              int              `json:"sourceUDPPort,omitempty"`
	AWGFields                  int              `json:"awgFields"`
	MTU                        int              `json:"mtu"`
	DurationMS                 int64            `json:"durationMs"`
	GlobalNetworkChanged       bool             `json:"globalNetworkChanged"`
}

// Raw IpcGet includes private material. Only fixed numeric fields leave this function.
func numericReading(raw string) reading {
	var r reading
	var sec, nsec int64
	for _, line := range strings.Split(raw, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "last_handshake_time_sec", "last_handshake_time_nsec", "tx_bytes", "rx_bytes", "listen_port":
		default:
			continue
		}
		n, e := strconv.ParseInt(v, 10, 64)
		if e != nil || n < 0 {
			continue
		}
		switch k {
		case "last_handshake_time_sec":
			sec = n
		case "last_handshake_time_nsec":
			if n < 1e9 {
				nsec = n
			}
		case "tx_bytes":
			r.TX += n
		case "rx_bytes":
			r.RX += n
		case "listen_port":
			if n <= 65535 {
				r.Port = int(n)
			}
		}
	}
	if sec > 0 && sec < 1<<33 {
		r.HandshakeNS = sec*1e9 + nsec
	}
	return r
}
func readDevice(dev *device.Device) reading {
	raw, e := dev.IpcGet()
	if e != nil {
		return reading{}
	}
	return numericReading(raw)
}
func recordReading(o *outcome, r reading, upAt time.Time) {
	o.TX = r.TX
	o.RX = r.RX
	o.SourceUDPPort = r.Port
	now := time.Now()
	if r.HandshakeNS >= upAt.Add(-2*time.Second).UnixNano() && r.HandshakeNS <= now.Add(2*time.Second).UnixNano() {
		o.Handshake = true
		at := (r.HandshakeNS - upAt.UnixNano()) / 1e6
		n := len(o.Handshakes)
		if n == 0 || at > o.Handshakes[n-1].HandshakeAtMS {
			o.Handshakes = append(o.Handshakes, handshakeEvent{ObservedAtMS: time.Since(upAt).Milliseconds(), HandshakeAtMS: at})
		}
	}
	o.HandshakeCount = len(o.Handshakes)
	if o.HandshakeCount >= 2 {
		o.RekeyAdvanceMS = o.Handshakes[o.HandshakeCount-1].HandshakeAtMS - o.Handshakes[0].HandshakeAtMS
		o.RekeyObserved = o.RekeyAdvanceMS >= 90000
	}
	o.ContinuousDeviceUpMS = time.Since(upAt).Milliseconds()
}
func egressIP(body []byte, cloudflare bool) (netip.Addr, bool) {
	v := strings.TrimSpace(string(body))
	if cloudflare {
		v = ""
		for _, line := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(line, "ip=") {
				v = strings.TrimPrefix(line, "ip=")
			}
		}
	}
	ip, e := netip.ParseAddr(v)
	return ip, e == nil && ip.Is4()
}
func passed(o outcome) (bool, string) {
	if o.SuccessfulSamples < 17 || o.SuccessfulSpanMS < 160000 || o.ContinuousDeviceUpMS < 160000 {
		return false, "insufficient-sustained-traffic"
	}
	if o.FailedSamples != 0 {
		return false, "periodic-check-failed"
	}
	if o.LongestSuccessGapMS > 30000 {
		return false, "traffic-check-gap-too-long"
	}
	if !o.RekeyObserved || o.HandshakeCount < 2 {
		return false, "fresh-later-handshake-unconfirmed"
	}
	if o.PostRekeySuccessfulSamples < 2 {
		return false, "post-rekey-traffic-unconfirmed"
	}
	if !o.Handshake || o.TX <= 0 || o.RX <= 0 {
		return false, "handshake-or-traffic-unconfirmed"
	}
	return true, ""
}
func main() {
	expectedFlag := flag.String("expected-egress", "", "Expected public IPv4 egress")
	duration := flag.Int("duration-seconds", 170, "Successful span target; 160..190 seconds")
	requestTimeout := flag.Int("request-timeout-seconds", 6, "Per-request HTTPS deadline; 6..15 seconds")
	flag.Parse()
	started := time.Now()
	o := outcome{Stage: "config", Mode: "continuous-soak", FirstSuccessMS: -1, LastSuccessMS: -1, RequestedSoakSeconds: *duration, RequestTimeoutSeconds: *requestTimeout, DialMode: "explicit-netstack-dns-dial", Samples: []sample{}, Handshakes: []handshakeEvent{}}
	defer func() {
		if recover() != nil {
			o.OK = false
			o.Code = "sanitized-panic"
		}
		o.DurationMS = time.Since(started).Milliseconds()
		json.NewEncoder(os.Stdout).Encode(o)
		if !o.OK {
			os.Exit(1)
		}
	}()
	expected, e := netip.ParseAddr(*expectedFlag)
	if e != nil || !expected.Is4() || *duration < 160 || *duration > 190 || *requestTimeout < 6 || *requestTimeout > 15 || flag.NArg() != 0 {
		o.Code = "invalid-options"
		return
	}
	// Wrapper must close stdin after the complete config and enforce 240s hard timeout.
	raw, e := io.ReadAll(io.LimitReader(os.Stdin, inputLimit+1))
	if e != nil {
		o.Code = "input-read-failed"
		return
	}
	p, e := parse(raw)
	for i := range raw {
		raw[i] = 0
	}
	raw = nil
	if e != nil {
		o.Code = "unsupported-or-invalid-config"
		return
	}
	o.AWGFields = p.awgFields
	o.MTU = p.mtu
	tun, tnet, e := netstack.CreateNetTUN(p.addresses, p.dns, p.mtu)
	if e != nil {
		o.Code = "netstack-create-failed"
		return
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelSilent, ""))
	defer dev.Close()
	if e = dev.IpcSet(p.uapi); e != nil {
		o.Code = "awg-config-rejected"
		return
	}
	p.uapi = ""
	if e = dev.Up(); e != nil {
		o.Code = "awg-device-up-failed"
		return
	}
	upAt := time.Now()
	defer func() { recordReading(&o, readDevice(dev), upAt) }()
	ctx, cancel := context.WithTimeout(context.Background(), 220*time.Second)
	defer cancel()
	transport := &http.Transport{DialContext: instrumentedDial(tnet, time.Duration(*requestTimeout)*time.Second), Proxy: nil, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, ForceAttemptHTTP2: false, DisableKeepAlives: true, ResponseHeaderTimeout: time.Duration(*requestTimeout-1) * time.Second, TLSHandshakeTimeout: time.Duration(*requestTimeout-1) * time.Second}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: time.Duration(*requestTimeout) * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("redirect-not-allowed") }}
	o.Stage = "soak"
	next := upAt
	for {
		if time.Until(next) > 0 {
			select {
			case <-ctx.Done():
				o.Code = "probe-deadline"
				return
			case <-time.After(time.Until(next)):
			}
		}
		begin := time.Now()
		s := sample{AtMS: time.Since(upAt).Milliseconds(), Services: []serviceResult{}}
		dnsCtx, dnsCancel := context.WithTimeout(ctx, 6*time.Second)
		ips, err := tnet.LookupContextHost(dnsCtx, "www.cloudflare.com")
		dnsCancel()
		s.DNS = err == nil && len(ips) > 0
		if s.DNS {
			o.DNSChecks++
		} else {
			s.Code = "tunnel-dns-failed"
		}
		for _, url := range []string{"https://www.cloudflare.com/cdn-cgi/trace", "https://api.ipify.org"} {
			reqCtx, reqCancel := context.WithTimeout(ctx, time.Duration(*requestTimeout)*time.Second)
			result := checkHTTPS(reqCtx, client, url, expected)
			s.Services = append(s.Services, result)
			ok, code := result.OK, result.Code
			reqCancel()
			if ok {
				s.HTTPSChecks++
				o.HTTPSChecks++
			} else if s.Code == "" {
				s.Code = code
			}
		}
		s.OK = s.DNS && s.HTTPSChecks == 2
		r := readDevice(dev)
		recordReading(&o, r, upAt)
		s.TX = r.TX
		s.RX = r.RX
		s.DurationMS = time.Since(begin).Milliseconds()
		o.Samples = append(o.Samples, s)
		successAt := time.Since(upAt).Milliseconds()
		if s.OK {
			o.SuccessfulSamples++
			if o.RekeyObserved {
				o.PostRekeySuccessfulSamples++
			}
			o.DNS = true
			o.HTTPS = true
			o.Egress = expected.String()
			o.EgressMatches = true
			if o.FirstSuccessMS < 0 {
				o.FirstSuccessMS = successAt
			}
			if o.LastSuccessMS >= 0 && successAt-o.LastSuccessMS > o.LongestSuccessGapMS {
				o.LongestSuccessGapMS = successAt - o.LastSuccessMS
			}
			o.LastSuccessMS = successAt
			o.SuccessfulSpanMS = o.LastSuccessMS - o.FirstSuccessMS
		} else {
			o.FailedSamples++
		}
		if o.FirstSuccessMS >= 0 && time.Since(upAt).Milliseconds()-o.FirstSuccessMS >= int64(*duration)*1000 {
			break
		}
		if time.Since(upAt) >= 205*time.Second {
			break
		}
		next = next.Add(10 * time.Second)
		if next.Before(time.Now()) {
			next = time.Now()
		}
	}
	recordReading(&o, readDevice(dev), upAt)
	o.OK, o.Code = passed(o)
	if o.OK {
		o.Stage = "complete"
	}
}
