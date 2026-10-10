package checker

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"
)

type Result struct {
	Target  string
	Open    bool
	Latency time.Duration
	Err     error
}

// CheckTCP dials target ("host:port") subject to ctx and a per-check timeout.
// Latency reflects wall-clock elapsed time whether the dial succeeds or fails.
func CheckTCP(ctx context.Context, target string, timeout time.Duration) Result {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	var dialer net.Dialer

	conn, err := dialer.DialContext(ctx, "tcp", target)
	elapsed := time.Since(start)

	if err != nil {
		return Result{Target: target, Open: false, Latency: elapsed, Err: err}
	}
	_ = conn.Close()

	return Result{Target: target, Open: true, Latency: elapsed}
}

// Classify returns a short, stable category tag for an error from CheckTCP.
// Returns "" when err is nil. Categories: "timeout", "dns", "refused", "other".
func Classify(err error) string {
	if err == nil {
		return ""
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "timeout"
	}

	if errors.Is(err, context.Canceled) {
		return "canceled"
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns"
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		return "refused"
	}

	return "other"
}
