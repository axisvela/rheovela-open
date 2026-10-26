# Harmovela producer adapter

Package `harmovela` adapts Rheovela work-item lifecycle observations to
Harmovela `task.*` events. It is transport-agnostic: callers inject a
`Publisher` that carries each event map to their transport of choice
(WebSocket, gRPC, NATS, in-process, etc.).

This adapter is the producer/observer direction only. Consuming Harmovela tasks
and feeding them back into Rheovela is owned by core.

## Module

```
module github.com/axisvela/rheovela-open/sdk/go/harmovela/v3
```

```go
import harmovela "github.com/axisvela/rheovela-open/sdk/go/harmovela/v3"
```

The adapter depends on the published
`github.com/axisvela/harmovela/implementations/go/v3 v3.0.0` module. No local
replace is required.

## Publisher

```go
type Publisher interface {
	Publish(context.Context, map[string]any) error
}

// PublisherFunc adapts a function to Publisher.
type PublisherFunc func(context.Context, map[string]any) error
```

Example with `PublisherFunc`:

```go
pub := harmovela.PublisherFunc(func(ctx context.Context, event map[string]any) error {
	return transport.Send(ctx, event)
})

obs, err := harmovela.New(harmovela.Config{
	TenantID:    "tenant_1",
	WorkspaceID: "workspace_1",
	Publisher:   pub,
})
if err != nil {
	return err
}

if err := obs.Open(ctx); err != nil {
	return err
}
defer obs.Close(ctx)

if err := obs.Observe(ctx, item); err != nil {
	return err
}
```

## Config and Observer

`Config` fields:

| Field | Meaning |
| --- | --- |
| `Source` | event source; defaults to `rheovela` |
| `TenantID` | optional envelope `tenant_id` |
| `WorkspaceID` | optional envelope `workspace_id` |
| `SessionID` | session identifier; defaults to `rheovela` |
| `Publisher` | required transport sink |
| `Now` | clock; defaults to `time.Now` |
| `TaskID` | maps a `rheovela.WorkItem` to a Harmovela task ID; defaults to the work-item ID |

Observer methods:

- `Open(ctx)` — emits `session.opened` then `session.ready`; repeated calls are no-ops.
- `Observe(ctx, item)` — maps the work item's current `State` to `task.*` events; observing the same state twice is a no-op.
- `Close(ctx)` — emits `session.closed` once; repeated calls are no-ops.

## State mapping

| Rheovela state | Harmovela event(s) |
| --- | --- |
| `ready` | `task.submitted` on first observation; `task.started` when resuming from `task.blocked` |
| `claimed` | `task.accepted` + `task.started` from submitted; `task.started` from accepted/blocked |
| `awaiting_approval` | `task.blocked` with `reason`, `pending_stage`, `approval_state` |
| `done` | `task.completed` |
| `failed` | `task.failed` with code `rheovela.work_item_failed` |
| `cancelled` | `task.cancelled` |

Invalid transitions return an error without publishing. A terminal task cannot
be reopened.

## Notes

- `done` payload carries the work item's `Result` as the raw result JSON, plus
  `checkpoint_ref` and `effect_cursor` when present.
- `tenant_id` and `workspace_id` envelope fields are emitted only when
  configured.
- The transport is injected; this package does not ship WebSocket, gRPC, or NATS
  implementations.
