package checker

import (
	"errors"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestCheckTCP_Open(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	res := CheckTCP(ln.Addr().String(), time.Second)

	if !res.Open {
		t.Errorf("expected Open=true, got false (err=%v)", res.Err)
	}
	if res.Err != nil {
		t.Errorf("expected Err=nil, got %v", res.Err)
	}
	if res.Latency <= 0 {
		t.Errorf("expected positive Latency, got %v", res.Latency)
	}
}

func TestCheckTCP_Refused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	res := CheckTCP(addr, time.Second)
	if res.Open {
		t.Errorf("expected Open=false, got true (err=%v)", res.Err)
	}
	if got := Classify(res.Err); got != "refused" {
		t.Errorf("Classify = %q, want %q (err=%v)", got, "refused", res.Err)
	}
}

func TestCheckTCP_Timeout(t *testing.T) {
	// TEST-NET-1 (RFC 5737) — reserved, non-routable. Dials here hang until timeout.
	res := CheckTCP("192.0.2.1:80", 50*time.Millisecond)

	if res.Open {
		t.Fatalf("expected Open=false, got true")
	}
	if got := Classify(res.Err); got != "timeout" {
		t.Errorf("Classify = %q, want %q (err=%v)", got, "timeout", res.Err)
	}
}

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"dns", &net.DNSError{Err: "no such host", Name: "nope.invalid"}, "dns"},
		{"refused", syscall.ECONNREFUSED, "refused"},
		{"wrapped refused", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, "refused"},
		{"other", errors.New("something else"), "other"},
		{"timeout", fakeTimeoutErr{}, "timeout"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(tc.err); got != tc.want {
				t.Errorf("Classify(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

// fakeTimeoutErr satisfies net.Error with Timeout() == true,
// so TestClassify doesn't depend on real network timing.
type fakeTimeoutErr struct{}

func (fakeTimeoutErr) Error() string   { return "fake timeout" }
func (fakeTimeoutErr) Timeout() bool   { return true }
func (fakeTimeoutErr) Temporary() bool { return false }
