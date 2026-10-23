package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

func fixture() string {
	k := base64.StdEncoding.EncodeToString(make([]byte, 32))
	return "[Interface]\nPrivateKey=" + k + "\nAddress=10.99.0.2/32\nDNS=1.1.1.1\nMTU=1280\nJc=5\nJmin=10\nJmax=20\nS1=11\nS2=12\nS3=0\nS4=0\nH1=12345-12349\nH2=67891\nH3=67892\nH4=67893\nI1=<b 0x1234>\n[Peer]\nPublicKey=" + k + "\nPresharedKey=" + k + "\nEndpoint=192.0.2.1:51820\nAllowedIPs=0.0.0.0/0,::/0\nPersistentKeepalive=25\n"
}
func TestPreserveAWG(t *testing.T) {
	p, e := parse([]byte(fixture()))
	if e != nil {
		t.Fatal("valid fixture rejected")
	}
	for _, line := range []string{"jc=5\n", "s4=0\n", "h1=12345-12349\n", "i1=<b 0x1234>\n", "allowed_ip=::/0\n"} {
		if !strings.Contains(p.uapi, line) {
			t.Fatal("AWG data lost")
		}
	}
	if p.mtu != 1280 || p.awgFields != 12 {
		t.Fatal("AWG metadata mismatch")
	}
}
func TestRejectEndpointHostDNS(t *testing.T) {
	if _, e := parse([]byte(strings.Replace(fixture(), "192.0.2.1:51820", "vpn.example.com:51820", 1))); e == nil {
		t.Fatal("host endpoint accepted")
	}
}
func TestRejectUnknownCommands(t *testing.T) {
	if _, e := parse([]byte(strings.Replace(fixture(), "MTU=1280", "PostUp=echo secret", 1))); e == nil {
		t.Fatal("system command accepted")
	}
}
func TestRejectDuplicatePeer(t *testing.T) {
	if _, e := parse([]byte(fixture() + "[Peer]\nPublicKey=x\n")); e == nil {
		t.Fatal("multiple peers accepted")
	}
}
func TestRejectMalformedKeyAndOversize(t *testing.T) {
	if _, e := parse([]byte(strings.Replace(fixture(), "PrivateKey=", "PrivateKey=bad", 1))); e == nil {
		t.Fatal("bad key accepted")
	}
	if _, e := parse(make([]byte, inputLimit+1)); e == nil {
		t.Fatal("oversize accepted")
	}
}
