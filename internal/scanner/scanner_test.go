package scanner

import (
	"net"
	"testing"
	"time"

	"github.com/ulugbek21/portwatch/internal/checker"
)

// listenN opens N local listeners on random ports and returns their addresses.
// Callers must call the cleanup func to close all listeners
func listenN(t *testing.T, n int) (addrs []string, cleanup func()) {
	t.Helper()
	lns := make([]net.Listener, 0, n)
	for i := 0; i < n; i++ {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			for _, l := range lns {
				_ = l.Close()
			}
			t.Fatalf("listen %d: %v", i, err)
		}
		lns = append(lns, ln)
		addrs = append(addrs, ln.Addr().String())
	}
	return addrs, func() {
		for _, ln := range lns {
			_ = ln.Close()
		}
	}
}

func TestScan_AllOpen(t *testing.T) {
	addrs, cleanup := listenN(t, 3)
	defer cleanup()

	results := Scan(addrs, 2, time.Second)

	if len(results) != len(addrs) {
		t.Fatalf("got %d results, want %d", len(results), len(addrs))
	}

	seen := make(map[string]bool)
	for _, r := range results {
		if !r.Open {
			t.Errorf("target %s, expected Open=true, got false (err=%v)", r.Target, r.Err)
		}
		seen[r.Target] = true
	}
	for _, a := range addrs {
		if !seen[a] {
			t.Errorf("missing target in results: %s", a)
		}
	}
}

func TestScan_MixedOpenAndRefused(t *testing.T) {
	addrs, cleanup := listenN(t, 3)
	defer cleanup()

	// Open a listener on a free port, then close it. The port will refuse
	// connections for the duration of the test.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen refused: %v", err)
	}
	refusedAddr := ln.Addr().String()
	_ = ln.Close()

	targets := append([]string{refusedAddr}, addrs...)
	results := Scan(targets, 2, time.Second)

	if len(results) != len(targets) {
		t.Fatalf("got %d results, want %d", len(results), len(targets))
	}

	byTarget := make(map[string]checker.Result)
	for _, r := range results {
		byTarget[r.Target] = r
	}

	if r := byTarget[refusedAddr]; r.Open {
		t.Errorf("refused target %s: expected Open=false, got true", refusedAddr)
	} else if got := checker.Classify(r.Err); got != "refused" {
		t.Errorf("refused target %s: Classify = %q, want %q (err=%v)", refusedAddr, got, "refused", r.Err)
	}

	for _, a := range addrs {
		if r := byTarget[a]; !r.Open {
			t.Errorf("open target %s: expected Open=true, got false (err=%v)", a, r.Err)
		}
	}
}

func TestScan_LargeFanout(t *testing.T) {
	// 50 real listeners, 8 workers — shakes out deadlocks and goroutine leaks
	// under -race without needing a mocked checker.
	const n = 50
	addrs, cleanup := listenN(t, n)
	defer cleanup()

	results := Scan(addrs, 8, time.Second)

	if len(results) != n {
		t.Fatalf("got %d results, want %d", len(results), n)
	}
	open := 0
	for _, r := range results {
		if r.Open {
			open++
		}
	}
	if open != n {
		t.Errorf("expected %d open, got %d", n, open)
	}
}

func TestScan_ZeroTargets(t *testing.T) {
	// Must return promptly with an empty slice; a hung channel pattern would
	// deadlock here and `go test` would kill us with a timeout.
	done := make(chan struct{})
	var results []checker.Result
	go func() {
		results = Scan(nil, 4, time.Second)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Scan(nil, ...) deadlocked")
	}

	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}
