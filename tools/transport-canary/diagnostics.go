package main

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/amnezia-vpn/amneziawg-go/v3/tun/netstack"
)

type serviceResult struct {
	Service        string `json:"service"`
	OK             bool   `json:"ok"`
	Stage          string `json:"stage"`
	Code           string `json:"code,omitempty"`
	ElapsedMS      int64  `json:"elapsedMs"`
	Timeout        bool   `json:"timeout"`
	DNSMS          int64  `json:"dnsMs"`
	TCPMS          int64  `json:"tcpMs"`
	TLSMS          int64  `json:"tlsMs"`
	FirstByteAtMS  int64  `json:"firstByteAtMs"`
	HTTPStatus     int    `json:"httpStatus,omitempty"`
	TLSVerified    bool   `json:"tlsVerified"`
	RequestWritten bool   `json:"requestWritten"`
}
type traceState struct {
	mu       sync.Mutex
	start    time.Time
	tlsStart time.Time
	r        serviceResult
}
type traceKey struct{}

func newTrace(service string) *traceState {
	return &traceState{start: time.Now(), r: serviceResult{Service: service, Stage: "dns", DNSMS: -1, TCPMS: -1, TLSMS: -1, FirstByteAtMS: -1}}
}
func (s *traceState) update(f func(*serviceResult)) { s.mu.Lock(); defer s.mu.Unlock(); f(&s.r) }
func (s *traceState) finish(ctx context.Context, err error, ok bool, code string) serviceResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.r.OK = ok
	s.r.Code = code
	s.r.ElapsedMS = time.Since(s.start).Milliseconds()
	s.r.Timeout = s.r.Timeout || errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded)
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		s.r.Timeout = true
	}
	if ok {
		s.r.Stage = "complete"
	}
	return s.r
}
func (s *traceState) trace() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		TLSHandshakeStart: func() { s.mu.Lock(); s.tlsStart = time.Now(); s.r.Stage = "tls"; s.mu.Unlock() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.r.TLSMS = time.Since(s.tlsStart).Milliseconds()
			if err == nil {
				s.r.TLSVerified = len(cs.VerifiedChains) > 0
				s.r.Stage = "http"
			}
		},
		WroteRequest: func(i httptrace.WroteRequestInfo) {
			s.update(func(r *serviceResult) { r.Stage = "http"; r.RequestWritten = i.Err == nil })
		},
		GotFirstResponseByte: func() {
			s.update(func(r *serviceResult) { r.Stage = "http"; r.FirstByteAtMS = time.Since(s.start).Milliseconds() })
		},
	}
}

// Equivalent TCP address ordering and partial-deadline algorithm to the pinned
// official netstack.DialContext. DNS is factored out only to measure its phase.
// DNS and TCP remain in the same userspace stack; no host resolver is available.
func instrumentedDial(tnet *netstack.Net, timeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		s, ok := ctx.Value(traceKey{}).(*traceState)
		if !ok {
			return nil, errors.New("missing-trace")
		}
		if network != "tcp" && network != "tcp4" && network != "tcp6" {
			return nil, errors.New("unsupported-network")
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		host, portText, e := net.SplitHostPort(address)
		if e != nil {
			return nil, errors.New("invalid-dial-address")
		}
		port, e := strconv.Atoi(portText)
		if e != nil || port < 1 || port > 65535 {
			return nil, errors.New("invalid-dial-port")
		}
		begin := time.Now()
		s.update(func(r *serviceResult) { r.Stage = "dns" })
		all, e := tnet.LookupContextHost(ctx, host)
		s.update(func(r *serviceResult) { r.DNSMS = time.Since(begin).Milliseconds() })
		if e != nil {
			return nil, e
		}
		var addrs []netip.AddrPort
		for _, raw := range all {
			ip, e := netip.ParseAddr(raw)
			if e == nil && ((ip.Is4() && network != "tcp6") || (ip.Is6() && network != "tcp4")) {
				addrs = append(addrs, netip.AddrPortFrom(ip, uint16(port)))
			}
		}
		if len(addrs) == 0 {
			return nil, errors.New("no-suitable-address")
		}
		begin = time.Now()
		s.update(func(r *serviceResult) { r.Stage = "tcp" })
		defer func() { s.update(func(r *serviceResult) { r.TCPMS = time.Since(begin).Milliseconds() }) }()
		var first error
		anyTimeout := false
		for i, addr := range addrs {
			if e := ctx.Err(); e != nil {
				s.update(func(r *serviceResult) { r.Timeout = errors.Is(e, context.DeadlineExceeded) })
				return nil, e
			}
			dialCtx := ctx
			dialCancel := func() {}
			if deadline, ok := ctx.Deadline(); ok {
				partial, e := splitDeadline(time.Now(), deadline, len(addrs)-i)
				if e != nil {
					if first == nil {
						first = e
					}
					break
				}
				if partial.Before(deadline) {
					dialCtx, dialCancel = context.WithDeadline(ctx, partial)
				}
			}
			c, e := tnet.DialContextTCPAddrPort(dialCtx, addr)
			dialCancel()
			if e == nil {
				return c, nil
			}
			var ne net.Error
			anyTimeout = anyTimeout || errors.Is(e, context.DeadlineExceeded) || (errors.As(e, &ne) && ne.Timeout())
			if first == nil {
				first = e
			}
		}
		if first == nil {
			first = errors.New("missing-address")
		}
		s.update(func(r *serviceResult) { r.Timeout = anyTimeout || errors.Is(ctx.Err(), context.DeadlineExceeded) })
		return nil, first
	}
}
func splitDeadline(now, deadline time.Time, remaining int) (time.Time, error) {
	left := deadline.Sub(now)
	if left <= 0 || remaining < 1 {
		return time.Time{}, context.DeadlineExceeded
	}
	timeout := left / time.Duration(remaining)
	if timeout < 2*time.Second {
		if left < 2*time.Second {
			timeout = left
		} else {
			timeout = 2 * time.Second
		}
	}
	return now.Add(timeout), nil
}
func serviceName(url string) string {
	switch url {
	case "https://www.cloudflare.com/cdn-cgi/trace":
		return "cloudflare"
	case "https://api.ipify.org":
		return "ipify"
	default:
		return "unsupported"
	}
}
func checkHTTPS(ctx context.Context, client *http.Client, url string, expected netip.Addr) serviceResult {
	s := newTrace(serviceName(url))
	if s.r.Service == "unsupported" {
		return s.finish(ctx, nil, false, "unsupported-service")
	}
	ctx = context.WithValue(ctx, traceKey{}, s)
	ctx = httptrace.WithClientTrace(ctx, s.trace())
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if e != nil {
		return s.finish(ctx, e, false, "https-request-failed")
	}
	req.Header.Set("User-Agent", "DeFiGuru-VPN-Sustained-Probe/1.0")
	resp, e := client.Do(req)
	if e != nil {
		return s.finish(ctx, e, false, "tunnel-https-failed")
	}
	s.update(func(r *serviceResult) { r.Stage = "http"; r.HTTPStatus = resp.StatusCode })
	body, e := io.ReadAll(io.LimitReader(resp.Body, 8193))
	resp.Body.Close()
	if e != nil || len(body) > 8192 || resp.StatusCode != 200 || resp.TLS == nil || len(resp.TLS.VerifiedChains) == 0 {
		return s.finish(ctx, e, false, "tunnel-https-response-failed")
	}
	ip, ok := egressIP(body, strings.Contains(url, "cloudflare"))
	if !ok {
		return s.finish(ctx, nil, false, "invalid-egress-response")
	}
	if ip != expected {
		return s.finish(ctx, nil, false, "unexpected-egress")
	}
	return s.finish(ctx, nil, true, "")
}
