# Harmovela Producer Adapter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a transport-agnostic Go adapter that publishes Rheovela work-item lifecycle as Harmovela `task.*` events.

**Architecture:** New module `sdk/go/harmovela` depends on the open Go SDK and Harmovela’s `task` package. An `Observer` keeps `task.Tracker` state per work item, builds valid envelopes, and publishes through an injected `Publisher`.

**Tech Stack:** Go 1.26, open Go SDK, Harmovela Go `task` package, testify.

**Spec:** `docs/superpowers/specs/2026-10-26-harmovela-adapter-design.md`

---

## File Structure

| File | Responsibility |
| --- | --- |
| `sdk/go/harmovela/go.mod` | New module and local replaces |
| `sdk/go/harmovela/adapter.go` | Publisher, Config, Observer, event mapping |
| `sdk/go/harmovela/adapter_test.go` | In-memory publisher tests |
| `sdk/go/harmovela/README.md` | Usage and mapping docs |
| `.github/workflows/ci.yml` | Adapter test job |
| `docs/STATUS.md`, `docs/ROADMAP.md` | Status/roadmap |
| `docs/testing/2026-10-26-harmovela-adapter/README.md` | Evidence |

---

### Task 1: Module, Publisher, sessions, and task mapping

**Files:**
- Create: `sdk/go/harmovela/go.mod`
- Create: `sdk/go/harmovela/adapter.go`
- Create: `sdk/go/harmovela/adapter_test.go`

- [ ] **Step 1: Create the module**

`go.mod`:

```go
module github.com/axisvela/rheovela-open/sdk/go/harmovela/v3

go 1.26.0

toolchain go1.26.10

require (
	github.com/axisvela/harmovela/implementations/go/v3 v3.0.0
	github.com/axisvela/rheovela-open/sdk/go/v3 v3.1.0
)

replace github.com/axisvela/rheovela-open/sdk/go/v3 => ../

replace github.com/axisvela/harmovela/implementations/go/v3 => ../../../../Harmovela/implementations/go
```

Run `go mod tidy` in the adapter directory.

- [ ] **Step 2: Write failing tests**

`adapter_test.go`:

```go
package harmovela_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	adapter "github.com/axisvela/rheovela-open/sdk/go/harmovela/v3"
	sdk "github.com/axisvela/rheovela-open/sdk/go/v3/rheovela"
)

type capture struct{ events []map[string]any }
func (c *capture) Publish(_ context.Context, event map[string]any) error { c.events = append(c.events, event); return nil }

func newObserver(t *testing.T) (*adapter.Observer, *capture) {
	t.Helper()
	c := &capture{}
	o, err := adapter.New(adapter.Config{Publisher: c, Now: func() time.Time {
		return time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)
	}})
	if err != nil { t.Fatal(err) }
	return o, c
}

func types(events []map[string]any) []string {
	out := make([]string, 0, len(events))
	for _, e := range events { out = append(out, e["type"].(string)) }
	return out
}

func TestObserveLifecycle(t *testing.T) {
	o, c := newObserver(t)
	ctx := context.Background()
	item := sdk.WorkItem{ID: "wi1", InstanceID: "i1", ActivityID: "a1", State: "ready"}
	if err := o.Observe(ctx, item); err != nil { t.Fatal(err) }
	item.State = "claimed"
	if err := o.Observe(ctx, item); err != nil { t.Fatal(err) }
	item.State = "done"
	item.Result = json.RawMessage(`{"ok":true}`)
	item.CheckpointRef = "ckpt-1"
	item.EffectCursor = "cur-1"
	if err := o.Observe(ctx, item); err != nil { t.Fatal(err) }
	want := []string{"task.submitted", "task.accepted", "task.started", "task.completed"}
	got := types(c.events)
	if len(got) != len(want) { t.Fatalf("types = %v", got) }
	for i := range want { if got[i] != want[i] { t.Fatalf("types = %v", got) } }
	completed := c.events[len(c.events)-1]["payload"].(map[string]any)
	if completed["checkpoint_ref"] != "ckpt-1" || completed["effect_cursor"] != "cur-1" {
		t.Fatalf("payload = %+v", completed)
	}
}

func TestObserveApprovalResume(t *testing.T) {
	o, c := newObserver(t)
	ctx := context.Background()
	item := sdk.WorkItem{ID: "wi1", State: "ready"}
	_ = o.Observe(ctx, item)
	item.State = "claimed"; _ = o.Observe(ctx, item)
	item.State = "awaiting_approval"
	item.Approval = &sdk.WorkItemApproval{State: "pending", Reason: "review"}
	_ = o.Observe(ctx, item)
	item.State = "ready"; _ = o.Observe(ctx, item)
	want := "task.blocked"
	if c.events[len(c.events)-1]["type"] != want { t.Fatalf("last = %v", c.events[len(c.events)-1]) }
}

func TestObserveSameStateNoop(t *testing.T) {
	o, c := newObserver(t)
	item := sdk.WorkItem{ID: "wi1", State: "ready"}
	_ = o.Observe(context.Background(), item)
	n := len(c.events)
	_ = o.Observe(context.Background(), item)
	if len(c.events) != n { t.Fatalf("duplicate event: %v", types(c.events)) }
}

func TestObserveInvalidTransition(t *testing.T) {
	o, c := newObserver(t)
	if err := o.Observe(context.Background(), sdk.WorkItem{ID: "wi1", State: "done"}); err == nil {
		t.Fatal("expected invalid transition")
	}
	if len(c.events) != 0 { t.Fatalf("events = %v", types(c.events)) }
}

func TestSessionOpenCloseIdempotent(t *testing.T) {
	o, c := newObserver(t)
	_ = o.Open(context.Background()); _ = o.Open(context.Background())
	_ = o.Close(context.Background()); _ = o.Close(context.Background())
	want := []string{"session.opened", "session.ready", "session.closed"}
	got := types(c.events)
	if len(got) != len(want) { t.Fatalf("types = %v", got) }
	for i := range want { if got[i] != want[i] { t.Fatalf("types = %v", got) } }
}
```

- [ ] **Step 3: Implement the adapter**

Create `adapter.go` with:

```go
type Publisher interface { Publish(context.Context, map[string]any) error }
type PublisherFunc func(context.Context, map[string]any) error
func (f PublisherFunc) Publish(ctx context.Context, event map[string]any) error { return f(ctx, event) }

type Config struct {
	Source, TenantID, WorkspaceID, SessionID string
	Publisher Publisher
	Now func() time.Time
	TaskID func(rheovela.WorkItem) string
}

type Observer struct {
	cfg Config
	mu sync.Mutex
	tasks map[string]*task.Tracker
	last map[string]string
	sessionOpen bool
	sessionSeq int
}

func New(cfg Config) (*Observer, error)
func (o *Observer) Open(ctx context.Context) error
func (o *Observer) Close(ctx context.Context) error
func (o *Observer) Observe(ctx context.Context, item rheovela.WorkItem) error
```

Mapping:

- `ready` from none -> `task.submitted`
- `ready` from `blocked` -> `task.started`
- `claimed` from none/submitted -> `task.accepted`, `task.started`
- `claimed` from accepted/blocked -> `task.started`
- `awaiting_approval` -> `task.blocked` with reason/pending stage/approval state
- `done` -> `task.completed` with result/checkpoint/effect
- `failed` -> `task.failed` with `last_error` or `work item failed`
- `cancelled` -> `task.cancelled`
- same state -> no-op
- invalid transition -> error, no publish
- add `tenant_id`/`workspace_id` top-level when configured
- session events manually build `{spec_version:"0.2", id, type, source, session_id, created_at, payload}`; `Open` emits opened+ready; `Close` emits closed.

Use `task.NewTracker(taskID, source, activityID)` and `Tracker.Transition`. If `Transition` returns nil, return `fmt.Errorf("harmovela: invalid transition ...")`.

- [ ] **Step 4: Run adapter tests**

```powershell
go test ./... -count=1
```
from `sdk/go/harmovela`.

- [ ] **Step 5: Commit**

```bash
git add sdk/go/harmovela/go.mod sdk/go/harmovela/go.sum sdk/go/harmovela/adapter.go sdk/go/harmovela/adapter_test.go
git commit -m "feat(harmovela): add producer observer adapter"
```

---

### Task 2: CI, docs, and evidence

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`
- Create: `sdk/go/harmovela/README.md`
- Create: `docs/testing/2026-10-26-harmovela-adapter/README.md`

- [ ] Add CI job:
  ```yaml
  harmovela-adapter:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: go test ./...
        working-directory: sdk/go/harmovela
  ```
- Update open STATUS/ROADMAP: Harmovela producer adapter implemented (unreleased);
  consumer remains in core; built-in transports remain out of scope.
- README: mapping table, `Publisher` example, session usage, raw result note.
- Evidence README with commands/results.
- Run:
  ```powershell
  go test ./...                 # open root
  go test ./...                 # sdk/go
  go test ./...                 # sdk/go/harmovela
  ```
- Commit `docs(open): record Harmovela producer adapter`.

---

## Self-Review Notes

- Producer/observer only; core consumer untouched.
- Transport is injected; no network dependency.
- Module is isolated from the worker SDK.
- Tests cover lifecycle, approval/resume, failure/cancel (add if missing), same-state, invalid transition, sessions.
- Docs explicitly mark the adapter unreleased until the next release.
