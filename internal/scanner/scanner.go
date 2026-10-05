package scanner

import (
	"sync"
	"time"

	"github.com/ulugbek21/portwatch/internal/checker"
)

// Scan dials every target concurrently using a pool of `workers` goroutines.
// Each check uses the provided timeout. Returns results in completion order,
// not input order (M6 sorts for presentation).
//
// If workers < 1, it's coerced to 1.
func Scan(targets []string, workers int, timeout time.Duration) []checker.Result {
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
			for target := range jobs {
				results <- checker.CheckTCP(target, timeout)
			}
		}()
	}

	// Producer: owns `jobs`, closes it when the input is exhausted.
	go func() {
		for _, t := range targets {
			jobs <- t
		}
		close(jobs)
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
