package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/ulugbek21/portwatch/internal/scanner"
)

func main() {
	timeout := flag.Duration("timeout", 2*time.Second, "per-check timeout")
	workers := flag.Int("workers", 16, "worker pool size")
	flag.Parse()

	targets := flag.Args()
	if len(targets) == 0 {
		fmt.Fprintln(os.Stderr, "usage: portwatch [--workers N] [--timeout D] host:port ...")
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	results := scanner.Scan(ctx, targets, *workers, *timeout)

	failed := 0
	for _, r := range results {
		status := "OPEN"
		if !r.Open {
			status = "Closed"
			failed++
		}
		fmt.Printf("%-30s %-6s %v\n", r.Target, status, r.Latency.Round(time.Millisecond))
	}

	if failed > 0 {
		os.Exit(1)
	}
}
