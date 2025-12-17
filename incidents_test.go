package xsoar_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user/go-xsoar"
)

func setupTestServer(t *testing.T, handler http.HandlerFunc) *xsoar.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := xsoar.NewClient(
		xsoar.WithBaseURL(server.URL),
		xsoar.WithAPIKey("test-key-id", "test-api-key"),
	)
	require.NoError(t, err)

	return client
}

func TestIncidentService_SearchPage(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/incidents/search", r.URL.Path)
			assert.Equal(t, "test-key-id", r.Header.Get("x-xdr-auth-id"))
			assert.Equal(t, "test-api-key", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			response := xsoar.IncidentPage{
				Data: []*xsoar.Incident{
					{ID: "inc-1", Name: "Test Incident 1", Severity: xsoar.SeverityHigh},
					{ID: "inc-2", Name: "Test Incident 2", Severity: xsoar.SeverityMedium},
				},
				Total:  2,
				Offset: 0,
			}
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		page, err := client.Incidents.SearchPage(ctx, nil, &xsoar.PageOptions{Limit: 100})
		require.NoError(t, err)

		assert.Len(t, page.Data, 2)
		assert.Equal(t, "inc-1", page.Data[0].ID)
		assert.Equal(t, xsoar.SeverityHigh, page.Data[0].Severity)
		assert.Equal(t, 2, page.Total)
		assert.False(t, page.HasMore())
	})

	t.Run("with filter", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			var reqBody map[string]any
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)

			filter, ok := reqBody["filter"].(map[string]any)
			assert.True(t, ok, "filter should be a map")
			assert.Contains(t, filter, "status")

			response := xsoar.IncidentPage{Data: []*xsoar.Incident{}, Total: 0}
			err = json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		filter := &xsoar.IncidentFilter{
			Status: []xsoar.IncidentStatus{xsoar.StatusActive},
		}
		_, err := client.Incidents.SearchPage(ctx, filter, nil)
		require.NoError(t, err)
	})

	t.Run("authentication error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, err := w.Write([]byte("invalid credentials"))
			assert.NoError(t, err)
		})

		ctx := context.Background()
		_, err := client.Incidents.SearchPage(ctx, nil, nil)
		require.Error(t, err)

		var authErr *xsoar.AuthenticationError
		require.ErrorAs(t, err, &authErr)
	})

	t.Run("server error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, err := w.Write([]byte("internal error"))
			assert.NoError(t, err)
		})

		ctx := context.Background()
		_, err := client.Incidents.SearchPage(ctx, nil, nil)
		require.Error(t, err)

		var serverErr *xsoar.ServerError
		require.ErrorAs(t, err, &serverErr)
	})
}

func TestIncidentService_Search(t *testing.T) {
	t.Run("iterates all pages", func(t *testing.T) {
		callCount := 0
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++

			var reqBody map[string]any
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)

			fromIndex, ok := reqBody["fromIndex"].(float64)
			assert.True(t, ok, "fromIndex should be a number")
			offset := int(fromIndex)

			var response xsoar.IncidentPage
			switch offset {
			case 0:
				response = xsoar.IncidentPage{
					Data:   []*xsoar.Incident{{ID: "inc-1"}, {ID: "inc-2"}},
					Total:  5,
					Offset: 0,
				}
			case 2:
				response = xsoar.IncidentPage{
					Data:   []*xsoar.Incident{{ID: "inc-3"}, {ID: "inc-4"}},
					Total:  5,
					Offset: 2,
				}
			case 4:
				response = xsoar.IncidentPage{
					Data:   []*xsoar.Incident{{ID: "inc-5"}},
					Total:  5,
					Offset: 4,
				}
			}
			err = json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		incidents, err := xsoar.Collect(client.Incidents.Search(ctx, nil))
		require.NoError(t, err)

		assert.Len(t, incidents, 5)
		assert.Equal(t, "inc-1", incidents[0].ID)
		assert.Equal(t, "inc-5", incidents[4].ID)
		assert.Equal(t, 3, callCount)
	})

	t.Run("stops on error", func(t *testing.T) {
		callCount := 0
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			response := xsoar.IncidentPage{
				Data:   []*xsoar.Incident{{ID: "inc-1"}},
				Total:  10,
				Offset: 0,
			}
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		incidents, err := xsoar.Collect(client.Incidents.Search(ctx, nil))
		require.Error(t, err)

		assert.Len(t, incidents, 1)
	})

	t.Run("respects context cancellation between items", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			response := xsoar.IncidentPage{
				Data:   []*xsoar.Incident{{ID: "inc-1"}, {ID: "inc-2"}, {ID: "inc-3"}},
				Total:  3,
				Offset: 0,
			}
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // Ensure cancel is always called

		var incidents []*xsoar.Incident
		var iterErr error

		for incident, err := range client.Incidents.Search(ctx, nil) {
			if err != nil {
				iterErr = err
				break
			}
			incidents = append(incidents, incident)
			if len(incidents) == 1 {
				cancel() // Cancel after receiving first incident
			}
		}

		require.Error(t, iterErr)
		require.ErrorIs(t, iterErr, context.Canceled)
		assert.Len(t, incidents, 1)
	})
}

func TestIncidentService_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/incident/inc-123", r.URL.Path)

			incident := xsoar.Incident{
				ID:       "inc-123",
				Name:     "Test Incident",
				Status:   xsoar.StatusActive,
				Severity: xsoar.SeverityCritical,
			}
			err := json.NewEncoder(w).Encode(incident)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		incident, err := client.Incidents.Get(ctx, "inc-123")
		require.NoError(t, err)

		assert.Equal(t, "inc-123", incident.ID)
		assert.Equal(t, xsoar.SeverityCritical, incident.Severity)
	})

	t.Run("not found", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		ctx := context.Background()
		_, err := client.Incidents.Get(ctx, "nonexistent")
		require.Error(t, err)

		var notFoundErr *xsoar.NotFoundError
		require.ErrorAs(t, err, &notFoundErr)
		assert.Equal(t, "incident", notFoundErr.ResourceType)
		assert.Equal(t, "nonexistent", notFoundErr.ResourceID)
	})

	t.Run("empty ID returns validation error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not make API call with empty ID")
		})

		ctx := context.Background()
		_, err := client.Incidents.Get(ctx, "")
		require.Error(t, err)

		var validationErr *xsoar.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})
}

func TestIncidentService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/incident", r.URL.Path)

			var reqBody xsoar.CreateIncidentRequest
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, "New Incident", reqBody.Name)
			assert.Equal(t, "Malware", reqBody.Type)

			response := xsoar.Incident{
				ID:       "inc-new",
				Name:     reqBody.Name,
				Type:     reqBody.Type,
				Status:   xsoar.StatusActive,
				Severity: reqBody.Severity,
				Created:  time.Now(),
			}
			w.WriteHeader(http.StatusCreated)
			err = json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		incident, err := client.Incidents.Create(ctx, &xsoar.CreateIncidentRequest{
			Name:     "New Incident",
			Type:     "Malware",
			Severity: xsoar.SeverityHigh,
		})
		require.NoError(t, err)

		assert.Equal(t, "inc-new", incident.ID)
		assert.Equal(t, "New Incident", incident.Name)
	})

	t.Run("validation error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte("name is required"))
			assert.NoError(t, err)
		})

		ctx := context.Background()
		_, err := client.Incidents.Create(ctx, &xsoar.CreateIncidentRequest{})
		require.Error(t, err)

		var validationErr *xsoar.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})

	t.Run("nil request returns validation error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not make API call with nil request")
		})

		ctx := context.Background()
		_, err := client.Incidents.Create(ctx, nil)
		require.Error(t, err)

		var validationErr *xsoar.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})

	t.Run("missing name returns validation error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not make API call with empty name")
		})

		ctx := context.Background()
		_, err := client.Incidents.Create(ctx, &xsoar.CreateIncidentRequest{
			Type: "Malware",
		})
		require.Error(t, err)

		var validationErr *xsoar.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})

	t.Run("missing type returns validation error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not make API call with empty type")
		})

		ctx := context.Background()
		_, err := client.Incidents.Create(ctx, &xsoar.CreateIncidentRequest{
			Name: "Test Incident",
		})
		require.Error(t, err)

		var validationErr *xsoar.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})
}

func TestIncidentService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/incident/update", r.URL.Path)

			var reqBody map[string]any
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, "inc-123", reqBody["id"])

			severity, ok := reqBody["severity"].(float64)
			assert.True(t, ok, "severity should be a number")
			assert.InDelta(t, float64(5), severity, 0.001)

			w.WriteHeader(http.StatusOK)
		})

		ctx := context.Background()
		severity := xsoar.SeverityCritical
		err := client.Incidents.Update(ctx, "inc-123", &xsoar.UpdateIncidentRequest{
			Severity: &severity,
		})
		require.NoError(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		ctx := context.Background()
		err := client.Incidents.Update(ctx, "nonexistent", &xsoar.UpdateIncidentRequest{})
		require.Error(t, err)

		var notFoundErr *xsoar.NotFoundError
		require.ErrorAs(t, err, &notFoundErr)
	})

	t.Run("empty ID returns validation error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not make API call with empty ID")
		})

		ctx := context.Background()
		err := client.Incidents.Update(ctx, "", &xsoar.UpdateIncidentRequest{})
		require.Error(t, err)

		var validationErr *xsoar.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})
}

func TestIncidentService_Close(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/incident/close", r.URL.Path)

			var reqBody map[string]any
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)
			assert.Equal(t, "inc-123", reqBody["id"])
			assert.Equal(t, "Resolved", reqBody["closeReason"])

			w.WriteHeader(http.StatusOK)
		})

		ctx := context.Background()
		err := client.Incidents.Close(ctx, "inc-123", &xsoar.CloseIncidentRequest{
			Reason: "Resolved",
			Notes:  "Issue has been resolved",
		})
		require.NoError(t, err)
	})

	t.Run("empty ID returns validation error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not make API call with empty ID")
		})

		ctx := context.Background()
		err := client.Incidents.Close(ctx, "", &xsoar.CloseIncidentRequest{Reason: "Test"})
		require.Error(t, err)

		var validationErr *xsoar.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})
}

func TestIncidentService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/incident/batchDelete", r.URL.Path)

			var reqBody map[string]any
			err := json.NewDecoder(r.Body).Decode(&reqBody)
			assert.NoError(t, err)

			ids, ok := reqBody["ids"].([]any)
			assert.True(t, ok, "ids should be an array")
			assert.Contains(t, ids, "inc-123")

			w.WriteHeader(http.StatusOK)
		})

		ctx := context.Background()
		err := client.Incidents.Delete(ctx, "inc-123")
		require.NoError(t, err)
	})

	t.Run("empty ID returns validation error", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("should not make API call with empty ID")
		})

		ctx := context.Background()
		err := client.Incidents.Delete(ctx, "")
		require.Error(t, err)

		var validationErr *xsoar.ValidationError
		require.ErrorAs(t, err, &validationErr)
	})
}

func TestIncidentService_WithRequestOptions(t *testing.T) {
	t.Run("custom headers", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "test-request-123", r.Header.Get("X-Request-ID"))
			assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))

			response := xsoar.IncidentPage{Data: []*xsoar.Incident{}, Total: 0}
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		_, err := client.Incidents.SearchPage(ctx, nil, nil,
			xsoar.WithRequestID("test-request-123"),
			xsoar.WithHeader("X-Custom-Header", "custom-value"),
		)
		require.NoError(t, err)
	})
}

func TestResponseSizeLimit(t *testing.T) {
	t.Run("rejects response exceeding size limit", func(t *testing.T) {
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			// Write a response larger than the default limit (10MB)
			// We'll write just over 10MB of data
			largeData := make([]byte, 11*1024*1024) // 11MB
			for i := range largeData {
				largeData[i] = 'x'
			}
			_, err := w.Write(largeData)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		_, err := client.Incidents.Get(ctx, "test-id")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "response too large")
	})
}

func TestURLEncoding(t *testing.T) {
	t.Run("properly encodes special characters in incident ID", func(t *testing.T) {
		var receivedRawPath string
		client := setupTestServer(t, func(w http.ResponseWriter, r *http.Request) {
			// Use RawPath which preserves percent-encoding, or RequestURI
			receivedRawPath = r.URL.EscapedPath()
			incident := xsoar.Incident{ID: "inc/test?id=123"}
			err := json.NewEncoder(w).Encode(incident)
			assert.NoError(t, err)
		})

		ctx := context.Background()
		_, err := client.Incidents.Get(ctx, "inc/test?id=123")
		require.NoError(t, err)

		// The path should be properly encoded - / becomes %2F, ? becomes %3F
		assert.Equal(t, "/incident/inc%2Ftest%3Fid=123", receivedRawPath)
	})
}
