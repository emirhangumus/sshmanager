// Package progress displays operation stages and elapsed-time heartbeats.
package progress

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// Run reports stages immediately, and repeats the current stage while it blocks.
// Messages go to the supplied diagnostic stream, leaving command stdout intact.
func Run(out io.Writer, operation func(func(string)) error) error {
	return run(out, operation, 2*time.Second)
}

func run(out io.Writer, operation func(func(string)) error, interval time.Duration) error {
	var mu sync.Mutex
	var stage string
	var since time.Time
	report := func(message string) {
		mu.Lock()
		defer mu.Unlock()
		if stage == message {
			return
		}
		stage, since = message, time.Now()
		_, _ = fmt.Fprintln(out, message)
	}
	done, stopped := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				mu.Lock()
				if stage != "" {
					_, _ = fmt.Fprintf(out, "Still working: %s (%s elapsed)\n", stage, time.Since(since).Round(time.Second))
				}
				mu.Unlock()
			}
		}
	}()
	defer func() { close(done); <-stopped }()
	return operation(report)
}
