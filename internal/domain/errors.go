package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for Engram client operations.
var (
	ErrEngramUnreachable = errors.New("engram: service unreachable")
	ErrNotFound          = errors.New("engram: resource not found")
	ErrInvalidResponse   = errors.New("engram: invalid response from server")
)

// Sentinel errors for store operations.
var (
	ErrTriageNotFound = errors.New("store: triage position not found")
	ErrCacheMiss      = errors.New("store: cache miss")
)

// Sentinel errors for config operations.
var (
	ErrConfigNotFound = errors.New("config: configuration file not found")
	ErrConfigInvalid  = errors.New("config: invalid configuration")
)

// ErrRetryExceeded is returned when all retry attempts have been exhausted.
type ErrRetryExceeded struct {
	Op      string
	LastErr error
}

func (e *ErrRetryExceeded) Error() string {
	return fmt.Sprintf("engram: retry exceeded for %s: %v", e.Op, e.LastErr)
}

func (e *ErrRetryExceeded) Unwrap() error {
	return e.LastErr
}

// IsNotFoundError reports whether err indicates a missing resource.
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrTriageNotFound)
}
