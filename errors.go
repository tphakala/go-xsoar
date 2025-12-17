package xsoar

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Sentinel errors for common failure modes.
var (
	ErrNoCredentials = errors.New("xsoar: no credentials configured")
	ErrNoBaseURL     = errors.New("xsoar: no base URL configured")
)

// APIError represents a general XSOAR API error.
type APIError struct {
	StatusCode int    `json:"status"`
	Message    string `json:"message"`
	RequestID  string `json:"requestId,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("xsoar: API error %d: %s (request_id=%s)", e.StatusCode, e.Message, e.RequestID)
	}
	return fmt.Sprintf("xsoar: API error %d: %s", e.StatusCode, e.Message)
}

// AuthenticationError indicates authentication failure (401/403).
type AuthenticationError struct {
	APIError
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("xsoar: authentication failed: %s", e.Message)
}

// As implements error unwrapping for errors.As to match *APIError.
func (e *AuthenticationError) As(target any) bool {
	if t, ok := target.(**APIError); ok {
		*t = &e.APIError
		return true
	}
	return false
}

// NotFoundError indicates the requested resource was not found (404).
type NotFoundError struct {
	APIError
	ResourceType string
	ResourceID   string
}

func (e *NotFoundError) Error() string {
	if e.ResourceType != "" && e.ResourceID != "" {
		return fmt.Sprintf("xsoar: %s not found: %s", e.ResourceType, e.ResourceID)
	}
	return fmt.Sprintf("xsoar: resource not found: %s", e.Message)
}

// As implements error unwrapping for errors.As to match *APIError.
func (e *NotFoundError) As(target any) bool {
	if t, ok := target.(**APIError); ok {
		*t = &e.APIError
		return true
	}
	return false
}

// ValidationError indicates invalid request data (400).
type ValidationError struct {
	APIError
	Fields map[string]string `json:"fields,omitempty"`
}

func (e *ValidationError) Error() string {
	if len(e.Fields) > 0 {
		return fmt.Sprintf("xsoar: validation error: %s (fields: %v)", e.Message, e.Fields)
	}
	return fmt.Sprintf("xsoar: validation error: %s", e.Message)
}

// As implements error unwrapping for errors.As to match *APIError.
func (e *ValidationError) As(target any) bool {
	if t, ok := target.(**APIError); ok {
		*t = &e.APIError
		return true
	}
	return false
}

// RateLimitError indicates the API rate limit was exceeded (429).
type RateLimitError struct {
	APIError
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("xsoar: rate limit exceeded, retry after %s", e.RetryAfter)
	}
	return "xsoar: rate limit exceeded"
}

// As implements error unwrapping for errors.As to match *APIError.
func (e *RateLimitError) As(target any) bool {
	if t, ok := target.(**APIError); ok {
		*t = &e.APIError
		return true
	}
	return false
}

// ServerError indicates an internal server error (5xx).
type ServerError struct {
	APIError
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("xsoar: server error %d: %s", e.StatusCode, e.Message)
}

// As implements error unwrapping for errors.As to match *APIError.
func (e *ServerError) As(target any) bool {
	if t, ok := target.(**APIError); ok {
		*t = &e.APIError
		return true
	}
	return false
}

// parseError converts an HTTP response into the appropriate error type.
func parseError(statusCode int, body []byte, headers http.Header) error {
	requestID := headers.Get("X-Request-ID")
	base := APIError{
		StatusCode: statusCode,
		RequestID:  requestID,
	}

	// Try to parse structured JSON error response
	if err := json.Unmarshal(body, &base); err != nil {
		// Fallback to raw body if not valid JSON
		base.Message = string(body)
	}

	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &AuthenticationError{APIError: base}
	case statusCode == http.StatusNotFound:
		return &NotFoundError{APIError: base}
	case statusCode == http.StatusBadRequest:
		validationErr := &ValidationError{APIError: base}
		// Best-effort parse of field-level validation errors
		var fieldData struct {
			Fields map[string]string `json:"fields"`
		}
		if json.Unmarshal(body, &fieldData) == nil && len(fieldData.Fields) > 0 {
			validationErr.Fields = fieldData.Fields
		}
		return validationErr
	case statusCode == http.StatusTooManyRequests:
		return &RateLimitError{
			APIError:   base,
			RetryAfter: parseRetryAfter(headers.Get("Retry-After")),
		}
	case statusCode >= http.StatusInternalServerError:
		return &ServerError{APIError: base}
	default:
		return &base
	}
}

// parseRetryAfter parses the Retry-After header value.
// It handles both seconds (integer) and HTTP-date formats.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	// Try parsing as seconds first
	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date (RFC 1123)
	if t, err := time.Parse(time.RFC1123, value); err == nil {
		duration := time.Until(t)
		if duration > 0 {
			return duration
		}
	}

	return 0
}
