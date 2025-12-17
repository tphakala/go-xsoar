package xsoar

import (
	"encoding/json"
	"time"
)

// Severity represents incident severity levels.
type Severity int

const (
	SeverityUnknown  Severity = 0
	SeverityInfo     Severity = 1
	SeverityLow      Severity = 2
	SeverityMedium   Severity = 3
	SeverityHigh     Severity = 4
	SeverityCritical Severity = 5
)

func (s *Severity) String() string {
	if s == nil {
		return "Unknown"
	}
	switch *s {
	case SeverityInfo:
		return "Info"
	case SeverityLow:
		return "Low"
	case SeverityMedium:
		return "Medium"
	case SeverityHigh:
		return "High"
	case SeverityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// MarshalJSON implements json.Marshaler.
func (s *Severity) MarshalJSON() ([]byte, error) {
	if s == nil {
		return []byte("null"), nil
	}
	return json.Marshal(int(*s))
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *Severity) UnmarshalJSON(data []byte) error {
	var v int
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*s = Severity(v)
	return nil
}

// IncidentStatus represents the status of an incident.
type IncidentStatus string

const (
	StatusActive   IncidentStatus = "Active"
	StatusPending  IncidentStatus = "Pending"
	StatusDone     IncidentStatus = "Done"
	StatusArchived IncidentStatus = "Archive"
)

// Label represents an incident label.
type Label struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Incident represents an XSOAR incident.
type Incident struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	Status        IncidentStatus `json:"status"`
	Severity      Severity       `json:"severity"`
	Owner         string         `json:"owner,omitempty"`
	Description   string         `json:"description,omitempty"`
	Phase         string         `json:"phase,omitempty"`
	PlaybookID    string         `json:"playbookId,omitempty"`
	InvestigateID string         `json:"investigationId,omitempty"`

	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Closed   time.Time `json:"closed,omitzero"` // Go 1.24+: omit when zero

	Labels []Label `json:"labels,omitempty"`

	// CustomFields holds customer-defined incident fields.
	CustomFields map[string]any `json:"CustomFields,omitempty"`

	// RawData captures any fields not explicitly modeled.
	RawData map[string]any `json:"-"`
}

// IncidentFilter defines search criteria for incidents.
type IncidentFilter struct {
	// Query is a Lucene-style query string.
	Query string `json:"query,omitempty"`

	// Status filters by incident status.
	Status []IncidentStatus `json:"status,omitempty"`

	// Severity filters by severity levels.
	Severity []Severity `json:"severity,omitempty"`

	// Type filters by incident type.
	Type []string `json:"type,omitempty"`

	// Owner filters by assigned owner.
	Owner []string `json:"owner,omitempty"`

	// FromDate filters incidents created after this time.
	FromDate time.Time `json:"fromDate,omitzero"`

	// ToDate filters incidents created before this time.
	ToDate time.Time `json:"toDate,omitzero"`
}

// PageOptions configures pagination for search requests.
type PageOptions struct {
	Offset int `json:"fromIndex"`
	Limit  int `json:"size,omitempty"`
}

// IncidentPage represents a page of incident results.
type IncidentPage struct {
	Data     []*Incident `json:"data"`
	Total    int         `json:"total"`
	Offset   int         `json:"fromIndex"`
	PageSize int         `json:"size"`
}

// HasMore returns true if there are more pages available.
func (p *IncidentPage) HasMore() bool {
	return p.Offset+len(p.Data) < p.Total
}

// NextOffset returns the offset for the next page.
func (p *IncidentPage) NextOffset() int {
	return p.Offset + len(p.Data)
}

// CreateIncidentRequest contains data for creating a new incident.
type CreateIncidentRequest struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Severity     Severity          `json:"severity,omitempty"`
	Owner        string            `json:"owner,omitempty"`
	Description  string            `json:"description,omitempty"`
	Labels       []Label           `json:"labels,omitempty"`
	CustomFields map[string]any    `json:"CustomFields,omitempty"`
	CreateDate   time.Time         `json:"createDate,omitzero"`
}

// UpdateIncidentRequest contains data for updating an incident.
type UpdateIncidentRequest struct {
	Severity     *Severity         `json:"severity,omitempty"`
	Owner        *string           `json:"owner,omitempty"`
	Status       *IncidentStatus   `json:"status,omitempty"`
	Description  *string           `json:"description,omitempty"`
	CustomFields map[string]any    `json:"CustomFields,omitempty"`
}

// CloseIncidentRequest contains data for closing an incident.
type CloseIncidentRequest struct {
	Reason     string `json:"closeReason,omitempty"`
	Notes      string `json:"closeNotes,omitempty"`
	CloseDate  time.Time `json:"closeDate,omitzero"`
}

// searchRequest is the internal request format for incident search.
type searchRequest struct {
	Filter     *IncidentFilter `json:"filter,omitempty"`
	PageOptions
}
