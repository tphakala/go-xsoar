package xsoar_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tphakala/go-xsoar"
)

func TestSeverity(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		tests := []struct {
			severity xsoar.Severity
			expected string
		}{
			{xsoar.SeverityUnknown, "Unknown"},
			{xsoar.SeverityInfo, "Info"},
			{xsoar.SeverityLow, "Low"},
			{xsoar.SeverityMedium, "Medium"},
			{xsoar.SeverityHigh, "High"},
			{xsoar.SeverityCritical, "Critical"},
		}

		for _, tt := range tests {
			assert.Equal(t, tt.expected, tt.severity.String())
		}
	})

	t.Run("JSON marshaling", func(t *testing.T) {
		data, err := json.Marshal(xsoar.SeverityHigh)
		require.NoError(t, err)
		assert.Equal(t, "4", string(data))
	})

	t.Run("JSON unmarshaling", func(t *testing.T) {
		var s xsoar.Severity
		err := json.Unmarshal([]byte("3"), &s)
		require.NoError(t, err)
		assert.Equal(t, xsoar.SeverityMedium, s)
	})

	t.Run("nil pointer returns Unknown", func(t *testing.T) {
		var s *xsoar.Severity
		// Should not panic on nil pointer
		assert.Equal(t, "Unknown", s.String())
	})

	t.Run("unknown severity value returns Unknown", func(t *testing.T) {
		s := xsoar.Severity(99)
		assert.Equal(t, "Unknown", s.String())
	})
}

func TestIncidentStatus(t *testing.T) {
	assert.Equal(t, xsoar.StatusActive, xsoar.IncidentStatus("Active"))
	assert.Equal(t, xsoar.StatusPending, xsoar.IncidentStatus("Pending"))
	assert.Equal(t, xsoar.StatusDone, xsoar.IncidentStatus("Done"))
	assert.Equal(t, xsoar.StatusArchived, xsoar.IncidentStatus("Archive"))
}

func TestIncidentPage(t *testing.T) {
	t.Run("HasMore true", func(t *testing.T) {
		page := &xsoar.IncidentPage{
			Data:   make([]*xsoar.Incident, 100),
			Total:  250,
			Offset: 0,
		}
		assert.True(t, page.HasMore())
		assert.Equal(t, 100, page.NextOffset())
	})

	t.Run("HasMore false at end", func(t *testing.T) {
		page := &xsoar.IncidentPage{
			Data:   make([]*xsoar.Incident, 50),
			Total:  250,
			Offset: 200,
		}
		assert.False(t, page.HasMore())
	})

	t.Run("HasMore false exact fit", func(t *testing.T) {
		page := &xsoar.IncidentPage{
			Data:   make([]*xsoar.Incident, 100),
			Total:  100,
			Offset: 0,
		}
		assert.False(t, page.HasMore())
	})
}

func TestIncidentJSONSerialization(t *testing.T) {
	t.Run("marshal with custom fields", func(t *testing.T) {
		incident := &xsoar.Incident{
			ID:       "inc-1",
			Name:     "Test Incident",
			Type:     "Malware",
			Status:   xsoar.StatusActive,
			Severity: xsoar.SeverityHigh,
			CustomFields: map[string]any{
				"customField1": "value1",
				"customField2": 123,
			},
		}

		data, err := json.Marshal(incident)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "inc-1", result["id"])
		assert.Equal(t, "Test Incident", result["name"])
		assert.InDelta(t, float64(4), result["severity"], 0.001) // JSON numbers are float64

		customFields, ok := result["CustomFields"].(map[string]any)
		require.True(t, ok, "CustomFields should be a map")
		assert.Equal(t, "value1", customFields["customField1"])
	})

	t.Run("unmarshal from XSOAR response", func(t *testing.T) {
		jsonData := `{
			"id": "inc-123",
			"name": "Security Alert",
			"type": "Phishing",
			"status": "Active",
			"severity": 3,
			"owner": "analyst@example.com",
			"created": "2024-01-15T10:30:00Z",
			"modified": "2024-01-15T11:00:00Z",
			"labels": [{"type": "source", "value": "email"}],
			"CustomFields": {"priority": "urgent"}
		}`

		var incident xsoar.Incident
		err := json.Unmarshal([]byte(jsonData), &incident)
		require.NoError(t, err)

		assert.Equal(t, "inc-123", incident.ID)
		assert.Equal(t, "Security Alert", incident.Name)
		assert.Equal(t, xsoar.StatusActive, incident.Status)
		assert.Equal(t, xsoar.SeverityMedium, incident.Severity)
		assert.Equal(t, "analyst@example.com", incident.Owner)
		assert.Len(t, incident.Labels, 1)
		assert.Equal(t, "source", incident.Labels[0].Type)
		assert.Equal(t, "urgent", incident.CustomFields["priority"])
	})
}

func TestLabel(t *testing.T) {
	label := xsoar.Label{Type: "category", Value: "malware"}

	data, err := json.Marshal(label)
	require.NoError(t, err)

	var result xsoar.Label
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Equal(t, label, result)
}
