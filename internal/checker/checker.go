package checker

import (
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

// CheckTCP dials target ("host:port") with the given timeout.
// Latency reflects wall-clock elapsed time whether the dial succeeds or fails.
func CheckTCP(target string, timeout time.Duration) Result {
	start := time.Now()
	dialer := net.Dialer{Timeout: timeout}

	conn, err := dialer.Dial("tcp", target)
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

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns"
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		return "refused"
	}

	return "other"
}
