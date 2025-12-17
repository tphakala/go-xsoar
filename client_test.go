package xsoar_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tphakala/go-xsoar"
)

func TestNewClient(t *testing.T) {
	t.Run("success with required options", func(t *testing.T) {
		client, err := xsoar.NewClient(
			xsoar.WithBaseURL("https://api.xsoar.example.com"),
			xsoar.WithAPIKey("key-id", "api-key"),
		)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NotNil(t, client.Incidents)
		assert.Equal(t, "https://api.xsoar.example.com", client.BaseURL())
	})

	t.Run("error without base URL", func(t *testing.T) {
		_, err := xsoar.NewClient(
			xsoar.WithAPIKey("key-id", "api-key"),
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, xsoar.ErrNoBaseURL)
	})

	t.Run("error without credentials", func(t *testing.T) {
		_, err := xsoar.NewClient(
			xsoar.WithBaseURL("https://api.xsoar.example.com"),
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, xsoar.ErrNoCredentials)
	})

	t.Run("error with partial credentials", func(t *testing.T) {
		_, err := xsoar.NewClient(
			xsoar.WithBaseURL("https://api.xsoar.example.com"),
			xsoar.WithAPIKey("key-id", ""),
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, xsoar.ErrNoCredentials)
	})

	t.Run("success with all options", func(t *testing.T) {
		client, err := xsoar.NewClient(
			xsoar.WithBaseURL("https://api.xsoar.example.com"),
			xsoar.WithAPIKey("key-id", "api-key"),
			xsoar.WithUserAgent("test-agent/1.0"),
		)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("success with custom timeout", func(t *testing.T) {
		client, err := xsoar.NewClient(
			xsoar.WithBaseURL("https://api.xsoar.example.com"),
			xsoar.WithAPIKey("key-id", "api-key"),
			xsoar.WithTimeout(60*time.Second),
		)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("success with custom HTTP client", func(t *testing.T) {
		customClient := &http.Client{
			Timeout: 90 * time.Second,
		}
		client, err := xsoar.NewClient(
			xsoar.WithBaseURL("https://api.xsoar.example.com"),
			xsoar.WithAPIKey("key-id", "api-key"),
			xsoar.WithHTTPClient(customClient),
		)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}
