// Package api provides low-level HTTP transport for XSOAR API calls.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tphakala/go-xsoar/internal/auth"
)

const (
	defaultHTTPTimeout   = 30 * time.Second
	defaultMaxBodySize   = 10 * 1024 * 1024 // 10MB
)

// Transport handles HTTP communication with the XSOAR API.
type Transport struct {
	BaseURL     *url.URL
	HTTPClient  *http.Client
	Credentials *auth.Credentials
	UserAgent   string
}

// NewTransport creates a Transport with the given configuration.
func NewTransport(baseURL string, creds *auth.Credentials, httpClient *http.Client) (*Transport, error) {
	if creds == nil {
		return nil, fmt.Errorf("credentials must be provided")
	}

	u, err := url.Parse(strings.TrimSuffix(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultHTTPTimeout,
		}
	}

	return &Transport{
		BaseURL:     u,
		HTTPClient:  httpClient,
		Credentials: creds,
		UserAgent:   "go-xsoar/1.0",
	}, nil
}

// Request represents an API request.
type Request struct {
	Method  string
	Path    string
	Body    any
	Headers http.Header
}

// Response represents an API response.
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Do executes an API request and returns the raw response.
func (t *Transport) Do(ctx context.Context, req *Request) (*Response, error) {
	httpReq, err := t.buildRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	httpResp, err := t.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	// Limit response body size to prevent memory exhaustion
	limitedReader := io.LimitReader(httpResp.Body, defaultMaxBodySize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if int64(len(body)) > defaultMaxBodySize {
		return nil, fmt.Errorf("response too large: exceeds %d bytes", defaultMaxBodySize)
	}

	return &Response{
		StatusCode: httpResp.StatusCode,
		Body:       body,
		Headers:    httpResp.Header,
	}, nil
}

// DoJSON executes a request and unmarshals the JSON response into result.
// It only attempts to unmarshal on success status codes (< 400).
func (t *Transport) DoJSON(ctx context.Context, req *Request, result any) (*Response, error) {
	resp, err := t.Do(ctx, req)
	if err != nil {
		return nil, err
	}

	// Only unmarshal on success status codes
	if result != nil && len(resp.Body) > 0 && resp.StatusCode < 400 {
		if err := json.Unmarshal(resp.Body, result); err != nil {
			return resp, fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return resp, nil
}

func (t *Transport) buildRequest(ctx context.Context, req *Request) (*http.Request, error) {
	u := t.BaseURL.JoinPath(req.Path)

	var bodyReader io.Reader
	if req.Body != nil {
		data, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u.String(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set default headers
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", t.UserAgent)

	// Apply authentication
	t.Credentials.Apply(httpReq)

	// Apply custom headers
	maps.Copy(httpReq.Header, req.Headers)

	return httpReq, nil
}
