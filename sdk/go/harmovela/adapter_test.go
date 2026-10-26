package harmovela_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/axisvela/rheovela-open/sdk/go/v3/rheovela"

	harmovela "github.com/axisvela/rheovela-open/sdk/go/harmovela/v3"
)

type recordingPublisher struct {
	events []map[string]any
}

func (p *recordingPublisher) Publish(_ context.Context, event map[string]any) error {
	p.events = append(p.events, event)
	return nil
}

func (p *recordingPublisher) types() []string {
	out := make([]string, len(p.events))
	for i, event := range p.events {
		out[i], _ = event["type"].(string)
	}
	return out
}

func newObserver(t *testing.T, cfg harmovela.Config) *harmovela.Observer {
	t.Helper()
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	}
	obs, err := harmovela.New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return obs
}

func assertTypes(t *testing.T, pub *recordingPublisher, want ...string) {
	t.Helper()
	got := pub.types()
	if len(got) != len(want) {
		t.Fatalf("event types = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("event[%d] type = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func lastEvent(t *testing.T, pub *recordingPublisher) map[string]any {
	t.Helper()
	if len(pub.events) == 0 {
		t.Fatal("no events published")
	}
	return pub.events[len(pub.events)-1]
}

func TestObserveLifecycle(t *testing.T) {
	pub := &recordingPublisher{}
	obs := newObserver(t, harmovela.Config{Publisher: pub})
	ctx := context.Background()

	item := rheovela.WorkItem{ID: "wi_1", ActivityID: "act_1", State: "ready"}
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe ready: %v", err)
	}
	assertTypes(t, pub, "task.submitted")

	item.State = "claimed"
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe claimed: %v", err)
	}
	assertTypes(t, pub, "task.submitted", "task.accepted", "task.started")

	item.State = "done"
	item.Result = json.RawMessage(`{"answer":42}`)
	item.CheckpointRef = "cp_1"
	item.EffectCursor = "effect_1"
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe done: %v", err)
	}
	assertTypes(t, pub, "task.submitted", "task.accepted", "task.started", "task.completed")

	completed := lastEvent(t, pub)
	if completed["source"] != "rheovela" {
		t.Fatalf("source = %v, want rheovela", completed["source"])
	}
	if completed["task_id"] != "wi_1" {
		t.Fatalf("task_id = %v, want wi_1", completed["task_id"])
	}
	payload, ok := completed["payload"].(map[string]any)
	if !ok {
		t.Fatalf("payload type = %T, want map[string]any", completed["payload"])
	}
	result, ok := payload["result"].(json.RawMessage)
	if !ok {
		t.Fatalf("result type = %T, want json.RawMessage", payload["result"])
	}
	if string(result) != `{"answer":42}` {
		t.Fatalf("result = %s, want {\"answer\":42}", result)
	}
	if payload["checkpoint_ref"] != "cp_1" {
		t.Fatalf("checkpoint_ref = %v, want cp_1", payload["checkpoint_ref"])
	}
	if payload["effect_cursor"] != "effect_1" {
		t.Fatalf("effect_cursor = %v, want effect_1", payload["effect_cursor"])
	}
	if _, ok := completed["tenant_id"]; ok {
		t.Fatal("tenant_id should be omitted when not configured")
	}
	if _, ok := completed["workspace_id"]; ok {
		t.Fatal("workspace_id should be omitted when not configured")
	}
}

func TestObserveApproval(t *testing.T) {
	pub := &recordingPublisher{}
	obs := newObserver(t, harmovela.Config{Publisher: pub})
	ctx := context.Background()

	item := rheovela.WorkItem{ID: "wi_2", ActivityID: "act_2", State: "ready"}
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe ready: %v", err)
	}
	item.State = "claimed"
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe claimed: %v", err)
	}
	item.State = "awaiting_approval"
	item.Approval = &rheovela.WorkItemApproval{
		State:        "pending",
		PendingStage: "deploy",
		Reason:       "need sign-off",
	}
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe awaiting_approval: %v", err)
	}
	assertTypes(t, pub, "task.submitted", "task.accepted", "task.started", "task.blocked")

	blocked := lastEvent(t, pub)
	payload := blocked["payload"].(map[string]any)
	if payload["reason"] != "need sign-off" {
		t.Fatalf("reason = %v, want need sign-off", payload["reason"])
	}
	if payload["pending_stage"] != "deploy" {
		t.Fatalf("pending_stage = %v, want deploy", payload["pending_stage"])
	}
	if payload["approval_state"] != "pending" {
		t.Fatalf("approval_state = %v, want pending", payload["approval_state"])
	}

	item.State = "ready"
	item.Approval = &rheovela.WorkItemApproval{State: "approved"}
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe ready resume: %v", err)
	}
	assertTypes(t, pub, "task.submitted", "task.accepted", "task.started", "task.blocked", "task.started")
}

func TestObserveFailedAndCancelled(t *testing.T) {
	ctx := context.Background()

	t.Run("failed with last error", func(t *testing.T) {
		pub := &recordingPublisher{}
		obs := newObserver(t, harmovela.Config{Publisher: pub})
		item := rheovela.WorkItem{ID: "wi_fail", State: "ready"}
		if err := obs.Observe(ctx, item); err != nil {
			t.Fatalf("Observe ready: %v", err)
		}
		item.State = "failed"
		item.LastError = "boom"
		if err := obs.Observe(ctx, item); err != nil {
			t.Fatalf("Observe failed: %v", err)
		}
		assertTypes(t, pub, "task.submitted", "task.failed")
		payload := lastEvent(t, pub)["payload"].(map[string]any)
		errObj := payload["error"].(map[string]any)
		if errObj["code"] != "rheovela.work_item_failed" {
			t.Fatalf("error code = %v, want rheovela.work_item_failed", errObj["code"])
		}
		if errObj["message"] != "boom" {
			t.Fatalf("error message = %v, want boom", errObj["message"])
		}
	})

	t.Run("failed without last error", func(t *testing.T) {
		pub := &recordingPublisher{}
		obs := newObserver(t, harmovela.Config{Publisher: pub})
		item := rheovela.WorkItem{ID: "wi_fail2", State: "ready"}
		if err := obs.Observe(ctx, item); err != nil {
			t.Fatalf("Observe ready: %v", err)
		}
		item.State = "failed"
		if err := obs.Observe(ctx, item); err != nil {
			t.Fatalf("Observe failed: %v", err)
		}
		errObj := lastEvent(t, pub)["payload"].(map[string]any)["error"].(map[string]any)
		if errObj["message"] != "work item failed" {
			t.Fatalf("error message = %v, want work item failed", errObj["message"])
		}
	})

	t.Run("cancelled", func(t *testing.T) {
		pub := &recordingPublisher{}
		obs := newObserver(t, harmovela.Config{Publisher: pub})
		item := rheovela.WorkItem{ID: "wi_cancel", State: "ready"}
		if err := obs.Observe(ctx, item); err != nil {
			t.Fatalf("Observe ready: %v", err)
		}
		item.State = "cancelled"
		if err := obs.Observe(ctx, item); err != nil {
			t.Fatalf("Observe cancelled: %v", err)
		}
		assertTypes(t, pub, "task.submitted", "task.cancelled")
	})
}

func TestObserveSameStateNoop(t *testing.T) {
	pub := &recordingPublisher{}
	obs := newObserver(t, harmovela.Config{Publisher: pub})
	ctx := context.Background()

	item := rheovela.WorkItem{ID: "wi_noop", State: "ready"}
	for i := 0; i < 2; i++ {
		if err := obs.Observe(ctx, item); err != nil {
			t.Fatalf("Observe ready #%d: %v", i, err)
		}
	}
	assertTypes(t, pub, "task.submitted")

	item.State = "claimed"
	for i := 0; i < 2; i++ {
		if err := obs.Observe(ctx, item); err != nil {
			t.Fatalf("Observe claimed #%d: %v", i, err)
		}
	}
	assertTypes(t, pub, "task.submitted", "task.accepted", "task.started")
}

func TestObserveFirstDoneIsInvalid(t *testing.T) {
	pub := &recordingPublisher{}
	obs := newObserver(t, harmovela.Config{Publisher: pub})

	item := rheovela.WorkItem{ID: "wi_done", State: "done"}
	if err := obs.Observe(context.Background(), item); err == nil {
		t.Fatal("expected error when first observation is done")
	}
	if len(pub.events) != 0 {
		t.Fatalf("published %d events, want none", len(pub.events))
	}
}

func TestTerminalTaskCannotBeReopened(t *testing.T) {
	pub := &recordingPublisher{}
	obs := newObserver(t, harmovela.Config{Publisher: pub})
	ctx := context.Background()

	item := rheovela.WorkItem{ID: "wi_terminal", State: "ready"}
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe ready: %v", err)
	}
	item.State = "claimed"
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe claimed: %v", err)
	}
	item.State = "done"
	if err := obs.Observe(ctx, item); err != nil {
		t.Fatalf("Observe done: %v", err)
	}
	before := len(pub.events)
	item.State = "ready"
	if err := obs.Observe(ctx, item); err == nil {
		t.Fatal("expected error reopening a terminal task")
	}
	if len(pub.events) != before {
		t.Fatalf("published %d events after terminal reopen, want %d", len(pub.events), before)
	}
}

func TestTenantWorkspaceAndTaskIDOverride(t *testing.T) {
	pub := &recordingPublisher{}
	obs := newObserver(t, harmovela.Config{
		TenantID:    "tenant_1",
		WorkspaceID: "workspace_1",
		Publisher:   pub,
		TaskID: func(item rheovela.WorkItem) string {
			return "task_" + item.ID
		},
	})

	item := rheovela.WorkItem{ID: "wi_3", State: "ready"}
	if err := obs.Observe(context.Background(), item); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	event := lastEvent(t, pub)
	if event["tenant_id"] != "tenant_1" {
		t.Fatalf("tenant_id = %v, want tenant_1", event["tenant_id"])
	}
	if event["workspace_id"] != "workspace_1" {
		t.Fatalf("workspace_id = %v, want workspace_1", event["workspace_id"])
	}
	if event["task_id"] != "task_wi_3" {
		t.Fatalf("task_id = %v, want task_wi_3", event["task_id"])
	}
}

func TestSessionOpenCloseIdempotent(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	pub := &recordingPublisher{}
	obs := newObserver(t, harmovela.Config{Publisher: pub, Now: func() time.Time { return now }})
	ctx := context.Background()

	if err := obs.Open(ctx); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := obs.Open(ctx); err != nil {
		t.Fatalf("Open again: %v", err)
	}
	assertTypes(t, pub, "session.opened", "session.ready")

	for i, want := range []string{"evt_session_1", "evt_session_2"} {
		event := pub.events[i]
		if event["id"] != want {
			t.Fatalf("event[%d] id = %v, want %s", i, event["id"], want)
		}
		if event["spec_version"] != "0.2" {
			t.Fatalf("spec_version = %v, want 0.2", event["spec_version"])
		}
		if event["source"] != "rheovela" {
			t.Fatalf("source = %v, want rheovela", event["source"])
		}
		if event["session_id"] != "rheovela" {
			t.Fatalf("session_id = %v, want rheovela", event["session_id"])
		}
		if event["created_at"] != "2026-01-02T03:04:05Z" {
			t.Fatalf("created_at = %v", event["created_at"])
		}
		payload := event["payload"].(map[string]any)
		if payload["session_id"] != "rheovela" {
			t.Fatalf("payload session_id = %v, want rheovela", payload["session_id"])
		}
	}

	if err := obs.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := obs.Close(ctx); err != nil {
		t.Fatalf("Close again: %v", err)
	}
	assertTypes(t, pub, "session.opened", "session.ready", "session.closed")
	if lastEvent(t, pub)["id"] != "evt_session_3" {
		t.Fatalf("closed id = %v, want evt_session_3", lastEvent(t, pub)["id"])
	}
}

func TestNewRequiresPublisher(t *testing.T) {
	if _, err := harmovela.New(harmovela.Config{}); err == nil {
		t.Fatal("expected error when Publisher is nil")
	}
}
