package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/netip"
	"strings"
	"testing"
	"time"
)

type fakeRoundTripper func(*http.Request) (*http.Response, error)

func (f fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type syntheticTimeout struct{}

func (syntheticTimeout) Error() string   { return "synthetic-only-do-not-serialize" }
func (syntheticTimeout) Timeout() bool   { return true }
func (syntheticTimeout) Temporary() bool { return true }
func TestPerServiceFailureStageAndTimeoutNeverLeakRawErrors(t *testing.T) {
	for _, stage := range []string{"dns", "tcp", "tls", "http"} {
		t.Run(stage, func(t *testing.T) {
			client := &http.Client{Transport: fakeRoundTripper(func(req *http.Request) (*http.Response, error) {
				s := req.Context().Value(traceKey{}).(*traceState)
				s.update(func(r *serviceResult) { r.Stage = stage })
				return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syntheticTimeout{}}
			})}
			r := checkHTTPS(context.Background(), client, "https://api.ipify.org", netip.MustParseAddr("192.0.2.1"))
			if r.Service != "ipify" || r.Stage != stage || !r.Timeout || r.OK || r.Code != "tunnel-https-failed" {
				t.Fatal("phase/timeout classification failed")
			}
			b, _ := json.Marshal(r)
			for _, forbidden := range []string{"synthetic-only", "api.ipify.org", "https://", "private_key", "preshared_key"} {
				if strings.Contains(string(b), forbidden) {
					t.Fatal("unsanitized diagnostic output")
				}
			}
		})
	}
}
func TestPerServiceSuccessRetainsTLSHTTPDiagnostics(t *testing.T) {
	cs := tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	client := &http.Client{Transport: fakeRoundTripper(func(req *http.Request) (*http.Response, error) {
		trace := httptrace.ContextClientTrace(req.Context())
		trace.TLSHandshakeStart()
		trace.TLSHandshakeDone(cs, nil)
		trace.WroteRequest(httptrace.WroteRequestInfo{})
		trace.GotFirstResponseByte()
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("192.0.2.1")), Header: http.Header{}, TLS: &cs}, nil
	})}
	r := checkHTTPS(context.Background(), client, "https://api.ipify.org", netip.MustParseAddr("192.0.2.1"))
	if !r.OK || r.Stage != "complete" || !r.TLSVerified || !r.RequestWritten || r.HTTPStatus != 200 || r.FirstByteAtMS < 0 || r.Timeout {
		t.Fatal("verified response evidence lost")
	}
}
func TestHTTPFailureRemainsFailureEvenAfterTLS(t *testing.T) {
	cs := tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
	client := &http.Client{Transport: fakeRoundTripper(func(req *http.Request) (*http.Response, error) {
		trace := httptrace.ContextClientTrace(req.Context())
		trace.TLSHandshakeStart()
		trace.TLSHandshakeDone(cs, nil)
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader("not-a-success")), Header: http.Header{}, TLS: &cs}, nil
	})}
	r := checkHTTPS(context.Background(), client, "https://api.ipify.org", netip.MustParseAddr("192.0.2.1"))
	if r.OK || r.Stage != "http" || r.HTTPStatus != 503 || r.Code != "tunnel-https-response-failed" {
		t.Fatal("HTTP error became successful")
	}
}
func TestTimeoutDetectionDoesNotTreatEveryTransportErrorAsTimeout(t *testing.T) {
	s := newTrace("cloudflare")
	r := s.finish(context.Background(), errors.New("synthetic-do-not-serialize"), false, "tunnel-https-failed")
	if r.Timeout {
		t.Fatal("ordinary error mislabeled timeout")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if s.finish(ctx, context.Canceled, false, "tunnel-https-failed").Timeout {
		t.Fatal("cancellation mislabeled timeout")
	}
	if !s.finish(context.Background(), context.DeadlineExceeded, false, "tunnel-https-failed").Timeout {
		t.Fatal("deadline not labeled timeout")
	}
}
func TestOfficialPartialDeadlineBounds(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		remaining time.Duration
		count     int
		want      time.Duration
	}{{12 * time.Second, 3, 4 * time.Second}, {3 * time.Second, 3, 2 * time.Second}, {time.Second, 3, time.Second}} {
		got, e := splitDeadline(now, now.Add(tc.remaining), tc.count)
		if e != nil || got.Sub(now) != tc.want {
			t.Fatal("partial deadline differs from official algorithm")
		}
	}
	if _, e := splitDeadline(now, now, 1); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal("expired deadline accepted")
	}
}
