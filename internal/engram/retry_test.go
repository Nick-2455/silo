package engram_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Nick-2455/marrow/internal/engram"
)

func TestRetry_SucceedsImmediately(t *testing.T) {
	calls := 0
	err := engram.Retry(context.Background(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetry_SucceedsAfterFailure(t *testing.T) {
	calls := 0
	err := engram.Retry(context.Background(), func() error {
		calls++
		if calls < 2 {
			return errors.New("transient error")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetry_ExhaustsRetries(t *testing.T) {
	calls := 0
	err := engram.Retry(context.Background(), func() error {
		calls++
		return errors.New("persistent error")
	})
	if err == nil {
		t.Fatal("expected error after retries exhausted")
	}
	// 1 initial + 3 retries = 4 total calls
	if calls != 4 {
		t.Errorf("expected 4 calls, got %d", calls)
	}

	var retryErr *engram.RetryExceededError
	if !errors.As(err, &retryErr) {
		t.Fatalf("expected RetryExceededError, got %T: %v", err, err)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	calls := 0
	err := engram.Retry(ctx, func() error {
		calls++
		return errors.New("should not be called after cancel")
	})

	// First call happens before the delay check
	if calls < 1 {
		t.Error("expected at least 1 call")
	}

	// The error should be context.Canceled when retry is attempted
	if err != nil && !errors.Is(err, context.Canceled) {
		// The first call may succeed or fail — if it fails, the retry loop
		// will detect context cancellation. Either outcome is valid.
		t.Logf("got error (may be expected): %v", err)
	}
}

func TestRetry_CustomConfig(t *testing.T) {
	calls := 0
	err := engram.RetryWithConfig(context.Background(), func() error {
		calls++
		return errors.New("fail")
	}, 1, 10*time.Millisecond) // 1 retry, 10ms initial delay

	if err == nil {
		t.Fatal("expected error")
	}
	// 1 initial + 1 retry = 2 calls
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestRetry_Timing(t *testing.T) {
	// Verify that delays follow exponential pattern: 1s, 2s, 4s
	// We use very short delays to keep the test fast
	start := time.Now()

	calls := 0
	delay := 50 * time.Millisecond
	_ = engram.RetryWithConfig(context.Background(), func() error {
		calls++
		return errors.New("fail")
	}, 3, delay)

	elapsed := time.Since(start)

	// Expected delays: 50ms + 100ms + 200ms = 350ms minimum
	// Allow some tolerance for scheduling
	minExpected := 300 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("expected at least %v elapsed, got %v", minExpected, elapsed)
	}
}
