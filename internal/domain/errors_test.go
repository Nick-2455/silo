package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Nick-2455/marrow/internal/domain"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{"ErrEngramUnreachable", domain.ErrEngramUnreachable, domain.ErrEngramUnreachable},
		{"ErrNotFound", domain.ErrNotFound, domain.ErrNotFound},
		{"ErrInvalidResponse", domain.ErrInvalidResponse, domain.ErrInvalidResponse},
		{"ErrTriageNotFound", domain.ErrTriageNotFound, domain.ErrTriageNotFound},
		{"ErrCacheMiss", domain.ErrCacheMiss, domain.ErrCacheMiss},
		{"ErrConfigNotFound", domain.ErrConfigNotFound, domain.ErrConfigNotFound},
		{"ErrConfigInvalid", domain.ErrConfigInvalid, domain.ErrConfigInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.err, tt.want) {
				t.Errorf("errors.Is(%v, %v) = false, want true", tt.err, tt.want)
			}
		})
	}
}

func TestErrRetryExceeded(t *testing.T) {
	inner := errors.New("connection refused")
	err := &domain.ErrRetryExceeded{
		Op:      "CreateResource",
		LastErr: inner,
	}

	// Test Error() message
	msg := err.Error()
	if msg == "" {
		t.Error("Error() returned empty string")
	}

	// Test Unwrap
	unwrapped := err.Unwrap()
	if unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}

	// Test errors.Is with wrapped error
	wrapped := fmt.Errorf("operation failed: %w", err)
	if !errors.Is(wrapped, inner) {
		t.Error("errors.Is should find inner error through ErrRetryExceeded")
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"ErrNotFound", domain.ErrNotFound, true},
		{"ErrTriageNotFound", domain.ErrTriageNotFound, true},
		{"wrapped not found", fmt.Errorf("wrapper: %w", domain.ErrNotFound), true},
		{"wrapped triage", fmt.Errorf("wrapper: %w", domain.ErrTriageNotFound), true},
		{"not found", domain.ErrEngramUnreachable, false},
		{"nil", nil, false},
		{"random", errors.New("something else"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.IsNotFoundError(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFoundError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that sentinel errors can be wrapped and still match
	wrapped := fmt.Errorf("engram: do request: %w", domain.ErrEngramUnreachable)
	if !errors.Is(wrapped, domain.ErrEngramUnreachable) {
		t.Error("wrapped ErrEngramUnreachable should match sentinel")
	}

	// Test double wrapping
	doubleWrapped := fmt.Errorf("app: bootstrap: %w", wrapped)
	if !errors.Is(doubleWrapped, domain.ErrEngramUnreachable) {
		t.Error("double-wrapped ErrEngramUnreachable should match sentinel")
	}

	// Test ErrRetryExceeded wraps properly
	inner := errors.New("timeout")
	retryErr := &domain.ErrRetryExceeded{Op: "Search", LastErr: inner}
	if retryErr.Unwrap() != inner {
		t.Error("ErrRetryExceeded should unwrap to LastErr")
	}
}
