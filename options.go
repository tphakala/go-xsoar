package xsoar

import (
	"net/http"
	"time"
)

// ClientOption configures a Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	baseURL    string
	keyID      string
	apiKey     string
	httpClient *http.Client
	timeout    time.Duration
	userAgent  string
}

// WithBaseURL sets the XSOAR API base URL.
func WithBaseURL(url string) ClientOption {
	return func(c *clientConfig) {
		c.baseURL = url
	}
}

// WithAPIKey sets the XSOAR 8.x API credentials.
func WithAPIKey(keyID, apiKey string) ClientOption {
	return func(c *clientConfig) {
		c.keyID = keyID
		c.apiKey = apiKey
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

// WithTimeout sets the default request timeout.
// Note: This option is ignored when WithHTTPClient is used;
// set the timeout directly on the provided client instead.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = d
	}
}

// WithUserAgent sets a custom User-Agent header.
func WithUserAgent(ua string) ClientOption {
	return func(c *clientConfig) {
		c.userAgent = ua
	}
}

// RequestOption configures individual API requests.
type RequestOption func(*requestConfig)

type requestConfig struct {
	headers http.Header
}

func newRequestConfig() *requestConfig {
	return &requestConfig{
		headers: make(http.Header),
	}
}

func (r *requestConfig) apply(opts ...RequestOption) {
	for _, opt := range opts {
		opt(r)
	}
}

// WithHeader adds a custom header to a request.
func WithHeader(key, value string) RequestOption {
	return func(r *requestConfig) {
		r.headers.Set(key, value)
	}
}

// WithHeaders adds multiple custom headers to a request.
func WithHeaders(headers map[string]string) RequestOption {
	return func(r *requestConfig) {
		for k, v := range headers {
			r.headers.Set(k, v)
		}
	}
}

// WithRequestID sets the X-Request-ID header for tracing.
func WithRequestID(id string) RequestOption {
	return WithHeader("X-Request-ID", id)
}
