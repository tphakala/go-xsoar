// Package xsoar provides a Go client for the Palo Alto Networks Cortex XSOAR 8.x / XSIAM API.
//
// Basic usage:
//
//	client, err := xsoar.NewClient(
//	    xsoar.WithBaseURL("https://api-tenant.xdr.us.paloaltonetworks.com"),
//	    xsoar.WithAPIKey(keyID, apiKey),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Search incidents using iterator
//	for incident, err := range client.Incidents.Search(ctx, filter) {
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Println(incident.Name)
//	}
package xsoar

import (
	"net/http"
	"time"

	"github.com/tphakala/go-xsoar/internal/api"
	"github.com/tphakala/go-xsoar/internal/auth"
)

// Default configuration values.
const defaultTimeout = 30 * time.Second

// Client is the XSOAR API client.
type Client struct {
	// Incidents provides access to incident operations.
	Incidents IncidentService

	transport *api.Transport
}

// NewClient creates a new XSOAR client with the given options.
func NewClient(opts ...ClientOption) (*Client, error) {
	cfg := &clientConfig{
		timeout: defaultTimeout,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.baseURL == "" {
		return nil, ErrNoBaseURL
	}

	if cfg.keyID == "" || cfg.apiKey == "" {
		return nil, ErrNoCredentials
	}

	creds := &auth.Credentials{
		KeyID:  cfg.keyID,
		APIKey: cfg.apiKey,
	}

	httpClient := cfg.httpClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: cfg.timeout,
		}
	}

	transport, err := api.NewTransport(cfg.baseURL, creds, httpClient)
	if err != nil {
		return nil, err
	}

	if cfg.userAgent != "" {
		transport.UserAgent = cfg.userAgent
	}

	client := &Client{
		transport: transport,
	}

	// Initialize services
	client.Incidents = newIncidentService(transport)

	return client, nil
}

// BaseURL returns the configured API base URL.
func (c *Client) BaseURL() string {
	return c.transport.BaseURL.String()
}
