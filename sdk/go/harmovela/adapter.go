// Package harmovela adapts Rheovela work-item lifecycle observations to
// Harmovela task events. It is transport-agnostic: callers inject a Publisher
// that carries the maps to whatever transport they use.
package harmovela

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/axisvela/harmovela/implementations/go/v3/task"
	"github.com/axisvela/rheovela-open/sdk/go/v3/rheovela"
)

// Publisher delivers a Harmovela event envelope. Implementations are
// responsible for any transport-specific encoding.
type Publisher interface {
	Publish(context.Context, map[string]any) error
}

// PublisherFunc adapts a function to the Publisher interface.
type PublisherFunc func(context.Context, map[string]any) error

// Publish calls f(ctx, event).
func (f PublisherFunc) Publish(ctx context.Context, event map[string]any) error {
	return f(ctx, event)
}

// Config configures an Observer. Source and SessionID default to "rheovela",
// Now defaults to time.Now, and TaskID defaults to the work item ID. Publisher
// is required.
type Config struct {
	Source      string
	TenantID    string
	WorkspaceID string
	SessionID   string
	Publisher   Publisher
	Now         func() time.Time
	TaskID      func(rheovela.WorkItem) string
}

// Observer maps Rheovela work-item states onto Harmovela task events.
type Observer struct {
	source      string
	tenantID    string
	workspaceID string
	sessionID   string
	publisher   Publisher
	now         func() time.Time
	taskID      func(rheovela.WorkItem) string

	mu         sync.Mutex
	tasks      map[string]*itemState
	opened     bool
	closed     bool
	sessionSeq int
}

type itemState struct {
	tracker   *task.Tracker
	lastState string
	seen      bool
}

// New creates an Observer.
func New(cfg Config) (*Observer, error) {
	if cfg.Publisher == nil {
		return nil, fmt.Errorf("harmovela: publisher is required")
	}

	source := cfg.Source
	if source == "" {
		source = "rheovela"
	}
	sessionID := cfg.SessionID
	if sessionID == "" {
		sessionID = "rheovela"
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	taskID := cfg.TaskID
	if taskID == nil {
		taskID = func(item rheovela.WorkItem) string { return item.ID }
	}

	return &Observer{
		source:      source,
		tenantID:    cfg.TenantID,
		workspaceID: cfg.WorkspaceID,
		sessionID:   sessionID,
		publisher:   cfg.Publisher,
		now:         now,
		taskID:      taskID,
		tasks:       make(map[string]*itemState),
	}, nil
}

// Open emits session.opened followed by session.ready. Repeated calls are
// no-ops.
func (o *Observer) Open(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.opened {
		return nil
	}
	if err := o.publish(ctx, o.sessionEvent("session.opened")); err != nil {
		return err
	}
	if err := o.publish(ctx, o.sessionEvent("session.ready")); err != nil {
		return err
	}
	o.opened = true
	return nil
}

// Close emits session.closed once. Repeated calls are no-ops.
func (o *Observer) Close(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.closed {
		return nil
	}
	if err := o.publish(ctx, o.sessionEvent("session.closed")); err != nil {
		return err
	}
	o.closed = true
	return nil
}

// Observe maps the current work-item state to Harmovela task events and
// publishes them. Observing the same state twice is a no-op. Invalid
// transitions return an error without publishing anything.
func (o *Observer) Observe(ctx context.Context, item rheovela.WorkItem) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	key := o.taskID(item)
	if key == "" {
		return fmt.Errorf("harmovela: work item has no task ID")
	}

	state := o.tasks[key]
	if state == nil {
		state = &itemState{
			tracker:   task.NewTracker(key, o.source, item.ActivityID),
			lastState: "none",
		}
		o.tasks[key] = state
	}

	if state.seen && state.lastState == item.State {
		return nil
	}
	if state.tracker.IsTerminal() {
		return fmt.Errorf("harmovela: task %q is terminal (%s) and cannot transition to %q", key, state.tracker.State, item.State)
	}

	events, err := o.buildEvents(state, item)
	if err != nil {
		return err
	}
	createdAt := o.now().UTC().Format(time.RFC3339)
	for _, event := range events {
		event["created_at"] = createdAt
		if err := o.publish(ctx, event); err != nil {
			state.lastState = item.State
			state.seen = true
			return err
		}
	}

	state.lastState = item.State
	state.seen = true
	return nil
}

func (o *Observer) buildEvents(state *itemState, item rheovela.WorkItem) ([]map[string]any, error) {
	tracker := state.tracker
	previous := tracker.State

	invalid := func() ([]map[string]any, error) {
		tracker.State = previous
		return nil, fmt.Errorf("harmovela: invalid transition %q from %q for task %q", item.State, previous, tracker.ID)
	}
	add := func(event map[string]any) ([]map[string]any, error) {
		if event == nil {
			return invalid()
		}
		return []map[string]any{event}, nil
	}

	switch item.State {
	case "ready":
		if previous == task.Submitted && !state.seen {
			return add(tracker.Transition("task.submitted", nil))
		}
		if previous == task.Blocked {
			return add(tracker.Transition("task.started", nil))
		}
		return invalid()

	case "claimed":
		switch previous {
		case task.Submitted:
			accepted := tracker.Transition("task.accepted", nil)
			if accepted == nil {
				return invalid()
			}
			started := tracker.Transition("task.started", nil)
			if started == nil {
				return invalid()
			}
			return []map[string]any{accepted, started}, nil
		case task.Accepted, task.Blocked:
			return add(tracker.Transition("task.started", nil))
		default:
			return invalid()
		}

	case "awaiting_approval":
		return add(tracker.Transition("task.blocked", approvalPayload(item)))

	case "done":
		return add(tracker.Completed(completedPayload(item)))

	case "failed":
		message := item.LastError
		if message == "" {
			message = "work item failed"
		}
		return add(tracker.Failed("rheovela.work_item_failed", message))

	case "cancelled":
		return add(tracker.Transition("task.cancelled", nil))

	default:
		return invalid()
	}
}

func approvalPayload(item rheovela.WorkItem) map[string]any {
	payload := map[string]any{
		"reason":         "",
		"pending_stage":  "",
		"approval_state": "",
	}
	if approval := item.Approval; approval != nil {
		payload["reason"] = approval.Reason
		payload["pending_stage"] = approval.PendingStage
		payload["approval_state"] = approval.State
	}
	return payload
}

func completedPayload(item rheovela.WorkItem) map[string]any {
	payload := make(map[string]any, 3)
	if len(item.Result) > 0 {
		payload["result"] = item.Result
	}
	if item.CheckpointRef != "" {
		payload["checkpoint_ref"] = item.CheckpointRef
	}
	if item.EffectCursor != "" {
		payload["effect_cursor"] = item.EffectCursor
	}
	return payload
}

func (o *Observer) sessionEvent(eventType string) map[string]any {
	o.sessionSeq++
	return map[string]any{
		"spec_version": "0.2",
		"id":           fmt.Sprintf("evt_session_%d", o.sessionSeq),
		"type":         eventType,
		"source":       o.source,
		"session_id":   o.sessionID,
		"created_at":   o.now().UTC().Format(time.RFC3339),
		"payload": map[string]any{
			"session_id": o.sessionID,
		},
	}
}

func (o *Observer) publish(ctx context.Context, event map[string]any) error {
	if o.tenantID != "" {
		event["tenant_id"] = o.tenantID
	}
	if o.workspaceID != "" {
		event["workspace_id"] = o.workspaceID
	}
	return o.publisher.Publish(ctx, event)
}
