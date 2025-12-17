package xsoar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/go-xsoar"
)

func TestAPIError(t *testing.T) {
	t.Run("Error without request ID", func(t *testing.T) {
		err := &xsoar.APIError{
			StatusCode: 500,
			Message:    "internal error",
		}
		assert.Equal(t, "xsoar: API error 500: internal error", err.Error())
	})

	t.Run("Error with request ID", func(t *testing.T) {
		err := &xsoar.APIError{
			StatusCode: 500,
			Message:    "internal error",
			RequestID:  "req-123",
		}
		assert.Equal(t, "xsoar: API error 500: internal error (request_id=req-123)", err.Error())
	})
}

func TestAuthenticationError(t *testing.T) {
	err := &xsoar.AuthenticationError{
		APIError: xsoar.APIError{
			StatusCode: 401,
			Message:    "invalid API key",
		},
	}
	assert.Equal(t, "xsoar: authentication failed: invalid API key", err.Error())

	// Test errors.As
	var apiErr *xsoar.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 401, apiErr.StatusCode)
}

func TestNotFoundError(t *testing.T) {
	t.Run("with resource info", func(t *testing.T) {
		err := &xsoar.NotFoundError{
			APIError:     xsoar.APIError{StatusCode: 404},
			ResourceType: "incident",
			ResourceID:   "inc-123",
		}
		assert.Equal(t, "xsoar: incident not found: inc-123", err.Error())
	})

	t.Run("without resource info", func(t *testing.T) {
		err := &xsoar.NotFoundError{
			APIError: xsoar.APIError{
				StatusCode: 404,
				Message:    "not found",
			},
		}
		assert.Equal(t, "xsoar: resource not found: not found", err.Error())
	})
}

func TestValidationError(t *testing.T) {
	t.Run("with fields", func(t *testing.T) {
		err := &xsoar.ValidationError{
			APIError: xsoar.APIError{
				StatusCode: 400,
				Message:    "invalid request",
			},
			Fields: map[string]string{
				"name": "required",
			},
		}
		assert.Contains(t, err.Error(), "xsoar: validation error: invalid request")
		assert.Contains(t, err.Error(), "name")
	})

	t.Run("without fields", func(t *testing.T) {
		err := &xsoar.ValidationError{
			APIError: xsoar.APIError{
				StatusCode: 400,
				Message:    "bad request",
			},
		}
		assert.Equal(t, "xsoar: validation error: bad request", err.Error())
	})
}

func TestRateLimitError(t *testing.T) {
	t.Run("with retry-after", func(t *testing.T) {
		err := &xsoar.RateLimitError{
			APIError:   xsoar.APIError{StatusCode: 429},
			RetryAfter: 30 * time.Second,
		}
		assert.Equal(t, "xsoar: rate limit exceeded, retry after 30s", err.Error())
	})

	t.Run("without retry-after", func(t *testing.T) {
		err := &xsoar.RateLimitError{
			APIError: xsoar.APIError{StatusCode: 429},
		}
		assert.Equal(t, "xsoar: rate limit exceeded", err.Error())
	})
}

func TestServerError(t *testing.T) {
	err := &xsoar.ServerError{
		APIError: xsoar.APIError{
			StatusCode: 503,
			Message:    "service unavailable",
		},
	}
	assert.Equal(t, "xsoar: server error 503: service unavailable", err.Error())
}

func TestErrorsAs(t *testing.T) {
	// Test that all error types can be detected with errors.As
	tests := []struct {
		name string
		err  error
	}{
		{"AuthenticationError", &xsoar.AuthenticationError{APIError: xsoar.APIError{StatusCode: 401}}},
		{"NotFoundError", &xsoar.NotFoundError{APIError: xsoar.APIError{StatusCode: 404}}},
		{"ValidationError", &xsoar.ValidationError{APIError: xsoar.APIError{StatusCode: 400}}},
		{"RateLimitError", &xsoar.RateLimitError{APIError: xsoar.APIError{StatusCode: 429}}},
		{"ServerError", &xsoar.ServerError{APIError: xsoar.APIError{StatusCode: 500}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var apiErr *xsoar.APIError
			require.ErrorAs(t, tt.err, &apiErr, "should be detectable as APIError")
		})
	}
}
