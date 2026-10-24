package rheovela

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/axisvela/rheovela-open/sdk/go/v3/worker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPWorkStorePollReadyMapsClaimable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/work-items" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("instance"); got != "" {
			t.Errorf("unexpected instance filter: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"w1","instance_id":"i1","activity_id":"a1","state":"ready","result":{"ok":true},"checkpoint_ref":"ckpt-1","effect_cursor":"cur-1","approval":{"state":"pending","pending_stage":"review","final_args":{"x":1},"reason":"needs review","requested_by":"agent","requested_at":"2026-10-14T00:00:00Z","decided_by":"human","decided_at":"2026-10-14T01:00:00Z","comment":"looks good","checkpoint_ref":"ckpt-a","effect_cursor":"cur-a"}},
			{"id":"w2","instance_id":"i2","activity_id":"a2","state":"claimed","lease_until":"2026-10-13T00:00:00Z"},
			{"id":"w3","instance_id":"i3","activity_id":"a3","state":"awaiting_approval"}
		]`))
	}))
	defer srv.Close()

	store := NewHTTPWorkStore(NewClient(srv.URL, WithHTTPClient(srv.Client())))
	items, err := store.PollReady()
	require.NoError(t, err)
	require.Len(t, items, 2, "ready and expired claimed items are returned")

	it := items[0]
	assert.Equal(t, "w1", it.ID)
	assert.Equal(t, "i1", it.InstanceID)
	assert.Equal(t, "a1", it.ActivityID)
	assert.Equal(t, "ready", it.State)
	assert.JSONEq(t, `{"ok":true}`, string(it.Result))
	assert.Equal(t, "ckpt-1", it.CheckpointRef)
	assert.Equal(t, "cur-1", it.EffectCursor)

	require.NotNil(t, it.Approval)
	assert.Equal(t, "pending", it.Approval.State)
	assert.Equal(t, "review", it.Approval.PendingStage)
	assert.JSONEq(t, `{"x":1}`, string(it.Approval.FinalArgs))
	assert.Equal(t, "needs review", it.Approval.Reason)
	assert.Equal(t, "agent", it.Approval.RequestedBy)
	assert.Equal(t, "human", it.Approval.DecidedBy)
	assert.Equal(t, "looks good", it.Approval.Comment)
	assert.Equal(t, "ckpt-a", it.Approval.CheckpointRef)
	assert.Equal(t, "cur-a", it.Approval.EffectCursor)

	claimed := items[1]
	assert.Equal(t, "w2", claimed.ID)
	assert.Equal(t, "i2", claimed.InstanceID)
	assert.Equal(t, "a2", claimed.ActivityID)
	assert.Equal(t, "claimed", claimed.State)
}

func TestHTTPWorkStoreClaim(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/work-items/w1/claim" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"ft-1","lease":"30s","lease_until":"2026-10-14T00:00:00Z"}`))
	}))
	defer srv.Close()

	store := NewHTTPWorkStoreWithContext(context.Background(), NewClient(srv.URL, WithHTTPClient(srv.Client())))
	token, err := store.Claim("w1", 30*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "ft-1", token)
	assert.Equal(t, "30s", body["lease"])
}

func TestHTTPWorkStoreCompleteWithResult(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/work-items/w1/complete" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	store := NewHTTPWorkStore(NewClient(srv.URL, WithHTTPClient(srv.Client())))
	err := store.CompleteWithResult("w1", "tok-1", worker.WorkResult{
		Result:        json.RawMessage(`{"ok":true}`),
		CheckpointRef: "ckpt-1",
		EffectCursor:  "cur-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "tok-1", body["token"])
	assert.Equal(t, map[string]any{"ok": true}, body["result"])
	assert.Equal(t, "ckpt-1", body["checkpoint_ref"])
	assert.Equal(t, "cur-1", body["effect_cursor"])
}

func TestHTTPWorkStoreRequestApproval(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/work-items/w1/request-approval" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	store := NewHTTPWorkStoreWithContext(context.Background(), NewClient(srv.URL, WithHTTPClient(srv.Client())))
	err := store.RequestApproval("w1", "tok-1", worker.ApprovalRequest{
		Actor:         "agent",
		Reason:        "over limit",
		FinalArgs:     json.RawMessage(`{"amount":10}`),
		CheckpointRef: "ckpt-1",
		EffectCursor:  "cur-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "tok-1", body["token"])
	assert.Equal(t, "agent", body["actor"])
	assert.Equal(t, "over limit", body["reason"])
	assert.Equal(t, map[string]any{"amount": float64(10)}, body["final_args"])
	assert.Equal(t, "ckpt-1", body["checkpoint_ref"])
	assert.Equal(t, "cur-1", body["effect_cursor"])
}
