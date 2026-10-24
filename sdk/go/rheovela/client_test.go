package rheovela

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"
)

// capture is a mutex-guarded record of the last request the fake server saw.
type capture struct {
	mu   sync.Mutex
	auth string
	path string
	body string
}

func (c *capture) set(r *http.Request) {
	b := make([]byte, r.ContentLength)
	if r.ContentLength > 0 {
		_, _ = r.Body.Read(b)
	}
	c.mu.Lock()
	c.auth = r.Header.Get("Authorization")
	c.path = r.URL.Path
	c.body = string(b)
	c.mu.Unlock()
}

func (c *capture) get() (auth, path, body string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.auth, c.path, c.body
}

func newFake(t *testing.T, h http.HandlerFunc) (*Client, *capture) {
	t.Helper()
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.set(r)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, WithTokenSource(StaticTokenSource("tok-1"))), cap
}

func TestClient_OpenRun(t *testing.T) {
	c, cap := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"instance_id":"i1","status":"active","project_id":"p1"}`))
	})
	inst, err := c.OpenRun(context.Background(), OpenRunRequest{Workflow: "wf-1", ProjectID: "p1", Actor: "axond"})
	if err != nil {
		t.Fatalf("OpenRun: %v", err)
	}
	if inst.InstanceID != "i1" || inst.Status != "active" {
		t.Errorf("instance = %+v", inst)
	}
	auth, path, body := cap.get()
	if auth != "Bearer tok-1" {
		t.Errorf("auth = %q, want Bearer tok-1", auth)
	}
	if path != "/api/v1/instances" {
		t.Errorf("path = %q", path)
	}
	if body == "" {
		t.Error("expected request body")
	}
}

func TestClient_Step(t *testing.T) {
	c, cap := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	if err := c.Step(context.Background(), "i1", StepRequest{Action: "enter", Stage: "s1"}); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if _, path, _ := cap.get(); path != "/api/v1/instances/i1/steps" {
		t.Errorf("path = %q", path)
	}
}

func TestClient_ErrorStatus(t *testing.T) {
	c, _ := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"instance not found"}`))
	})
	if _, err := c.GetInstance(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClient_ClaimWorkItem(t *testing.T) {
	c, cap := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"ft-1","lease":"30s","lease_until":"2026-10-14T00:00:00Z","work_item":{"id":"w1","instance_id":"i1","activity_id":"a1","state":"ready","result":{"ok":true},"checkpoint_ref":"ckpt-1"}}`))
	})
	claim, err := c.ClaimWorkItem(context.Background(), "w1", "30s")
	if err != nil {
		t.Fatalf("ClaimWorkItem: %v", err)
	}
	if claim.Token != "ft-1" {
		t.Errorf("token = %q", claim.Token)
	}
	if claim.WorkItem == nil {
		t.Fatal("claim.WorkItem is nil")
	}
	if claim.WorkItem.State != "ready" {
		t.Errorf("state = %q", claim.WorkItem.State)
	}
	if got := string(claim.WorkItem.Result); got != `{"ok":true}` {
		t.Errorf("result = %s", got)
	}
	if claim.WorkItem.CheckpointRef != "ckpt-1" {
		t.Errorf("checkpoint_ref = %q", claim.WorkItem.CheckpointRef)
	}
	if _, path, _ := cap.get(); path != "/api/v1/work-items/w1/claim" {
		t.Errorf("path = %q", path)
	}
}

func TestWorkItemResultAndApprovalMethods(t *testing.T) {
	var paths []string
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		bodies = append(bodies, body)
		switch r.URL.Path {
		case "/api/v1/work-items/wi1":
			_, _ = w.Write([]byte(`{"id":"wi1","instance_id":"i1","activity_id":"a1","state":"ready","result":{"ok":true},"checkpoint_ref":"ckpt-1","effect_cursor":"cur-1","attempts":2,"max_attempts":5,"last_error":"boom","approval":{"state":"pending","pending_stage":"review","final_args":{"x":1},"reason":"needs review","requested_by":"agent","requested_at":"2026-10-14T00:00:00Z","decided_by":"human","decided_at":"2026-10-14T01:00:00Z","comment":"looks good","checkpoint_ref":"ckpt-1","effect_cursor":"cur-1"}}`))
		case "/api/v1/work-items/wi1/complete", "/api/v1/work-items/wi1/request-approval", "/api/v1/work-items/wi1/approve", "/api/v1/work-items/wi1/reject":
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	c := NewClient(server.URL, WithHTTPClient(server.Client()))
	wi, err := c.GetWorkItem(context.Background(), "wi1")
	if err != nil {
		t.Fatal(err)
	}
	if string(wi.Result) != `{"ok":true}` {
		t.Errorf("result = %s", wi.Result)
	}
	if wi.CheckpointRef != "ckpt-1" {
		t.Errorf("checkpoint_ref = %q", wi.CheckpointRef)
	}
	if wi.EffectCursor != "cur-1" {
		t.Errorf("effect_cursor = %q", wi.EffectCursor)
	}
	if wi.Attempts != 2 || wi.MaxAttempts != 5 || wi.LastError != "boom" {
		t.Errorf("attempts/max/last = %d/%d/%q", wi.Attempts, wi.MaxAttempts, wi.LastError)
	}
	if wi.Approval == nil {
		t.Fatal("approval is nil")
	}
	if wi.Approval.State != "pending" || wi.Approval.PendingStage != "review" {
		t.Errorf("approval state/stage = %q/%q", wi.Approval.State, wi.Approval.PendingStage)
	}
	if got := string(wi.Approval.FinalArgs); got != `{"x":1}` {
		t.Errorf("approval final_args = %s", got)
	}
	if wi.Approval.Reason != "needs review" || wi.Approval.RequestedBy != "agent" {
		t.Errorf("approval reason/requested_by = %q/%q", wi.Approval.Reason, wi.Approval.RequestedBy)
	}
	if !wi.Approval.RequestedAt.Equal(time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("requested_at = %v", wi.Approval.RequestedAt)
	}
	if wi.Approval.DecidedBy != "human" || wi.Approval.Comment != "looks good" {
		t.Errorf("approval decided_by/comment = %q/%q", wi.Approval.DecidedBy, wi.Approval.Comment)
	}
	if wi.Approval.DecidedAt == nil || !wi.Approval.DecidedAt.Equal(time.Date(2026, 10, 14, 1, 0, 0, 0, time.UTC)) {
		t.Errorf("decided_at = %v", wi.Approval.DecidedAt)
	}
	if wi.Approval.CheckpointRef != "ckpt-1" || wi.Approval.EffectCursor != "cur-1" {
		t.Errorf("approval checkpoint/effect = %q/%q", wi.Approval.CheckpointRef, wi.Approval.EffectCursor)
	}

	if err := c.CompleteWorkItemWithResult(context.Background(), "wi1", "tok", WorkResult{Result: json.RawMessage(`{"ok":true}`), CheckpointRef: "ckpt-1"}); err != nil {
		t.Fatal(err)
	}
	if err := c.RequestWorkItemApproval(context.Background(), "wi1", "tok", ApprovalRequest{
		Actor:         "agent",
		Reason:        "review",
		FinalArgs:     json.RawMessage(`{"x":1}`),
		CheckpointRef: "ckpt-1",
		EffectCursor:  "cur-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := c.ApproveWorkItem(context.Background(), "wi1", "human", "ok", json.RawMessage(`{"approved":true}`)); err != nil {
		t.Fatal(err)
	}
	if err := c.RejectWorkItem(context.Background(), "wi1", "human", "no"); err != nil {
		t.Fatal(err)
	}

	wantPaths := []string{
		"GET /api/v1/work-items/wi1",
		"POST /api/v1/work-items/wi1/complete",
		"POST /api/v1/work-items/wi1/request-approval",
		"POST /api/v1/work-items/wi1/approve",
		"POST /api/v1/work-items/wi1/reject",
	}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("paths = %v, want %v", paths, wantPaths)
	}

	complete := bodies[1]
	if complete["token"] != "tok" || complete["checkpoint_ref"] != "ckpt-1" {
		t.Errorf("complete body = %v", complete)
	}
	if got := mustJSON(t, complete["result"]); got != `{"ok":true}` {
		t.Errorf("complete result = %s", got)
	}

	requestApproval := bodies[2]
	if requestApproval["token"] != "tok" || requestApproval["reason"] != "review" || requestApproval["actor"] != "agent" {
		t.Errorf("request-approval body = %v", requestApproval)
	}
	if got := mustJSON(t, requestApproval["final_args"]); got != `{"x":1}` {
		t.Errorf("request-approval final_args = %s", got)
	}
	if requestApproval["checkpoint_ref"] != "ckpt-1" || requestApproval["effect_cursor"] != "cur-1" {
		t.Errorf("request-approval refs = %v", requestApproval)
	}

	approve := bodies[3]
	if approve["actor"] != "human" || approve["comment"] != "ok" {
		t.Errorf("approve body = %v", approve)
	}
	if got := mustJSON(t, approve["final_args"]); got != `{"approved":true}` {
		t.Errorf("approve final_args = %s", got)
	}

	reject := bodies[4]
	if reject["actor"] != "human" || reject["reason"] != "no" {
		t.Errorf("reject body = %v", reject)
	}
}

func TestLegacyCompleteWorkItemSendsTokenOnly(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, WithHTTPClient(server.Client()))
	if err := c.CompleteWorkItem(context.Background(), "wi1", "tok"); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"token": "tok"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %v, want %v", got, want)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
