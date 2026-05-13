package engram

import (
	"context"
	"math"
	"time"
)

const (
	defaultMaxRetries   = 3
	defaultInitialDelay = 1 * time.Second
)

// Retry wraps an operation with exponential backoff.
// Delays: 1s, 2s, 4s (max 3 retries).
func Retry(ctx context.Context, op func() error) error {
	return RetryWithConfig(ctx, op, defaultMaxRetries, defaultInitialDelay)
}

// RetryWithConfig wraps an operation with configurable exponential backoff.
func RetryWithConfig(ctx context.Context, op func() error, maxRetries int, initialDelay time.Duration) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := initialDelay * time.Duration(int(math.Pow(2, float64(attempt-1))))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		lastErr = op()
		if lastErr == nil {
			return nil
		}
	}

	return &RetryExceededError{
		Attempts: maxRetries + 1,
		LastErr:  lastErr,
	}
}

// RetryExceededError is returned when all retry attempts have been exhausted.
type RetryExceededError struct {
	Attempts int
	LastErr  error
}

func (e *RetryExceededError) Error() string {
	return "engram: all retry attempts exhausted"
}

func (e *RetryExceededError) Unwrap() error {
	return e.LastErr
}
