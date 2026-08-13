# RHEOVELA Developer Guide

> English · [中文](DEVELOPMENT.zh.md)

This guide is for developers working with the `rheovela-open` repository:
extending the contracts, building against the Worker SDKs, authoring example
workflows, and wiring the 5-language SDKs to a running core node.

> **The kernel is elsewhere.** The engine, storage, CLI (`rheo`) and HTTP server
> live in `github.com/axisrobo/rheovela` (AGPL-3.0). `rheovela-open`
> (Apache-2.0) is the public layer: contracts, SDKs, API references, examples
> and CI. Enterprise capabilities live in `github.com/axisrobo/rheovela-ee`.
> This guide covers the open layer; pointers to core docs are marked `core:`.

---

## 1. Repository layout

| Path | Content |
|------|---------|
| `contracts/` | Versioned Go types — `Event`, `WorkflowDefinition`, `Stage`, `Transition`, `RunContext` (`schema_version=1`) |
| `sdk/worker/` | Go Worker SDK — `WorkStore` port + `Worker` loop |
| `sdk/python/` | Python Worker SDK (stdlib-only, Python 3.9+) |
| `sdk/typescript/` | TypeScript Worker SDK (zero-dep, Node 18+ `fetch`) |
| `sdk/java/` | Java Worker SDK (JDK 21, zero deps) |
| `sdk/rust/` | Rust Worker SDK (std-only, zero crates) |
| `api/` | HTTP API reference, OpenAPI 3.0 spec, JSON Schemas + schema tests |
| `examples/` | Sample workflows (RHEO IR DSL) and runnable worker examples |
| `docs/` | Features, product intro, **operations** (`OPERATIONS.md`) and this guide |
| `.github/workflows/ci.yml` | CI: go / python / java / rust jobs |

Dependency direction: `rheovela-open` (Apache) → `rheovela` (AGPL core) →
`rheovela-ee` (Enterprise). Core imports the open contracts; community SDKs are
never AGPL-tainted.

---

## 2. Contracts (`contracts/`)

`contracts/types.go` is the authoritative, versioned contract surface
(`SchemaVersion = "1"`). Changing a contract is a cross-repo change — the core
engine and ee both consume these types.

Key types:

| Type | Purpose |
|------|---------|
| `Event` | immutable event (`type`, `actor_id`, `payload`, `wall_time`, optional `signature`) |
| `WorkflowDefinition` | `workflow_id`, `stages[]`, `transitions[]` |
| `Stage` | `id`, `title`, `assignee`/`assignee_spec`, `inputs`, `outputs`, optional `gate` |
| `AssigneeSpec` | `kind`: `actor` / `role` / `capability` / `open` |
| `Transition` | `from` → `to`, optional `condition` expression |
| `RunContext` | run projection: status, current stage, variables, completed stages |
| `KnownEventTypes` | the catalog of event types the engine folds |

### Adding a new event type

1. Add the name to `KnownEventTypes` (e.g. `"ProcessPaused"`).
2. Update the core engine fold (`internal/engine`, core repo) to handle it.
3. Update `api/event.schema.json` so the JSON Schema covers the new type.
4. Add/extend the round-trip test in `contracts/types_test.go`.
5. Bump `SchemaVersion` only for a **breaking** change; additive fields keep v1.

### JSON Schemas + OpenAPI

- `api/workflow.schema.json` — `WorkflowDefinition` (draft-07)
- `api/event.schema.json` — `Event` (covers all 13 event types)
- `api/openapi.yaml` — HTTP Ops API + Worker API routes
- `api/schema_test.go` — keeps the schemas consistent with `contracts` and
  validates the bundled example definitions

Validate against the schemas:

```sh
go test ./api/ -v                 # schema consistency + sample validation
```

---

## 3. Worker SDKs — how to build a worker

All 5 SDKs implement the same contract against the core Worker HTTP API:

| SDK | Location | Runtime | Loop API |
|-----|----------|---------|----------|
| Go | `sdk/worker` | Go 1.25 | `Worker.ProcessOnce(ctx)` |
| Python | `sdk/python/worker.py` | Python 3.9+ (stdlib) | `Worker.process_once()` |
| TypeScript | `sdk/typescript/worker.ts` | Node 18+ (fetch) | `Worker.processOnce()` |
| Java | `sdk/java/worker/Worker.java` | JDK 21 | `Worker.processOnce(fn, lease)` |
| Rust | `sdk/rust/` | Rust std (offline) | `Worker::process_once(fn, lease)` |

The Worker HTTP API each SDK talks to:

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/work-items?instance=<id>` | list ready work items |
| `POST` | `/api/v1/work-items/{id}/claim` | claim → `{token, lease, lease_until}` |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | renew lease `{token, lease}` |
| `POST` | `/api/v1/work-items/{id}/complete` | complete `{token}` |
| `POST` | `/api/v1/work-items/{id}/fail` | fail `{token, error}` |

A worker is a **loop**: poll ready items → claim (fencing token) → run your
function → complete on success / fail with an error. The SDK handles the
protocol; you provide the work function.

### Go (embedded, local/edge)

The Go SDK defines the `WorkStore` port. In production the core provides
`runtime.WorkerBridge` (implements `WorkStore` on the local broker); you can
also implement the 5 methods against any backend.

```go
db, _ := runtime.OpenDB("~/.proc/proc.db")
st := runtime.NewStore(db)
bridge := runtime.NewWorkerBridge(st)

fn := func(ctx context.Context, item worker.WorkItem) worker.WorkResult {
    // do the work...
    return worker.WorkResult{Status: "success", Outcome: "ok"}
}

w := worker.New(bridge, fn, 30*time.Second)
for {
    n, _ := w.ProcessOnce(context.Background())
    if n == 0 { time.Sleep(2 * time.Second) }
}
```

### Python / TypeScript / Java / Rust (HTTP)

Point the `Client` at a `rheo serve` address; the `Worker` loop is identical
across languages:

```python
import worker
w = worker.Worker(worker.Client("http://localhost:8080"), fn, lease="30s")
while True:
    if w.process_once() == 0:
        time.sleep(2)
```

Full per-language docs and code:
- [Go](../sdk/worker/README.en.md) · [Python](../sdk/python/README.en.md) ·
  [TypeScript](../sdk/typescript/README.en.md) · [Java](../sdk/java/README.en.md) ·
  [Rust](../sdk/rust/README.en.md)

### Concurrency & fencing

- `claim` returns a **fencing token**; only the token holder may
  `complete`/`fail`.
- When a lease expires, another worker can reclaim the item — a stale token is
  rejected (prevents double execution).
- Call `heartbeat` for long-running work so the lease does not expire mid-task.

---

## 4. Building & testing

```sh
# Go (contracts, schemas, Go SDK)
go test ./...

# Python SDK
python -m py_compile sdk/python/worker.py
python -m unittest discover -s sdk/python -v

# TypeScript SDK (Node 18+; use --experimental-transform-types on Node 22.7+)
node --experimental-transform-types --test sdk/typescript/worker.test.ts

# Java SDK
javac -d out sdk/java/worker/Worker.java sdk/java/example/Example.java
java -cp out example.Example

# Rust SDK (offline build)
cd sdk/rust && cargo build --offline && cargo run --offline
```

`go test ./api/ -v` additionally validates the JSON Schemas and OpenAPI spec.

---

## 5. Example workflows

`examples/` contains RHEO IR DSL source files plus runnable worker examples.

### RHEO IR DSL

`.rheo` files declare `process`, `activity`, `transition`, `compensate` lines
and compile to a `WorkflowDefinition` JSON (via core `rheo ir tojson`):

```
process PurchaseApproval v1 {
  activity validate : AgentTask capability "invoice.validate"
  activity approve  : HumanTask role "cost-center-owner"
  activity post     : ServiceTask capability "erp.invoice.post" effect_key "invoice/${id}"
  transition validate -> approve when amount >= 1000
  transition validate -> post    when amount < 1000
  compensate post with capability "erp.invoice.void"
}
```

Compile + define + run with the core CLI:

```sh
rheo ir tojson examples/purchase-approval.rheo --output purchase-approval.json
rheo workflow define --file purchase-approval.json
rheo run open PurchaseApproval --var amount=1500 --as alice
# ... step through validate/approve/post, then close
```

### Runnable worker examples

- `examples/go-worker/` — Go worker with an in-memory `fakeStore`
  (`go run ./examples/go-worker/`)
- `examples/python-worker/` — Python worker against a local fake Worker API
  (`python examples/python-worker/main.py`)
- `examples/typescript-worker/` — TS worker with a mocked `fetch`
  (`node examples/typescript-worker/worker-example.mjs`)

Each shows the full poll → claim → complete/fail loop and how to swap the fake
backend for a real `rheo serve` (`Client("http://localhost:8080")`).

---

## 6. CI

`.github/workflows/ci.yml` runs 4 independent jobs on every push/PR:

| Job | What it runs |
|-----|--------------|
| `test` | `go test ./...` (contracts + schemas + Go SDK) |
| `python` | compile + `unittest` for `sdk/python` |
| `java` | `javac` + run `example.Example` |
| `rust` | `cargo build --offline` + `cargo run --offline` |

Keep SDK jobs green before pushing; there is no cross-language test harness —
each SDK is validated against the shared Worker API contract independently.

---

## 7. Release & versioning

Versioned by **semantic-version tags** (`v1.0.0`, `v0.9.0-beta`,
`v1.0.0-rc.1`). Development milestone tags (`v1.x-core`) never produce a GitHub
Release.

| Artifact | Where | How |
|----------|-------|-----|
| Contracts (`schema_version=1`) | `contracts/` | bump only on breaking changes |
| `rheo` binary (5 targets + SHA256SUMS) | core repo Releases | CI on a semantic-version tag |
| OpenAPI + JSON Schemas | `api/` | regenerate + validate via `go test ./api/` |
| CHANGELOG | `CHANGELOG.md` | add an entry per release |

To cut a release:

1. Freeze changes; confirm `go test ./...` and all SDK jobs pass.
2. Tag a semantic version and push (core repo builds binaries; open repo docs/
   contracts ride the same tag name).
3. Update `CHANGELOG.md` and the docs' version stamps.

---

## 8. Troubleshooting (developer)

| Symptom | Likely cause / fix |
|---------|--------------------|
| `go test ./...` fails in `api/` | Schemas drifted from `contracts`; run `go test ./api/ -v` and update the JSON Schema |
| New event type not folding | `KnownEventTypes` updated but core `internal/engine` fold is missing it (core repo) |
| Worker `claim` 409s | Another worker holds the item; raise the lease or heartbeat more often |
| Node 18 cannot run `worker.test.ts` | Transpile with `tsc`/`esbuild`, or use `--experimental-transform-types` on Node 22.7+ |
| Java example won't compile | JDK must be 21+ (`java.net.http`); re-check `javac` classpath |
| Rust build fails offline | External crates were added; keep `sdk/rust` std-only |

---

See also:
- **Operations guide**: [OPERATIONS.md](OPERATIONS.md) (English) · [中文](OPERATIONS.zh.md)
- **Features**: [FEATURES.en.md](FEATURES.en.md) · [中文](FEATURES.md)
- **HTTP API reference**: [api/README.en.md](../api/README.en.md) · [中文](../api/README.md)
- **Core CLI/Go API** (in `github.com/axisrobo/rheovela`): `docs/api.md`
