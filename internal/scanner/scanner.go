package scanner

import (
	"context"
	"sync"
	"time"

	"github.com/ulugbek21/portwatch/internal/checker"
)

// Scan dials every target concurrently using a pool of `workers` goroutines.
// Each check uses the provided per-check timeout. Returns results in completion
// order. If ctx is cancelled, Scan returns promptly with partial results.
//
// If workers < 1, it's coerced to 1.
func Scan(ctx context.Context, targets []string, workers int, timeout time.Duration) []checker.Result {
	if workers < 1 {
		workers = 1
	}

	jobs := make(chan string)
	results := make(chan checker.Result)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case target, ok := <-jobs:
					if !ok {
						return
					}
					res := checker.CheckTCP(ctx, target, timeout)
					select {
					case <-ctx.Done():
						return
					case results <- res:
					}
				}
			}
		}()
	}

	// Producer: owns `jobs`, closes it on exhaustion or cancellation.
	go func() {
		defer close(jobs)
		for _, t := range targets {
			select {
			case <-ctx.Done():
				return
			case jobs <- t:
			}
		}
	}()

	// Fan-in closer: results is written by N workers, so no single worker
	// can close it. One dedicated goroutine waits for all workers, then closes.
	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]checker.Result, 0, len(targets))
	for r := range results {
		out = append(out, r)
	}

	return out
}
