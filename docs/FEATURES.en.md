# RHEOVELA Features

> English · [中文](FEATURES.md)

RHEOVELA is a **Dynamic Process & Durable Workflow Platform**: it materializes governed capability plans into persistent, recoverable, approvable, migratable and auditable Process Instances. It is responsible for "how long-running work reliably flows" across humans, agents, robots and enterprise systems.

This repository (`rheovela-open`) is the Apache-2.0 public layer: versioned contracts, SDKs and APIs. The core runtime lives in [rheovela](https://github.com/axisrobo/rheovela) (AGPL-3.0); enterprise capabilities live in [rheovela-ee](https://github.com/axisrobo/rheovela-ee) (Enterprise).

---

## Core Features

### 1. Event Sourcing + Deterministic Kernel
- Every state change is produced by a validated Command as an immutable, append-only Event. Any projection can be rebuilt from `definition snapshot + ordered events` with identical results.
- The pure-function fold never touches network, clock, randomness or models; replay is deterministic for any legal event sequence.
- **15 event types**: RunOpened / StepEntered / StepCompleted / StepFailed / StepSkipped / RunClosed / StageAssigned / TimerFired / Migrated / SubprocessStarted / CompensationExecuted / ProcessSuspended / ProcessResumed and more.

### 2. Versioned Definitions + Publish Validation
- `definition_versions` are immutable: same digest is idempotent, new content creates a new version; running instances are pinned to a version.
- Definitions are validated before publish: non-empty unique stages, transitions reference declared nodes, **acyclic**, condition syntax parseable.

### 3. Atomic Command Pipeline (Idempotency)
- Every Command carries an idempotency key, actor and digest; **event, projection, idempotency row and outbox commit in one transaction**.
- Replay with same key+digest returns the same result; a different digest conflicts; concurrent duplicates are rejected by a unique constraint.

### 4. Work Items & Leases (Claim/Lease/Fencing)
- Work Items can be claimed by humans, agents, services or robots: claim grants a lease + fencing token; heartbeat renews; expired leases allow reassignment; **stale-token completions are rejected** (prevents double execution).

### 5. Durable Timers
- Timers persist and fire **exactly once** after process restart (atomic fire-and-mark). Supports deadline / duration / business calendar.

### 6. Effect Ledger (Effect Integrity)
- External side effects register an intent first, then record an observed outcome; the same idempotency key never re-executes; unknown outcomes enter reconciliation instead of blind retry.
- This is the Effect Integrity model — not a mythical "exactly-once".

### 7. Recovery & Fault Injection
- **RHEO-Bench 10/10**: B1 replay / B2 crash matrix / B3 duplicate & reorder / B4 branch DAG / B5 timers / B6 worker loss / B7 unknown effects / B8 compensation / B9 migration / B10 revocation.
- Checkpoints (L2 recovery point): event position + folded snapshot; `RebuildProjection` fast path.
- Crash-matrix tests: restart survival, no lost acknowledged commands, projection rebuild, seq conflicts.

### 8. Branches / DAG / Subprocesses
- Token-based activation kernel: unselected branches are marked `Bypassed` and never block completion; AND-joins wait for all dependencies; ambiguous conditions enter Waiting instead of silently branching.
- **Subprocess expansion**: parent-child correlation (`parent_instance_id` + depth), depth limits, budget/authority decay.

### 9. Compensation / Migration / Replan
- Compensation graphs run in reverse order of success, with manual fallback; every action produces a `CompensationExecuted` audit event.
- Migration: dry-run compatibility analysis (additive is compatible; removals/condition changes rejected); running instances migrate online to a new definition version.
- Replan handoff: produce a revised definition → dry-run → migrate, fully idempotent.

### 10. Suspend / Resume
- `ProcessSuspended` / `ProcessResumed` events; step/close commands are rejected while suspended, then continue after resume.

### 11. Evidence Chain & Signatures
- `audit.Build` traces from a business outcome back through: definition → grant → assignment → execution → effect.
- Event signing: optional HMAC; **chained signatures** (each signature covers the previous event) detect reorder/deletion.
- `history --verify` / `audit export` / `GET /api/v1/audit/{id}`.

### 12. Governance & Retention
- Legal hold excludes instances from retention purge; retention cleans closed-instance projections by MaxAge (events preserved); partition registry + instance assignment; `ha_locks` lease locks (HA groundwork).

### 13. Edge Sync
- Outbox commits with events; `sync/delta` (incremental pull) → `sync/ack` (acknowledge) → `sync/pending` (unsent queue); the event stream is the authority, so a lost outbox can be rebuilt.

### 14. Multi-Target Interfaces
- **CLI** (25+ commands): workflow define/validate/diff/import-bpmn/export-bpmn, run open/step/close/suspend/resume/list, checkpoint, migrate/replan, audit, history, serve/watch, sync, partition, bench, dr, auth, ir.
- **HTTP Ops API** (40+ endpoints): instances / steps / suspend-resume / compensate / workflows / work-items / audit / sync / events(SSE) / health / status / metrics.
- **MCP gateway**: `POST /mcp` JSON-RPC 2.0 exposing 8 tools (open_run / step / evidence …) for agents.
- **Worker API**: claim / heartbeat / complete / fail.

### 15. Identity
- `rheo auth`: OIDC Device Flow (RFC 8628) login + whoami.
- `identity.Identity` port (AEGIVELA abstraction): static provider / Keycloak (enterprise).

### 16. Open Architecture (Open-Core)
- A public `runtime` package is the kernel contract surface (Store/Service/Identity/Signer/audit) — ee and SDKs never import `internal/*`.
- **5-language Worker SDKs**: Go (embedded) / Python / TypeScript / Java / Rust; all speak the HTTP Worker API.
- Versioned JSON Schemas: `workflow.schema.json`, `event.schema.json`; OpenAPI: `openapi.yaml`.
- BPMN 2.0 subset import/export (tasks/gateways/events/conditions, round-trip consistent).

---

## Quick Start

```sh
go build -o rheo ./cmd/rheo        # or release/build.ps1
rheo workflow define --file examples/expense.json
rheo run open expense --var amount=500
rheo step enter validate --as alice
rheo step exit validate --as alice
rheo run close --outcome done
rheo serve --addr :8080
```

See [docs/product.en.md](docs/product.en.md) and the Beta manual ([docs/BETA.md](https://github.com/axisrobo/rheovela/blob/master/docs/BETA.md)) for the release criteria.
