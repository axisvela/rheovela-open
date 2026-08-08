> English · [中文](product.md)

# RHEOVELA — Product Introduction

> RHEOVELA: **Dynamic Process & Durable Workflow Platform**.
> Materializes a governed **Capability Plan** into a durable, resumable,
> auditable **Process Instance**.
> Current state: **v0.9.0-beta** (public release layer, Apache-2.0).

## Positioning

RHEOVELA is a runtime platform for "dynamic process + durable workflow". Core ideas:

1. **Capability as contract**: a process consists of activities; each activity declares
   the capability and role it requires via `AgentTask` / `HumanTask` / `ServiceTask`,
   instead of binding to a concrete implementation.
2. **Event sourcing**: every state change is an appended event; the current state is
   derived by folding the event stream with pure functions. Replay is rebuild —
   naturally auditable and debuggable.
3. **Deterministic kernel**: activation state machine (AND-join / XOR-split / bypass)
   plus a typed condition AST — the same event stream always yields the same result.
4. **Governance-oriented change**: structural changes do not go through "runtime graph
   editing"; they go through **versioning, migration or approval** (versioned
   definitions + migration dry-run + live migration).
5. **Durability guarantees**: idempotent command pipeline, effect ledger (external
   side-effect bookkeeping), compensation, timers, worker lease/fencing token — so that
   restarts, retries and concurrency never produce duplicate business effects.

## Three-repo structure

| Repo | License | Role |
|------|---------|------|
| [`rheovela-open`](https://github.com/axisrobo/rheovela-open) (this repo) | Apache-2.0 | Public release layer (**v0.9.0-beta**): versioned `contracts/` (`schema_version=1`), `sdk/` (worker SDKs, **5 languages**: Go / Python / TypeScript / Java / Rust), `api/` (HTTP API reference), `examples/`, `docs/` |
| [`rheovela`](https://github.com/axisrobo/rheovela) | AGPL-3.0 | Core: store / kernel / engine / application / scheduler / broker / effects / compensation / migration / server / CLI (`cmd/rheo`), and exposes the public `runtime` package |
| `rheovela-ee` | Enterprise | Enterprise layer: multi-tenancy, HA, audit/evidence explorer, migration console, enterprise IdP (AEGIVELA) |

**Dependency direction**: `rheovela-open` → `rheovela` (core imports open's contracts)
→ `rheovela-ee`. Core contains no enterprise/closed-source features; contracts live
under Apache, so community worker/client SDKs are not infected by AGPL.

## Key concepts

### Event sourcing & deterministic kernel

- Events are append-only (the `events` table); each stream gets a server-assigned
  aggregate sequence number `seq`, preventing duplicates/out-of-order.
- `internal/engine` folds events with pure functions (`KnownEventTypes` — **13 types**:
  `RunOpened`, `StepEntered`, `StepCompleted`, `StepFailed`, `StepSkipped`,
  `RunClosed`, `StageAssigned`, `TimerFired`, `Migrated`,
  `CompensationExecuted`, `SubprocessStarted`, `ProcessSuspended`,
  `ProcessResumed`) to rebuild run state.
- Projections (`run_contexts` / `process_instances`) are written in the same
  transaction as the event stream and can be rebuilt at any time via
  `RebuildProjection`.

### Idempotency

- Every command (open / step / close / assign / migrate) carries an `idempotency_key`.
- Idempotency keys are checked/written inside the command transaction
  (`idempotency_keys`); a request digest detects conflicts, so duplicate submissions
  never produce duplicate effects.

### Effect ledger

- External side effects (calling ERP, making payments, posting entries) first record an
  intent, then an outcome
  (`effect_records(idempotency_key, target, request_digest, state, evidence)`).
- Unknown outcomes enter reconciliation; effects are deduplicated by idempotency key,
  guaranteeing each external action happens exactly once.

### Compensation

- The RHEO IR declares compensation actions with `compensate <activity> with ...`.
- `internal/compensation` builds the compensation plan in **reverse order** and
  executes it (`CompensationExecuted` events); activities that already succeeded can be
  rolled back on the failure path; actions that were not declared fall back to manual
  fallback.

### Migration

- Definitions are versioned (`definition_versions`); running instances are pinned to a
  `definition_version`.
- Migration first dry-runs (`internal/migration.Analyze`): **incremental changes are
  compatible; deleting nodes/edges is incompatible and rejected**; compatible paths
  migrate live via `MigrateInstance` + `Migrated` events, and projections and replay
  respect the migration.

### Evidence

- Every event carries an `actor_id`, a signature (HMAC-SHA256, `--signing-key`) and a
  timestamp.
- `internal/audit` builds the evidence chain: plan → events → assignment → execution,
  and cross-references effect records; HTTP endpoints verify signatures when started
  with `--verify-key`.

## Data model

SQLite (`modernc.org/sqlite`, pure Go, no CGO, WAL) with 13 tables: event stream,
projects, definition versions, process instances, idempotency keys, timers, work
items, effect records, stage executions, etc. Contract types (`Event` /
`WorkflowDefinition` / `ProcessInstance` / `RunContext`) are versioned in `contracts/`.

## Quick start

RHEOVELA's kernel and CLI live in the core repo (AGPL). This repo provides the
contracts, SDKs and examples.

1. Build/install the core CLI: `rheo` (`cmd/rheo`, Windows `rheo.exe`).
2. See the examples: `examples/README.md` — import workflows in `examples/` with
   `rheo workflow define --file <json>`, then open a run with
   `rheo run open <workflow-id> --var k=v`.
3. Write a worker: `sdk/worker/README.md` — implement the `WorkStore` port (or embed
   core's `runtime.WorkerBridge`), and consume work items with `Worker` / `ProcessOnce`.
4. Talk to the HTTP API: `api/README.md` — `rheo serve` exposes routes such as
   `/api/v1/instances`.

Full CLI / Go API docs are in core `docs/api.md`, and the architecture in core
`docs/architecture.md`.
