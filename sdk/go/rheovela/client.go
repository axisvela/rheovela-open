// Package rheovela is the official Go client for the Rheovela HTTP API
// (/api/v1, see api/openapi.yaml). It covers instance lifecycle (open run,
// steps, suspend/resume, compensate, audit) and the work-item broker
// (claim/heartbeat/complete/fail).
//
// The client is transport-only and has no dependency on any consumer; provide
// a CredentialSource to attach a bearer token, or leave it unset for a
// trusted local endpoint.
package rheovela

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Instance is a durable process instance.
type Instance struct {
	InstanceID        string            `json:"instance_id"`
	ProjectID         string            `json:"project_id"`
	DefinitionID      string            `json:"definition_id"`
	DefinitionVersion int               `json:"definition_version"`
	Status            string            `json:"status"`
	OpenedBy          string            `json:"opened_by"`
	Variables         map[string]string `json:"variables"`
	ParentInstanceID  string            `json:"parent_instance_id,omitempty"`
	Depth             int               `json:"depth"`
	PartitionID       string            `json:"partition_id,omitempty"`
	StartedAt         time.Time         `json:"started_at"`
	ClosedAt          *time.Time        `json:"closed_at,omitempty"`
}

// WorkItem is a claimable unit of work with lease/fencing semantics.
type WorkItem struct {
	ID           string     `json:"id"`
	InstanceID   string     `json:"instance_id"`
	ActivityID   string     `json:"activity_id"`
	State        string     `json:"state"`
	Assignee     string     `json:"assignee,omitempty"`
	LeaseUntil   *time.Time `json:"lease_until,omitempty"`
	FencingToken string     `json:"fencing_token,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// OpenRunRequest opens a new instance for a workflow.
type OpenRunRequest struct {
	Workflow       string            `json:"workflow"`
	ProjectID      string            `json:"project"`
	Actor          string            `json:"actor"`
	Variables      map[string]string `json:"variables"`
	IdempotencyKey string            `json:"idempotency_key"`
}

// StepRequest mutates a stage of an instance. Action is one of
// enter|complete|fail|skip|assign.
type StepRequest struct {
	Action         string `json:"action"`
	Stage          string `json:"stage"`
	Actor          string `json:"actor"`
	AssignedTo     string `json:"assigned_to,omitempty"`
	Error          string `json:"error,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// Claim is the result of claiming a work item.
type Claim struct {
	Token      string    `json:"token"`
	Lease      string    `json:"lease"`
	LeaseUntil time.Time `json:"lease_until"`
}

const defaultLease = "30s"

// Client is a Rheovela HTTP API client.
type Client struct {
	baseURL   string
	tokens    CredentialSource
	projectID string
	hc        *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithTokenSource sets the credential source (default: no Authorization header).
func WithTokenSource(src CredentialSource) Option {
	return func(c *Client) {
		if src != nil {
			c.tokens = src
		}
	}
}

// WithScope sets the default project scope for list operations.
func WithScope(projectID string) Option { return func(c *Client) { c.projectID = projectID } }

// WithHTTPClient overrides the default HTTP client (nil is ignored).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.hc = hc
		}
	}
}

// NewClient creates a client for baseURL (default http://127.0.0.1:8080).
func NewClient(baseURL string, opts ...Option) *Client {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8080"
	}
	c := &Client{baseURL: strings.TrimRight(baseURL, "/"), hc: &http.Client{Timeout: 30 * time.Second}}
	for _, o := range opts {
		if o != nil {
			o(c)
		}
	}
	return c
}

// OpenRun opens a new instance.
func (c *Client) OpenRun(ctx context.Context, req OpenRunRequest) (*Instance, error) {
	var out Instance
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/instances", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListInstances lists instances, optionally filtered by project.
func (c *Client) ListInstances(ctx context.Context, projectID string) ([]Instance, error) {
	if projectID == "" {
		projectID = c.projectID
	}
	path := "/api/v1/instances"
	if projectID != "" {
		path += "?project=" + url.QueryEscape(projectID)
	}
	var out []Instance
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetInstance fetches one instance.
func (c *Client) GetInstance(ctx context.Context, id string) (*Instance, error) {
	var out Instance
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/instances/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Step applies a stage transition to an instance.
func (c *Client) Step(ctx context.Context, instanceID string, req StepRequest) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/instances/"+url.PathEscape(instanceID)+"/steps", req, nil)
}

// Suspend suspends an active instance.
func (c *Client) Suspend(ctx context.Context, instanceID, actor, reason string) error {
	body := map[string]string{"actor": actor, "reason": reason}
	return c.doJSON(ctx, http.MethodPost, "/api/v1/instances/"+url.PathEscape(instanceID)+"/suspend", body, nil)
}

// Resume resumes a suspended instance.
func (c *Client) Resume(ctx context.Context, instanceID, actor string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/instances/"+url.PathEscape(instanceID)+"/resume", map[string]string{"actor": actor}, nil)
}

// Compensate compensates an instance; the plan JSON is returned verbatim.
func (c *Client) Compensate(ctx context.Context, instanceID, actor string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/instances/"+url.PathEscape(instanceID)+"/compensate", map[string]string{"actor": actor}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Audit returns the verified audit chain JSON for an instance.
func (c *Client) Audit(ctx context.Context, instanceID string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/audit/"+url.PathEscape(instanceID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListWorkItems lists work items, optionally filtered by instance.
func (c *Client) ListWorkItems(ctx context.Context, instanceID string) ([]WorkItem, error) {
	path := "/api/v1/work-items"
	if instanceID != "" {
		path += "?instance=" + url.QueryEscape(instanceID)
	}
	var out []WorkItem
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ClaimWorkItem claims a work item, returning its fencing token and lease.
func (c *Client) ClaimWorkItem(ctx context.Context, id, lease string) (*Claim, error) {
	if lease == "" {
		lease = defaultLease
	}
	var out Claim
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/work-items/"+url.PathEscape(id)+"/claim", map[string]string{"lease": lease}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// HeartbeatWorkItem renews a claim.
func (c *Client) HeartbeatWorkItem(ctx context.Context, id, token, lease string) error {
	if lease == "" {
		lease = defaultLease
	}
	return c.doJSON(ctx, http.MethodPost, "/api/v1/work-items/"+url.PathEscape(id)+"/heartbeat", map[string]string{"token": token, "lease": lease}, nil)
}

// CompleteWorkItem completes a claimed work item.
func (c *Client) CompleteWorkItem(ctx context.Context, id, token string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/work-items/"+url.PathEscape(id)+"/complete", map[string]string{"token": token}, nil)
}

// FailWorkItem fails a claimed work item.
func (c *Client) FailWorkItem(ctx context.Context, id, token, reason string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/work-items/"+url.PathEscape(id)+"/fail", map[string]string{"token": token, "error": reason}, nil)
}

// Health checks the service is reachable.
func (c *Client) Health(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodGet, "/api/v1/health", nil, nil)
}

// doJSON performs a request, decoding a JSON response into out when non-nil.
func (c *Client) doJSON(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("rheovela: encode: %w", err)
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("rheovela: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.tokens != nil {
		tok, err := c.tokens.Token(ctx)
		if err != nil {
			return fmt.Errorf("rheovela: credential: %w", err)
		}
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("rheovela: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("rheovela: read: %w", err)
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("rheovela: http %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("rheovela: decode: %w", err)
		}
	}
	return nil
}
