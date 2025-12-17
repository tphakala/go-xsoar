// Package xsoar provides a native Go client for the Palo Alto Networks
// Cortex XSOAR 8.x / XSIAM REST API.
//
// # Features
//
//   - Service-based architecture for expandability
//   - Modern Go 1.25+ iterators for pagination
//   - Typed errors for precise error handling
//   - Functional options for flexible configuration
//   - No runtime dependencies (test dependencies only)
//
// # Quick Start
//
//	client, err := xsoar.NewClient(
//	    xsoar.WithBaseURL("https://api-tenant.xdr.us.paloaltonetworks.com"),
//	    xsoar.WithAPIKey(keyID, apiKey),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Search incidents
//	filter := &xsoar.IncidentFilter{
//	    Status: []xsoar.IncidentStatus{xsoar.StatusActive},
//	}
//
//	for incident, err := range client.Incidents.Search(ctx, filter) {
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Printf("Incident: %s (%s)\n", incident.Name, incident.Severity)
//	}
//
// # Error Handling
//
// The package uses typed errors that can be inspected with errors.As:
//
//	incident, err := client.Incidents.Get(ctx, "invalid-id")
//	if err != nil {
//	    var notFound *xsoar.NotFoundError
//	    if errors.As(err, &notFound) {
//	        // Handle not found
//	    }
//	}
//
// # Pagination
//
// Use iterators for automatic pagination:
//
//	// Iterate over all results
//	for incident, err := range client.Incidents.Search(ctx, filter) {
//	    // ...
//	}
//
//	// Collect all results into a slice
//	incidents, err := xsoar.Collect(client.Incidents.Search(ctx, filter))
//
//	// Or use manual pagination
//	page, err := client.Incidents.SearchPage(ctx, filter, &xsoar.PageOptions{
//	    Offset: 0,
//	    Limit:  100,
//	})
package xsoar
