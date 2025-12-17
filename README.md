# go-xsoar

[![Go Reference](https://pkg.go.dev/badge/github.com/tphakala/go-xsoar.svg)](https://pkg.go.dev/github.com/tphakala/go-xsoar)
[![CI](https://github.com/tphakala/go-xsoar/actions/workflows/ci.yaml/badge.svg)](https://github.com/tphakala/go-xsoar/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tphakala/go-xsoar)](https://goreportcard.com/report/github.com/tphakala/go-xsoar)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Native Go API client for Palo Alto Networks Cortex XSOAR 8.x / XSIAM.

## Features

- **Modern Go** - Requires Go 1.24+, uses `iter.Seq2` iterators for pagination
- **Type-safe** - Strongly typed models with `errors.As()` support for error handling
- **Service-based** - Clean API surface: `client.Incidents.Search()`, `client.Incidents.Get()`
- **Flexible** - Functional options pattern for configuration
- **Testable** - Interfaces with mockery support, injectable HTTP client

## Installation

```bash
go get github.com/tphakala/go-xsoar
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/tphakala/go-xsoar"
)

func main() {
    // Create client
    client, err := xsoar.NewClient(
        xsoar.WithBaseURL("https://api-tenant.xdr.us.paloaltonetworks.com"),
        xsoar.WithAPIKey("your-key-id", "your-api-key"),
    )
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Search incidents using iterator
    for incident, err := range client.Incidents.Search(ctx, &xsoar.IncidentFilter{
        Status: []xsoar.IncidentStatus{xsoar.StatusActive},
    }) {
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("Incident: %s - %s\n", incident.ID, incident.Name)
    }
}
```

## Authentication

XSOAR 8.x / XSIAM uses advanced API key authentication with two headers:

| Header | Value |
|--------|-------|
| `x-xdr-auth-id` | API Key ID |
| `Authorization` | API Key |

Generate API keys in XSOAR/XSIAM under **Settings > API Keys**.

## Usage

### Client Configuration

```go
client, err := xsoar.NewClient(
    xsoar.WithBaseURL("https://api-tenant.xdr.us.paloaltonetworks.com"),
    xsoar.WithAPIKey(keyID, apiKey),
    xsoar.WithTimeout(30 * time.Second),    // optional
    xsoar.WithHTTPClient(customClient),     // optional
    xsoar.WithUserAgent("my-app/1.0"),      // optional
)
```

### Searching Incidents

```go
// Using iterator (fetches pages lazily)
filter := &xsoar.IncidentFilter{
    Status:   []xsoar.IncidentStatus{xsoar.StatusActive},
    Severity: []xsoar.Severity{xsoar.SeverityHigh, xsoar.SeverityCritical},
}

for incident, err := range client.Incidents.Search(ctx, filter) {
    if err != nil {
        return err
    }
    process(incident)
}

// Collect all results into a slice
incidents, err := xsoar.Collect(client.Incidents.Search(ctx, filter))

// Get first N results
incidents, err := xsoar.CollectN(client.Incidents.Search(ctx, filter), 10)

// Get first result only
incident, err := xsoar.First(client.Incidents.Search(ctx, filter))

// Low-level pagination control
page, err := client.Incidents.SearchPage(ctx, filter, &xsoar.PageOptions{
    Offset: 0,
    Limit:  100,
})
```

### CRUD Operations

```go
// Get incident by ID
incident, err := client.Incidents.Get(ctx, "inc-123")

// Create incident
incident, err := client.Incidents.Create(ctx, &xsoar.CreateIncidentRequest{
    Name:     "Security Alert",
    Type:     "Malware",
    Severity: xsoar.SeverityHigh,
    CustomFields: map[string]any{
        "source": "SIEM",
    },
})

// Update incident
severity := xsoar.SeverityCritical
owner := "analyst@company.com"
err := client.Incidents.Update(ctx, "inc-123", &xsoar.UpdateIncidentRequest{
    Severity: &severity,
    Owner:    &owner,
})

// Close incident
err := client.Incidents.Close(ctx, "inc-123", &xsoar.CloseIncidentRequest{
    Reason: "Resolved",
    Notes:  "False positive confirmed",
})

// Delete incident
err := client.Incidents.Delete(ctx, "inc-123")
```

### Per-Request Options

```go
incident, err := client.Incidents.Get(ctx, "inc-123",
    xsoar.WithRequestID("trace-abc-123"),
    xsoar.WithHeader("X-Custom-Header", "value"),
)
```

## Error Handling

All errors implement the standard `error` interface and can be inspected using `errors.As()`:

```go
incident, err := client.Incidents.Get(ctx, "inc-123")
if err != nil {
    var authErr *xsoar.AuthenticationError
    var notFoundErr *xsoar.NotFoundError
    var validationErr *xsoar.ValidationError
    var rateLimitErr *xsoar.RateLimitError
    var serverErr *xsoar.ServerError

    switch {
    case errors.As(err, &authErr):
        log.Fatal("Invalid credentials")
    case errors.As(err, &notFoundErr):
        log.Printf("Incident %s not found", notFoundErr.ResourceID)
    case errors.As(err, &validationErr):
        log.Printf("Validation error: %s", validationErr.Message)
    case errors.As(err, &rateLimitErr):
        log.Printf("Rate limited, retry after %s", rateLimitErr.RetryAfter)
    case errors.As(err, &serverErr):
        log.Printf("Server error: %s", serverErr.Message)
    default:
        log.Printf("Error: %v", err)
    }
}
```

## Models

### Severity Levels

| Constant | Value | String |
|----------|-------|--------|
| `SeverityUnknown` | 0 | "Unknown" |
| `SeverityInfo` | 1 | "Info" |
| `SeverityLow` | 2 | "Low" |
| `SeverityMedium` | 3 | "Medium" |
| `SeverityHigh` | 4 | "High" |
| `SeverityCritical` | 5 | "Critical" |

### Incident Status

| Constant | Value |
|----------|-------|
| `StatusActive` | "Active" |
| `StatusPending` | "Pending" |
| `StatusDone` | "Done" |
| `StatusArchived` | "Archive" |

## Iterator Helpers

The package provides helper functions for working with `iter.Seq2[T, error]` iterators:

```go
// Collect all items
items, err := xsoar.Collect(iterator)

// Collect up to N items
items, err := xsoar.CollectN(iterator, 10)

// Get first item (returns ErrEmptyIterator if empty)
item, err := xsoar.First(iterator)

// Take first N items (returns new iterator)
limited := xsoar.Take(iterator, 5)

// Filter items
filtered := xsoar.Filter(iterator, func(i *xsoar.Incident) bool {
    return i.Severity >= xsoar.SeverityHigh
})

// Transform items
mapped := xsoar.Map(iterator, func(i *xsoar.Incident) string {
    return i.ID
})
```

## Testing

The package provides interfaces that can be mocked using [mockery](https://github.com/vektra/mockery):

```bash
go generate ./...
```

Example test:

```go
func TestMyService(t *testing.T) {
    mockIncidents := mocks.NewIncidentService(t)
    mockIncidents.On("Get", mock.Anything, "inc-123", mock.Anything).
        Return(&xsoar.Incident{ID: "inc-123", Name: "Test"}, nil)

    // Use mockIncidents in your tests
}
```

## Requirements

- Go 1.24 or later
- XSOAR 8.x or XSIAM (legacy XSOAR versions not supported)

## License

MIT
