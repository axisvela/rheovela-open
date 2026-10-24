> English · [中文](README.md)

# Go Worker SDK — `sdk/worker`

The Go SDK for workers (Apache-2.0): a durable task consumption loop of
**claim → heartbeat → complete/fail**. It handles claim / lease / fencing token /
idempotent completion and structured failure by default; the worker does not need to
care about SQLite, transactions or event-stream details.

## WorkStore — durable runtime port

`WorkStore` is the **port** between a worker and the durable runtime. A worker only
depends on this interface, not on a concrete storage implementation:

```go
type WorkStore interface {
    PollReady() ([]WorkItem, error)                                   // fetch all claimable work items (ready + expired claimed)
    Claim(id string, lease time.Duration) (token string, err error)   // claim, returns a fencing token
    Heartbeat(id, token string, lease time.Duration) error            // renew the lease
    Complete(id, token string) error                                  // idempotent completion
    Fail(id, token, errMsg string) error                              // structured failure
}
```

The **fencing token** returned by `Claim` is a concurrency guard: only the worker
holding the token can `Complete`/`Fail` the item, preventing a stale worker from
writing after its lease expired and the item was re-claimed by another worker.

`WorkItem` / `WorkResult`:

```go
type WorkItem struct {
    ID            string
    InstanceID    string
    ActivityID    string
    State         string
    Result        json.RawMessage
    CheckpointRef string
    EffectCursor  string
    Approval      *Approval
}

type WorkResult struct {
    Status        string // "success" | "failure" | "approval"
    Error         string
    Outcome       string
    Result        json.RawMessage
    CheckpointRef string
    EffectCursor  string
    Approval      *ApprovalRequest
}
```

## How to implement WorkStore

### Option A: embed core's `runtime.WorkerBridge` (recommended, local/edge)

Core (AGPL) ships two bridges in the public `runtime` package:

- **`LegacyWorkerBridge`** (constructor `runtime.NewWorkerBridge`): implements
  `WorkStore` only. `Complete` goes through the broker and **does not persist
  results/checkpoint/effect_cursor**; approvals are unsupported. Kept for
  compatibility.
- **`WorkerBridge`** (constructor `runtime.NewWorkerBridgeWithService`):
  implements `WorkStore` plus the optional `ResultStore` and `ApprovalStore`
  ports. `Complete` / `CompleteWithResult` delegate to
  `Service.CompleteWorkItem`, atomically advancing the stage, persisting the
  result, and creating the next stage item. `RequestApproval` moves the item to
  `awaiting_approval`. Production code should use this bridge.

```go
import (
    "github.com/axisvela/rheovela/runtime"   // core public API
    worker "github.com/axisvela/rheovela-open/sdk/worker"
)

db, err := runtime.OpenDB(dbPath)      // open SQLite
st := runtime.NewStore(db)
svc := runtime.NewService(st)
bridge := runtime.NewWorkerBridgeWithService(st, svc,
    runtime.WithWorkerActor("worker-1"), // recorded as the StepCompleted actor
)
```

- **`WithWorkerActor(actor)`**: sets the worker actor recorded on Service commands.
  With an Identity provider configured, the actor must hold the execute scope
  (and the approve scope for approval requests).
- **`ResultStore`**: `CompleteWithResult(id, token, WorkResult)` persists the
  result together with `checkpoint_ref` / `effect_cursor`.
- **`ApprovalStore`**: `RequestApproval(id, token, ApprovalRequest)` requests a
  human decision and fails closed when no Service is configured.

Dependency direction: the `worker` SDK (Apache) itself has zero dependency on core;
both bridges live on the core side and bridge core's broker/service semantics to the
SDK's ports.

### Option B: implement `WorkStore` yourself

Implement the five methods above on your own backend (e.g. a future Postgres adapter,
a remote gateway, or an enterprise broker). `RunStoreContract` (core
`internal/store/contract_test.go`) is the semantic-consistency gate that every storage
adapter must pass.

## Worker / WorkFn / ProcessOnce

`Worker` threads "poll + claim + execute + settle" into one loop:

```go
w := worker.New(bridge, fn, lease, opts...)
n, err := w.ProcessOnce(ctx)   // poll once and process all ready items, returns the processed count
```

- **`fn`**: `WorkFn = func(ctx context.Context, item WorkItem) WorkResult`.
- **`lease`**: per-item lease duration (`time.Duration`).
- **`ProcessOnce`**: `PollReady` → `Claim` each item (skipping items that fail to
  claim due to concurrency / being unclaimable) → run `fn` → on `success` call
  `Complete`, otherwise `Fail(res.Error)`; when `ctx` is cancelled it stops
  immediately and returns `context.Canceled`.
- **`WithLogger(logf)`**: optional logging (`logf(format, args...)`) printing
  claim/complete/fail.

## Minimal example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/axisvela/rheovela/runtime"
    worker "github.com/axisvela/rheovela-open/sdk/worker"
)

func main() {
    db, err := runtime.OpenDB("~/.proc/proc.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    st := runtime.NewStore(db)
    svc := runtime.NewService(st)
    bridge := runtime.NewWorkerBridgeWithService(st, svc,
        runtime.WithWorkerActor("worker-1"))

    // fn: runs the work item and returns success/failure
    fn := func(ctx context.Context, item worker.WorkItem) worker.WorkResult {
        fmt.Printf("working on %s (activity=%s)\n", item.ID, item.ActivityID)
        return worker.WorkResult{Status: "success", Outcome: "ok"}
    }

    w := worker.New(bridge, fn, 30*time.Second,
        worker.WithLogger(func(format string, args ...any) { fmt.Printf(format+"\n", args...) }))

    for {
        n, err := w.ProcessOnce(context.Background())
        if err != nil {
            fmt.Printf("process: %v\n", err)
        }
        if n == 0 {
            time.Sleep(2 * time.Second) // nothing to do, retry later
        }
    }
}
```

## Working with core

- core's `rheo serve` exposes the HTTP Ops API (`api/README.md`); `work_items` are
  produced by the scheduler / broker and consumed by the worker SDK.
- The event stream (`StepEntered` / `StepCompleted`, etc.) and the effect ledger are
  maintained by core; the worker just settles via `WorkStore` and never writes to the
  event table directly.
- Existing tests are in `worker_test.go` (complete/fail/concurrent-skip/ctx-cancel).
