package engram

import (
	"context"
	"errors"

	"github.com/Nick-2455/silo/internal/domain"
)

var ErrEngramUnavailable = errors.New("engram: unavailable — running in degraded mode")

// NoopClient is a no-op EngramClient used when Engram is unavailable.
type NoopClient struct{}

func (n *NoopClient) CreateResource(_ context.Context, _ domain.Resource) (string, error) {
	return "", ErrEngramUnavailable
}

func (n *NoopClient) GetResource(_ context.Context, id string) (domain.Resource, error) {
	return domain.Resource{ID: id}, ErrEngramUnavailable
}

func (n *NoopClient) SearchResources(_ context.Context, _ string) ([]domain.Resource, error) {
	return nil, ErrEngramUnavailable
}

func (n *NoopClient) UpdateResource(_ context.Context, _ string, _ map[string]any) error {
	return ErrEngramUnavailable
}

func (n *NoopClient) IsReachable(_ context.Context) bool {
	return false
}

func (n *NoopClient) SaveNode(_ context.Context, _, _ string, _ map[string]any, _, _ string) (string, error) {
	return "", ErrEngramUnavailable
}

func (n *NoopClient) UpdateNode(_ context.Context, _ string, _ map[string]any) error {
	return ErrEngramUnavailable
}

func (n *NoopClient) SearchByProject(_ context.Context, _ string) ([]domain.DiscoveredObservation, error) {
	return nil, ErrEngramUnavailable
}

func (n *NoopClient) Close() error {
	return nil
}