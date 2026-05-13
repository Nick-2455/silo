package engram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Nick-2455/marrow/internal/domain"
)

const (
	defaultTimeout = 10 * time.Second
)

// HTTPClient is the HTTP-based implementation of domain.EngramClient.
type HTTPClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewHTTPClient creates a new Engram client.
func NewHTTPClient(baseURL, apiKey string) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// CreateResource creates a new resource in Engram.
func (c *HTTPClient) CreateResource(ctx context.Context, r domain.Resource) (string, error) {
	payload := map[string]any{
		"type": "resource",
		"content": map[string]any{
			"url":     r.URL,
			"title":   r.Title,
			"content": r.Content,
			"bucket":  string(r.Bucket),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("engram: marshal create payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url("/observations"), bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("engram: create request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	var obs Observation
	if err := json.NewDecoder(resp.Body).Decode(&obs); err != nil {
		return "", fmt.Errorf("engram: decode response: %w", err)
	}

	return obs.ID, nil
}

// GetResource retrieves a resource by ID.
func (c *HTTPClient) GetResource(ctx context.Context, id string) (domain.Resource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/observations/"+id), nil)
	if err != nil {
		return domain.Resource{}, fmt.Errorf("engram: create request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return domain.Resource{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	var obs Observation
	if err := json.NewDecoder(resp.Body).Decode(&obs); err != nil {
		return domain.Resource{}, fmt.Errorf("engram: decode response: %w", err)
	}

	return observationToResource(obs), nil
}

// SearchResources searches for resources using FTS5.
func (c *HTTPClient) SearchResources(ctx context.Context, query string) ([]domain.Resource, error) {
	reqURL := c.url("/observations/search")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("engram: create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("q", query)
	req.URL.RawQuery = q.Encode()

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("engram: decode search response: %w", err)
	}

	resources := make([]domain.Resource, 0, len(searchResp.Results))
	for _, obs := range searchResp.Results {
		resources = append(resources, observationToResource(obs))
	}

	return resources, nil
}

// GetRoadmap returns all resources grouped by bucket.
func (c *HTTPClient) GetRoadmap(ctx context.Context) (map[domain.Bucket][]domain.Resource, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/observations/roadmap"), nil)
	if err != nil {
		return nil, fmt.Errorf("engram: create request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var groups map[string][]Observation
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		return nil, fmt.Errorf("engram: decode roadmap response: %w", err)
	}

	result := make(map[domain.Bucket][]domain.Resource)
	for bucketKey, observations := range groups {
		bucket := domain.Bucket(bucketKey)
		for _, obs := range observations {
			result[bucket] = append(result[bucket], observationToResource(obs))
		}
	}

	return result, nil
}

// UpdateResource updates fields of an existing resource.
func (c *HTTPClient) UpdateResource(ctx context.Context, id string, updates map[string]any) error {
	payload := map[string]any{
		"content": updates,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("engram: marshal update payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.url("/observations/"+id), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("engram: create request: %w", err)
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("engram: update failed with status %d", resp.StatusCode)
	}

	return nil
}

// IsReachable checks if the Engram service is responding.
func (c *HTTPClient) IsReachable(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url("/health"), nil)
	if err != nil {
		return false
	}

	resp, err := c.doWithRetry(ctx, req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}

// doOnce executes an HTTP request with auth headers and error handling (no retry).
func (c *HTTPClient) doOnce(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, wrapNetworkError(err)
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		_ = resp.Body.Close()
		return nil, domain.ErrAuth
	case http.StatusTooManyRequests:
		_ = resp.Body.Close()
		return nil, domain.ErrRateLimited
	case http.StatusNotFound:
		_ = resp.Body.Close()
		return nil, domain.ErrNotFound
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("%w: status %d: %s", domain.ErrInvalidResponse, resp.StatusCode, string(body))
	}

	return resp, nil
}

// doWithRetry executes an HTTP request with exponential backoff on transient errors.
// Does NOT retry on auth, not-found, or rate-limit errors.
func (c *HTTPClient) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	err := Retry(ctx, func() error {
		var rErr error
		resp, rErr = c.doOnce(req)
		if rErr == nil {
			return nil
		}
		if errors.Is(rErr, domain.ErrAuth) || errors.Is(rErr, domain.ErrNotFound) || errors.Is(rErr, domain.ErrRateLimited) {
			// Non-retryable — wrap so Retry will propagate it immediately
			return fmt.Errorf("non-retryable: %w", rErr)
		}
		return rErr
	})
	if err != nil {
		// Unwrap non-retryable errors to preserve original sentinel
		var retryErr *RetryExceededError
		if errors.As(err, &retryErr) {
			return nil, retryErr
		}
		return resp, nil
	}
	return resp, nil
}

// url builds a full URL from a path.
func (c *HTTPClient) url(path string) string {
	return c.baseURL + path
}

// wrapNetworkError wraps network-level errors appropriately.
func wrapNetworkError(err error) error {
	return fmt.Errorf("%w: %v", domain.ErrEngramUnreachable, err)
}

// observationToResource converts an Engram Observation to a domain Resource.
func observationToResource(obs Observation) domain.Resource {
	content := obs.Content
	if content == nil {
		content = make(map[string]interface{})
	}

	r := domain.Resource{
		ID:        obs.ID,
		CreatedAt: obs.CreatedAt,
		UpdatedAt: obs.UpdatedAt,
	}

	if v, ok := content["url"].(string); ok {
		r.URL = v
	}
	if v, ok := content["title"].(string); ok {
		r.Title = v
	}
	if v, ok := content["content"].(string); ok {
		r.Content = v
	}
	if v, ok := content["bucket"].(string); ok {
		r.Bucket = domain.Bucket(v)
	}

	return r
}
