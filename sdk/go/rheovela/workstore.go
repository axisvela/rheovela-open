package rheovela

import (
	"context"
	"time"

	"github.com/axisrobo/rheovela-open/sdk/go/worker"
)

// HTTPWorkStore adapts the Rheovela client to the worker SDK's WorkStore port,
// so worker.New can drive a worker loop over the Rheovela HTTP API.
type HTTPWorkStore struct {
	c   *Client
	ctx context.Context
}

var _ worker.WorkStore = (*HTTPWorkStore)(nil)

// NewHTTPWorkStore builds a worker WorkStore over the given client.
func NewHTTPWorkStore(c *Client) *HTTPWorkStore {
	return &HTTPWorkStore{c: c, ctx: context.Background()}
}

// PollReady lists work items currently known to the service.
func (s *HTTPWorkStore) PollReady() ([]worker.WorkItem, error) {
	items, err := s.c.ListWorkItems(s.ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]worker.WorkItem, 0, len(items))
	for _, it := range items {
		out = append(out, worker.WorkItem{ID: it.ID, InstanceID: it.InstanceID, ActivityID: it.ActivityID, State: it.State})
	}
	return out, nil
}

// Claim claims a work item and returns its fencing token.
func (s *HTTPWorkStore) Claim(id string, lease time.Duration) (string, error) {
	claim, err := s.c.ClaimWorkItem(s.ctx, id, lease.String())
	if err != nil {
		return "", err
	}
	return claim.Token, nil
}

// Heartbeat renews a claim.
func (s *HTTPWorkStore) Heartbeat(id, token string, lease time.Duration) error {
	return s.c.HeartbeatWorkItem(s.ctx, id, token, lease.String())
}

// Complete completes a claimed work item.
func (s *HTTPWorkStore) Complete(id, token string) error {
	return s.c.CompleteWorkItem(s.ctx, id, token)
}

// Fail fails a claimed work item.
func (s *HTTPWorkStore) Fail(id, token, errMsg string) error {
	return s.c.FailWorkItem(s.ctx, id, token, errMsg)
}
