# Harmovela Producer Adapter

- Date: 2026-10-26
- Status: Approved design, pending implementation
- Owner: open
- Related: `docs/ROADMAP.md` Next item 3, core `internal/harmovela`

## 1. Context

Core can consume Harmovela task events and apply them to bound Rheovela stages
(`internal/harmovela` consumer + handler). The open side currently has no
producer adapter: there is no supported way to publish Rheovela work-item/process
lifecycle as Harmovela `task.*` observations.

This change adds a producer/observer adapter in a separate Go module under
`sdk/go/harmovela`, leaving the worker SDK lightweight.

## 2. Goals

- Publish Rheovela work-item lifecycle as Harmovela task events.
- Provide session open/close events with the Harmovela envelope shape.
- Make transport pluggable through a `Publisher` interface.
- Keep the adapter dependency-isolated from the worker SDK module.
- Provide deterministic mapping and tests with an in-memory publisher.

## 3. Non-goals

- Consuming Harmovela events (core already owns this).
- Built-in WebSocket/gRPC/NATS transport implementations.
- Cross-language Harmovela adapters.
- Changing the Harmovela protocol or core consumer.

## 4. Module Layout

Directory: `sdk/go/harmovela/`

Module: `github.com/axisvela/rheovela-open/sdk/go/harmovela/v3`

Dependencies:

- `github.com/axisvela/rheovela-open/sdk/go/v3` for `rheovela.WorkItem`
- `github.com/axisvela/harmovela/implementations/go/v3/task` for the task
  state machine and event envelopes

Local development uses `replace` directives to `../` and
`../../../../Harmovela/implementations/go`.

## 5. API

```go
type Publisher interface {
    Publish(ctx context.Context, event map[string]any) error
}

type PublisherFunc func(ctx context.Context, event map[string]any) error

func (f PublisherFunc) Publish(ctx context.Context, event map[string]any) error

type Config struct {
    Source      string // default "rheovela"
    TenantID    string // optional envelope tenant
    WorkspaceID string // optional envelope workspace
    SessionID   string // default "rheovela"
    Publisher   Publisher
    Now         func() time.Time
    TaskID      func(rheovela.WorkItem) string // default item.ID
}

func New(cfg Config) (*Observer, error)

func (o *Observer) Open(ctx context.Context) error
func (o *Observer) Close(ctx context.Context) error
func (o *Observer) Observe(ctx context.Context, item rheovela.WorkItem) error
```

`Observer` keeps one `task.Tracker` per task ID and a last-observed state.

## 6. Event Mapping

| Work item state | Previous state | Published event(s) |
| --- | --- | --- |
| `ready` | none | `task.submitted` |
| `ready` | `blocked` | `task.started` |
| `claimed` | none/`submitted` | `task.accepted`, `task.started` |
| `claimed` | `accepted` | `task.started` |
| `claimed` | `blocked` | `task.started` |
| `awaiting_approval` | any non-terminal | `task.blocked` |
| `done` | any non-terminal | `task.completed` with `result`, `checkpoint_ref`, `effect_cursor` |
| `failed` | any non-terminal | `task.failed` with last error |
| `cancelled` | any non-terminal | `task.cancelled` |

Rules:

- Same state observed twice is a no-op (except repeated `progress` is not
  emitted by this adapter).
- Invalid transitions return an error and publish nothing; the tracker state is
  not advanced.
- Terminal tasks are not re-opened.
- The task ID defaults to the work-item ID; `Config.TaskID` can override it.
- Approval/blocked payload includes `reason`, `pending_stage`, and the approval
  state where available.
- Completed payload includes raw `result`, `checkpoint_ref`, and `effect_cursor`
  when present.
- Failure uses Harmovela’s error payload (`code`, `message`, `retryable=false`)
  with the work item’s `last_error` or a generic message.

## 7. Envelope

Use the Harmovela Go `task.Tracker` for task event envelopes. After building,
add optional top-level fields:

- `tenant_id`
- `workspace_id`

Session events (`session.opened`, `session.ready`, `session.closed`) are built
by the adapter with:

```json
{
  "spec_version": "0.2",
  "id": "evt_session_<n>",
  "type": "session.opened",
  "source": "rheovela",
  "session_id": "rheovela",
  "created_at": "<RFC3339>",
  "payload": {"session_id": "rheovela"}
}
```

`Open` emits `session.opened` then `session.ready`; `Close` emits
`session.closed` once. Repeated `Open`/`Close` is a no-op.

## 8. Testing

- Submit -> claim -> complete mapping.
- Approval/blocked mapping and resume.
- Failure and cancellation mapping.
- Same-state no-op.
- Invalid transition returns an error and publishes nothing.
- Tenant/workspace fields added; task ID override.
- Session open/close idempotency.
- In-memory `Publisher` records events; no network.

## 9. CI and Docs

- New CI job for `sdk/go/harmovela` (`go test ./...`).
- README for the adapter with mapping table and publisher example.
- Open STATUS/ROADMAP: Harmovela producer adapter implemented (unreleased).
- Evidence README with commands/results.

## 10. Acceptance Criteria

1. A caller can inject any `Publisher` and observe work items.
2. Work-item lifecycle maps to valid Harmovela `task.*` events.
3. Sessions can be opened/closed idempotently.
4. Invalid transitions fail without emitting events.
5. Adapter tests pass in its own module; worker SDK remains unchanged.
6. Docs/STATUS/ROADMAP and CI reflect the adapter.

## 11. Out of Scope

- WebSocket/gRPC/NATS publisher implementations.
- Consumer/bridge direction.
- Cross-language producer adapters.
