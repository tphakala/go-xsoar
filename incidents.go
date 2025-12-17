package xsoar

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/url"

	"github.com/tphakala/go-xsoar/internal/api"
)

const (
	defaultPageSize = 100
	maxPageSize     = 1000
)

// IncidentService provides operations on XSOAR incidents.
//
//go:generate mockery --name=IncidentService --output=mocks --outpkg=mocks --filename=incident_service.go
type IncidentService interface {
	// Search returns an iterator over all incidents matching the filter.
	// The iterator fetches pages lazily as you iterate.
	Search(ctx context.Context, filter *IncidentFilter, opts ...RequestOption) iter.Seq2[*Incident, error]

	// SearchPage returns a single page of incidents.
	// Use this for manual pagination control.
	SearchPage(ctx context.Context, filter *IncidentFilter, page *PageOptions, opts ...RequestOption) (*IncidentPage, error)

	// Get retrieves a single incident by ID.
	Get(ctx context.Context, id string, opts ...RequestOption) (*Incident, error)

	// Create creates a new incident.
	Create(ctx context.Context, req *CreateIncidentRequest, opts ...RequestOption) (*Incident, error)

	// Update modifies an existing incident.
	Update(ctx context.Context, id string, req *UpdateIncidentRequest, opts ...RequestOption) error

	// Close closes an incident with the given reason.
	Close(ctx context.Context, id string, req *CloseIncidentRequest, opts ...RequestOption) error

	// Delete removes an incident by ID.
	Delete(ctx context.Context, id string, opts ...RequestOption) error
}

// incidentService implements IncidentService.
type incidentService struct {
	transport *api.Transport
}

func newIncidentService(transport *api.Transport) *incidentService {
	return &incidentService{transport: transport}
}

// Search returns an iterator over all incidents matching the filter.
func (s *incidentService) Search(ctx context.Context, filter *IncidentFilter, opts ...RequestOption) iter.Seq2[*Incident, error] {
	return func(yield func(*Incident, error) bool) {
		offset := 0
		pageSize := defaultPageSize

		for {
			page, err := s.SearchPage(ctx, filter, &PageOptions{
				Offset: offset,
				Limit:  pageSize,
			}, opts...)

			if err != nil {
				yield(nil, err)
				return
			}

			if !s.yieldPageItems(ctx, page, yield) {
				return
			}

			if !page.HasMore() {
				return
			}

			offset = page.NextOffset()
		}
	}
}

// yieldPageItems yields each incident from the page to the iterator.
// Returns false if iteration should stop (context cancelled or yield returned false).
func (s *incidentService) yieldPageItems(ctx context.Context, page *IncidentPage, yield func(*Incident, error) bool) bool {
	for _, incident := range page.Data {
		if err := ctx.Err(); err != nil {
			yield(nil, err)
			return false
		}
		if !yield(incident, nil) {
			return false
		}
	}
	return true
}

// SearchPage returns a single page of incidents.
func (s *incidentService) SearchPage(ctx context.Context, filter *IncidentFilter, page *PageOptions, opts ...RequestOption) (*IncidentPage, error) {
	reqCfg := newRequestConfig()
	reqCfg.apply(opts...)

	if page == nil {
		page = &PageOptions{}
	}
	if page.Limit <= 0 {
		page.Limit = defaultPageSize
	}
	if page.Limit > maxPageSize {
		page.Limit = maxPageSize
	}

	body := &searchRequest{
		Filter:      filter,
		PageOptions: *page,
	}

	var result IncidentPage
	resp, err := s.transport.DoJSON(ctx, &api.Request{
		Method:  http.MethodPost,
		Path:    "/incidents/search",
		Body:    body,
		Headers: reqCfg.headers,
	}, &result)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, parseError(resp.StatusCode, resp.Body, resp.Headers)
	}

	return &result, nil
}

// validateID checks that an incident ID is not empty.
func validateID(id string) error {
	if id == "" {
		return &ValidationError{
			APIError: APIError{Message: "incident ID cannot be empty"},
		}
	}
	return nil
}

// validateCreateRequest validates the create incident request.
func validateCreateRequest(req *CreateIncidentRequest) error {
	if req == nil {
		return &ValidationError{
			APIError: APIError{Message: "create request cannot be nil"},
		}
	}
	if req.Name == "" {
		return &ValidationError{
			APIError: APIError{Message: "incident name is required"},
		}
	}
	if req.Type == "" {
		return &ValidationError{
			APIError: APIError{Message: "incident type is required"},
		}
	}
	return nil
}

// Get retrieves a single incident by ID.
func (s *incidentService) Get(ctx context.Context, id string, opts ...RequestOption) (*Incident, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}

	reqCfg := newRequestConfig()
	reqCfg.apply(opts...)

	var result Incident
	resp, err := s.transport.DoJSON(ctx, &api.Request{
		Method:  http.MethodGet,
		Path:    fmt.Sprintf("/incident/%s", url.PathEscape(id)),
		Headers: reqCfg.headers,
	}, &result)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, &NotFoundError{
			APIError:     APIError{StatusCode: http.StatusNotFound, Message: "incident not found"},
			ResourceType: "incident",
			ResourceID:   id,
		}
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, parseError(resp.StatusCode, resp.Body, resp.Headers)
	}

	return &result, nil
}

// Create creates a new incident.
func (s *incidentService) Create(ctx context.Context, req *CreateIncidentRequest, opts ...RequestOption) (*Incident, error) {
	if err := validateCreateRequest(req); err != nil {
		return nil, err
	}

	reqCfg := newRequestConfig()
	reqCfg.apply(opts...)

	var result Incident
	resp, err := s.transport.DoJSON(ctx, &api.Request{
		Method:  http.MethodPost,
		Path:    "/incident",
		Body:    req,
		Headers: reqCfg.headers,
	}, &result)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, parseError(resp.StatusCode, resp.Body, resp.Headers)
	}

	return &result, nil
}

// Update modifies an existing incident.
func (s *incidentService) Update(ctx context.Context, id string, req *UpdateIncidentRequest, opts ...RequestOption) error {
	if err := validateID(id); err != nil {
		return err
	}

	reqCfg := newRequestConfig()
	reqCfg.apply(opts...)

	// XSOAR update requires incident ID in the body
	body := map[string]any{
		"id": id,
	}
	if req.Severity != nil {
		body["severity"] = *req.Severity
	}
	if req.Owner != nil {
		body["owner"] = *req.Owner
	}
	if req.Status != nil {
		body["status"] = *req.Status
	}
	if req.Description != nil {
		body["description"] = *req.Description
	}
	if req.CustomFields != nil {
		body["CustomFields"] = req.CustomFields
	}

	resp, err := s.transport.DoJSON(ctx, &api.Request{
		Method:  http.MethodPost,
		Path:    "/incident/update",
		Body:    body,
		Headers: reqCfg.headers,
	}, nil)

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{
			APIError:     APIError{StatusCode: http.StatusNotFound, Message: "incident not found"},
			ResourceType: "incident",
			ResourceID:   id,
		}
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return parseError(resp.StatusCode, resp.Body, resp.Headers)
	}

	return nil
}

// Close closes an incident.
func (s *incidentService) Close(ctx context.Context, id string, req *CloseIncidentRequest, opts ...RequestOption) error {
	if err := validateID(id); err != nil {
		return err
	}

	reqCfg := newRequestConfig()
	reqCfg.apply(opts...)

	body := map[string]any{
		"id":          id,
		"status":      StatusDone,
		"closeReason": req.Reason,
	}
	if req.Notes != "" {
		body["closeNotes"] = req.Notes
	}

	resp, err := s.transport.DoJSON(ctx, &api.Request{
		Method:  http.MethodPost,
		Path:    "/incident/close",
		Body:    body,
		Headers: reqCfg.headers,
	}, nil)

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{
			APIError:     APIError{StatusCode: http.StatusNotFound, Message: "incident not found"},
			ResourceType: "incident",
			ResourceID:   id,
		}
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return parseError(resp.StatusCode, resp.Body, resp.Headers)
	}

	return nil
}

// Delete removes an incident by ID.
func (s *incidentService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	if err := validateID(id); err != nil {
		return err
	}

	reqCfg := newRequestConfig()
	reqCfg.apply(opts...)

	resp, err := s.transport.DoJSON(ctx, &api.Request{
		Method:  http.MethodPost,
		Path:    "/incident/batchDelete",
		Body:    map[string]any{"ids": []string{id}},
		Headers: reqCfg.headers,
	}, nil)

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusNotFound {
		return &NotFoundError{
			APIError:     APIError{StatusCode: http.StatusNotFound, Message: "incident not found"},
			ResourceType: "incident",
			ResourceID:   id,
		}
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return parseError(resp.StatusCode, resp.Body, resp.Headers)
	}

	return nil
}
