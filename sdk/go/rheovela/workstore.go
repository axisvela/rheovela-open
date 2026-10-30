package rheovela

import (
	"context"
	"time"

	"github.com/axisvela/rheovela-open/sdk/go/v3/worker"
)

// HTTPWorkStore adapts the Rheovela client to the worker SDK's WorkStore port,
// so worker.New can drive a worker loop over the Rheovela HTTP API.
type HTTPWorkStore struct {
	c   *Client
	ctx context.Context
}

var (
	_ worker.WorkStore       = (*HTTPWorkStore)(nil)
	_ worker.ClaimGuardStore = (*HTTPWorkStore)(nil)
	_ worker.ResultStore     = (*HTTPWorkStore)(nil)
	_ worker.ApprovalStore   = (*HTTPWorkStore)(nil)
)

// NewHTTPWorkStore builds a worker WorkStore over the given client.
func NewHTTPWorkStore(c *Client) *HTTPWorkStore {
	return NewHTTPWorkStoreWithContext(context.Background(), c)
}

// NewHTTPWorkStoreWithContext builds a worker WorkStore that uses ctx for all
// client calls.
func NewHTTPWorkStoreWithContext(ctx context.Context, c *Client) *HTTPWorkStore {
	return &HTTPWorkStore{c: c, ctx: ctx}
}

// PollReady lists claimable work items globally. The global endpoint returns
// items that can currently be claimed: ready items plus claimed items whose
// lease has expired. Items still awaiting approval or in a terminal state are
// skipped if the server ever includes them.
func (s *HTTPWorkStore) PollReady() ([]worker.WorkItem, error) {
	items, err := s.c.ListWorkItems(s.ctx, "")
	if err != nil {
		return nil, err
	}
	out := make([]worker.WorkItem, 0, len(items))
	for _, it := range items {
		if it.State != "ready" && it.State != "claimed" {
			continue
		}
		out = append(out, toWorkerItem(it))
	}
	return out, nil
}

func toWorkerItem(it WorkItem) worker.WorkItem {
	w := worker.WorkItem{
		ID: it.ID, InstanceID: it.InstanceID, ActivityID: it.ActivityID, State: it.State,
		Assignee: it.Assignee, Kind: it.Kind, Payload: it.Payload,
		RequiredEvidence: it.RequiredEvidence, Evidence: it.Evidence,
		Budget: it.Budget, Spent: it.Spent, ToolScope: it.ToolScope,
		CredentialRef: it.CredentialRef, ApprovalRoles: it.ApprovalRoles,
		CompletionMode: it.CompletionMode, RequiredApprovals: it.RequiredApprovals,
		RequiredCompletions: it.RequiredCompletions, Deadline: it.Deadline,
		Escalation: it.Escalation,
		Result:     it.Result, CheckpointRef: it.CheckpointRef, EffectCursor: it.EffectCursor,
	}
	if it.Approval != nil {
		w.Approval = &worker.Approval{
			State: it.Approval.State, PendingStage: it.Approval.PendingStage,
			FinalArgs: it.Approval.FinalArgs, Reason: it.Approval.Reason,
			RequestedBy: it.Approval.RequestedBy, RequestedAt: it.Approval.RequestedAt,
			DecidedBy: it.Approval.DecidedBy, DecidedAt: it.Approval.DecidedAt,
			Comment:       it.Approval.Comment,
			CheckpointRef: it.Approval.CheckpointRef, EffectCursor: it.Approval.EffectCursor,
		}
	}
	return w
}

// Claim claims a work item and returns its fencing token.
func (s *HTTPWorkStore) Claim(id string, lease time.Duration) (string, error) {
	claim, err := s.c.ClaimWorkItem(s.ctx, id, lease.String())
	if err != nil {
		return "", err
	}
	return claim.Token, nil
}

// ClaimWithOptions claims with actor/tools/credential guards.
func (s *HTTPWorkStore) ClaimWithOptions(id string, lease time.Duration, opts worker.ClaimOptions) (string, error) {
	claim, err := s.c.ClaimWorkItemWithOptions(s.ctx, id, lease.String(), ClaimOptions{
		Actor: opts.Actor, Tools: opts.Tools, Credential: opts.Credential,
	})
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

// CompleteWithResult completes a claimed work item with a result payload and
// checkpoint/effect references.
func (s *HTTPWorkStore) CompleteWithResult(id, token string, res worker.WorkResult) error {
	return s.c.CompleteWorkItemWithResult(s.ctx, id, token, WorkResult{
		Result: res.Result, Evidence: res.Evidence, Cost: res.Cost,
		CheckpointRef: res.CheckpointRef, EffectCursor: res.EffectCursor,
	})
}

// RequestApproval asks for human approval before a work item advances.
func (s *HTTPWorkStore) RequestApproval(id, token string, req worker.ApprovalRequest) error {
	return s.c.RequestWorkItemApproval(s.ctx, id, token, ApprovalRequest{
		Actor: req.Actor, Reason: req.Reason, FinalArgs: req.FinalArgs,
		CheckpointRef: req.CheckpointRef, EffectCursor: req.EffectCursor,
	})
}

// Fail fails a claimed work item.
func (s *HTTPWorkStore) Fail(id, token, errMsg string) error {
	return s.c.FailWorkItem(s.ctx, id, token, errMsg)
}
