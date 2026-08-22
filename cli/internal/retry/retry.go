// Package retry provides bounded, context-aware exponential backoff for
// transient network operations. It intentionally does not retry requests that
// are known to be unauthorized, malformed, or otherwise permanent failures.
package retry

import (
	"context"
	"time"
)

const (
	defaultAttempts = 4
	baseDelay       = 150 * time.Millisecond
	maxDelay        = 2 * time.Second
)

// Do invokes operation until it succeeds, returns a non-retryable error, the
// context expires, or the bounded attempt budget is exhausted. The operation
// returns whether its error is safe to retry.
func Do(ctx context.Context, attempts int, operation func() (err error, retryable bool)) error {
	if attempts < 1 {
		attempts = defaultAttempts
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err, canRetry := operation()
		if err == nil {
			return nil
		}
		lastErr = err
		if !canRetry || attempt == attempts {
			return lastErr
		}

		delay := backoff(attempt)
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	return lastErr
}

func backoff(attempt int) time.Duration {
	delay := baseDelay
	for step := 1; step < attempt && delay < maxDelay; step++ {
		delay *= 2
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}
