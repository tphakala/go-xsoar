// Package auth provides XSOAR 8.x / XSIAM authentication.
package auth

import "net/http"

// Credentials holds XSOAR 8.x API authentication credentials.
type Credentials struct {
	KeyID  string
	APIKey string
}

// Apply adds authentication headers to an HTTP request.
func (c *Credentials) Apply(req *http.Request) {
	if c == nil {
		return
	}
	req.Header.Set("x-xdr-auth-id", c.KeyID)
	req.Header.Set("Authorization", c.APIKey)
}

// Valid reports whether credentials are configured.
func (c *Credentials) Valid() bool {
	return c != nil && c.KeyID != "" && c.APIKey != ""
}
