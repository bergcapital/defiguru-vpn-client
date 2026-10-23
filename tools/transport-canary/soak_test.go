package main

import (
	"testing"
	"time"
)

func TestNumericReadingDiscardsSecretAndMalformedFields(t *testing.T) {
	raw := "private_key=not-a-counter\npublic_key=not-a-counter\npreshared_key=not-a-counter\nlast_handshake_time_sec=1700000000\nlast_handshake_time_nsec=123456789\ntx_bytes=1200\nrx_bytes=3400\nlisten_port=55000\nunknown=777\n"
	r := numericReading(raw)
	if r.TX != 1200 || r.RX != 3400 || r.Port != 55000 || r.HandshakeNS != 1700000000123456789 {
		t.Fatal("numeric whitelist mismatch")
	}
	invalid := numericReading("tx_bytes=-1\nrx_bytes=oops\nlisten_port=99999\nlast_handshake_time_sec=9223372036854775807\n")
	if invalid != (reading{}) {
		t.Fatal("invalid numeric values accepted")
	}
}
func TestLateHandshakeDistinctAndRepeatedSnapshotDoesNotInflate(t *testing.T) {
	up := time.Now().Add(-170 * time.Second)
	o := outcome{}
	a := reading{HandshakeNS: up.Add(time.Second).UnixNano(), TX: 1, RX: 1}
	recordReading(&o, a, up)
	recordReading(&o, a, up)
	if o.HandshakeCount != 1 || o.RekeyObserved {
		t.Fatal("same handshake incorrectly treated as rekey")
	}
	a.HandshakeNS = up.Add(124 * time.Second).UnixNano()
	recordReading(&o, a, up)
	if o.HandshakeCount != 2 || !o.RekeyObserved || o.RekeyAdvanceMS != 123000 {
		t.Fatal("late distinct handshake not verified")
	}
	a.HandshakeNS = up.Add(200 * time.Second).UnixNano()
	recordReading(&o, a, up)
	if o.HandshakeCount != 2 {
		t.Fatal("future timestamp incorrectly accepted")
	}
}
func goodOutcome() outcome {
	return outcome{SuccessfulSamples: 18, SuccessfulSpanMS: 170000, ContinuousDeviceUpMS: 171000, LongestSuccessGapMS: 10000, RekeyObserved: true, HandshakeCount: 2, PostRekeySuccessfulSamples: 4, Handshake: true, TX: 100, RX: 200}
}
func TestSuccessRequiresTrafficThroughRekey(t *testing.T) {
	o := goodOutcome()
	if ok, _ := passed(o); !ok {
		t.Fatal("complete sustained proof rejected")
	}
	for _, mutate := range []func(*outcome){func(o *outcome) { o.HandshakeCount = 1 }, func(o *outcome) { o.RekeyObserved = false }, func(o *outcome) { o.PostRekeySuccessfulSamples = 0 }} {
		o = goodOutcome()
		mutate(&o)
		if ok, _ := passed(o); ok {
			t.Fatal("missing late-key evidence accepted")
		}
	}
}
func TestSuccessRejectsShortRunFailuresAndLongGaps(t *testing.T) {
	for _, mutate := range []func(*outcome){func(o *outcome) { o.SuccessfulSpanMS = 159999 }, func(o *outcome) { o.ContinuousDeviceUpMS = 159999 }, func(o *outcome) { o.FailedSamples = 1 }, func(o *outcome) { o.SuccessfulSamples = 16 }, func(o *outcome) { o.LongestSuccessGapMS = 30001 }} {
		o := goodOutcome()
		mutate(&o)
		if ok, _ := passed(o); ok {
			t.Fatal("incomplete sustained proof accepted")
		}
	}
}
func TestEgressParsingRequiresActualIPv4(t *testing.T) {
	ip, ok := egressIP([]byte("fl=abc\nip=192.0.2.1\n"), true)
	if !ok || ip.String() != "192.0.2.1" {
		t.Fatal("cloudflare egress not parsed")
	}
	for _, body := range []string{"", "192.0.2.1<script>", "2001:db8::1", "private_key=value"} {
		if _, ok := egressIP([]byte(body), false); ok {
			t.Fatal("invalid egress accepted")
		}
	}
}
