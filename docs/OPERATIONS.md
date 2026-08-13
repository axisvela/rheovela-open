# RHEOVELA Operations Guide

> English · [中文](OPERATIONS.zh.md)

This guide covers operating a RHEOVELA node built on the core engine
(`rheo`, `github.com/axisrobo/rheovela`). The `rheovela-open` repository
provides the contracts, Worker SDKs and API references you need to integrate;
the runtime itself is the `rheo` binary.

> **Scope**: single-node local/on-prem operations. Multi-tenant deployments,
> identity and the control plane live in `rheovela-ee` (`rheo-ee serve`).
> For HA/DR limits see [Backup & DR](#backup--dr).

---

## 1. Install

### 1.1 Download a release binary (recommended)

Every semantic-version release publishes 5 cross-compiled targets plus a
`SHA256SUMS` manifest on the GitHub Releases page:

| Asset | Platform |
|-------|----------|
| `rheo-windows-amd64.exe` | Windows amd64 |
| `rheo-linux-amd64` | Linux amd64 |
| `rheo-linux-arm64` | Linux arm64 |
| `rheo-darwin-amd64` | macOS amd64 |
| `rheo-darwin-arm64` | macOS arm64 |
| `SHA256SUMS` | checksum manifest |

Verify the checksum before first run:

```sh
# macOS / Linux
sha256sum -c SHA256SUMS --ignore-missing rheo-linux-amd64
# Windows PowerShell
Get-FileHash -Algorithm SHA256 .\rheo-windows-amd64.exe
```

### 1.2 Build from source

Requires Go 1.25. Build a single binary (no CGO — `modernc.org/sqlite`):

```sh
go build -o rheo ./cmd/rheo
```

or cross-compile all 5 targets with the release script (writes
`dist/SHA256SUMS`):

```powershell
powershell -ExecutionPolicy Bypass -File release/build.ps1
```

Check the version:

```sh
rheo --version   # → rheo v1.0.0-rc.1 (stamped at release)
```

### 1.3 Data directory

State lives in a single SQLite file. Default location is
`~/.proc/proc.db` (WAL mode). Override with `rheo --db <path>` on `serve`
and with `RHEO_DB` on other commands if supported. Back this file up — see
[Backup & DR](#backup--dr).

---

## 2. First run: define, open, step, close

```sh
# Create and select a project
rheo project new quickstart      # → Project #1 created: quickstart
rheo project use 1

# Import a workflow definition (JSON; or `rheo ir tojson <file>.rheo`)
rheo workflow define --file expense.json

# Open a run with variables and an actor
rheo run open expense-reimbursement --var amount=1200 --as alice

# Walk the stages
rheo step enter submit --as alice
rheo step exit submit
rheo step enter dept_approve --as bob
rheo step exit dept_approve

# Inspect, then close
rheo view                       # TUI
rheo history --run <run-id>     # event log
rheo run close --outcome done
```

Every command appends a signed, immutable event; the run status is derived by
folding the event stream — never by mutating state.

---

## 3. Run the HTTP server

`rheo serve` exposes the HTTP Ops API, the Worker API, the MCP gateway and
optional SSE streaming on one mux:

```sh
rheo serve [--addr :8080] [--db <path>] [--verify-key <hex>] \
           [--scheduler-interval <duration>] [--scheduler-off] \
           [--tls-cert <file> --tls-key <file>] [--readonly]
```

| Flag | Description |
|------|-------------|
| `--addr` | listen address (default `:8080`) |
| `--db` | SQLite database path |
| `--verify-key <hex>` | HMAC key for audit signature verification |
| `--scheduler-interval <d>` | expired-timer polling interval (default `1s`) |
| `--scheduler-off` | disable the durable timer scheduler |
| `--tls-cert` / `--tls-key` | serve HTTPS (must be a matched pair) |
| `--readonly` | observation-only: scheduler off, mutating methods rejected (403) |

Liveness and overview:

```sh
curl http://localhost:8080/api/v1/health    # {"status":"ok","db":"ok"}
curl http://localhost:8080/api/v1/status    # {instances, active, done, failed, ...}
curl http://localhost:8080/api/v1/metrics   # {events, instances, timers_pending, ...}
```

Full route reference: [api/README.md](../api/README.en.md) (English) ·
[api/README.md](../api/README.md) (中文). OpenAPI 3.0 spec: `api/openapi.yaml`.

### 3.1 MCP gateway

`POST /mcp` speaks JSON-RPC 2.0 (MCP tools: `open_run`, `enter_step`,
`complete_step`, `fail_step`, `skip_step`, `close_run`, `list_instances`,
`get_evidence`). Useful for wiring agents/assistants to the runtime.

```sh
curl -X POST http://localhost:8080/mcp -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

### 3.2 TLS

Enable HTTPS with a matched `--tls-cert`/`--tls-key` pair (a lone cert or key
is rejected). Terminate TLS at a reverse proxy (nginx/Caddy) in production
and pin the listen address to a loopback/private interface.

### 3.3 Read-only mode

Start with `--readonly` to turn the server into a pure observation surface:
the scheduler is disabled and every `POST`/`PUT`/`PATCH`/`DELETE` is rejected
with `403 {"error":"read-only mode"}`. Use it for read replicas, auditors,
or dashboards that must not mutate state.

---

## 4. Run workers

Work items are claimed and settled over the Worker HTTP API:

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/work-items?instance=<id>` | list ready work items |
| `POST` | `/api/v1/work-items/{id}/claim` | claim → `{token, lease, lease_until}` |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | renew the lease `{token, lease}` |
| `POST` | `/api/v1/work-items/{id}/complete` | complete `{token}` |
| `POST` | `/api/v1/work-items/{id}/fail` | fail `{token, error}` |

`rheovela-open` ships worker SDKs in 5 languages. Each provides a
poll → claim → run → complete/fail loop; the fencing token prevents double
execution when a lease expires and another worker reclaims the item.

```sh
# Go (embedded)     → sdk/worker (WorkStore port; core runtime.WorkerBridge)
# Python            → sdk/python  (stdlib-only, Python 3.9+)
# TypeScript        → sdk/typescript (zero-dep, Node 18+ fetch)
# Java              → sdk/java (JDK 21, zero deps)
# Rust              → sdk/rust (std-only, zero crates, offline build)
```

Example: point any worker at the server address and start its loop:

```python
import worker
w = worker.Worker(worker.Client("http://localhost:8080"), fn, lease="30s")
while True:
    if w.process_once() == 0:
        time.sleep(2)
```

Language-specific setup and code samples:
- [Go](../sdk/worker/README.en.md) · [Python](../sdk/python/README.en.md) ·
  [TypeScript](../sdk/typescript/README.en.md) · [Java](../sdk/java/README.en.md) ·
  [Rust](../sdk/rust/README.en.md)

Run the bundled end-to-end examples:
[examples/README.en.md](../examples/README.en.md).

---

## 5. Monitor & audit

### 5.1 Operational endpoints

| Endpoint | What it tells you |
|----------|-------------------|
| `GET /api/v1/health` | liveness (DB ping) |
| `GET /api/v1/status` | instance counts by status, unsent outbox, partitions, legal holds |
| `GET /api/v1/metrics` | engine counters: events, timers pending, ready work items, unknown effects |
| `GET /api/v1/events?stream=<id>&after_seq=<n>` | SSE event stream (push) |
| `GET /api/v1/sync/delta?stream=<id>&after_seq=<n>` | incremental event snapshot (edge pull) |

### 5.2 Audit

```sh
rheo history --run <run-id> --verify          # per-event [valid]/[INVALID]
rheo audit export <run-id> --format json      # evidence chain as JSON
rheo audit export <run-id> --verify-key <hex> # signature-verified export
```

With `--signing-key <hex>` on writes and `--verify-key <hex>` on the server
(or env `RHEO_SIGNING_KEY`), every event is HMAC-SHA256 signed; chained
signatures let you detect reordering or deletion.

### 5.3 Watch live events

```sh
rheo watch <run-id>            # stream events every 500ms as SSE data: lines
```

---

## 6. Backup & DR

The entire node state is one SQLite file, so DR is a file story.

- **RPO = 0 within the file** — every command commits in one transaction (WAL);
  a crash loses at most an uncommitted transaction.
- **RTO ≈ `rheo dr restore <backup>` + reopen** — a file copy plus restart, no
  replay or index rebuild.

### 3-step restore drill

```sh
# 1) backup and verify
rheo dr backup /safe/proc.db.bak
rheo dr verify /safe/proc.db.bak        # must print OK: true

# 2) simulate loss (stop the node first)
#    del /your/proc.db            (Windows)
#    rm  /your/proc.db            (POSIX)

# 3) restore and reopen
rheo dr restore /safe/proc.db.bak
rheo dr verify /your/proc.db
rheo serve
```

### Scheduled backups

Run backups on a cadence matching your RPO budget, and verify every backup in
the same job:

```sh
rheo dr backup backups/daily.db.bak && rheo dr verify backups/daily.db.bak
```

Keep backups on a separate disk/object store from the live DB. If `verify`
does not print `OK: true`, do not run the node against that file.

> **HA note**: `ha_locks` provides a lease-lock (fencing) primitive, but
> multi-node leader election, replication and automatic failover are **not yet
> implemented**. Treat each node as a standalone durable unit and back it up.

---

## 7. Day-2 operations at a glance

| Task | Command |
|------|---------|
| List runs (filter by status/project) | `rheo run list [--status active] [--project 1]` |
| Show run detail + stage timeline | `rheo run show <run-id>` |
| Suspend / resume a run | `rheo run suspend <run-id> --reason <text>` / `rheo run resume <run-id>` |
| Compensate a closed/failed run | `POST /api/v1/instances/{id}/compensate` |
| List pending sync records | `rheo sync pending` / `rheo sync ack <id>` |
| Manage partitions | `rheo partition list/register/assign/instances` |
| Checkpoint for L2 recovery | `rheo checkpoint create <run-id>` / `rheo checkpoint list <run-id>` |
| Live-migrate a run | `rheo migrate <run-id> --to <workflow-id>` |
| Replan a running instance | `rheo replan <run-id> --stages a,b,c [--dry-run]` |
| Benchmark the node | `rheo bench --concurrency 8 --duration 30s --report bench.json` |
| Validate a definition | `rheo workflow validate <file>` |
| Import / export BPMN | `rheo workflow import-bpmn <file>` / `rheo workflow export-bpmn <file>` |

Full CLI reference: core `docs/api.md` (in `github.com/axisrobo/rheovela`).

---

## 8. Troubleshooting

| Symptom | Check |
|---------|-------|
| `rheo serve` won't start | DB in use (a `serve`/CLI already open)? Locked file from another process? |
| `dr verify` fails | Do **not** restore/run against the file; the backup is incomplete or corrupt |
| Audit shows `[INVALID]` | Signing key mismatch between writer and verifier |
| Worker claims keep failing | Lease too short / another worker holds the item; raise the lease or heartbeats |
| Timer never fires | `--scheduler-off` was passed, or the scheduler interval is long |
| Mutating HTTP calls return 403 | Server started with `--readonly` |
| Connection reset behind TLS | Verify cert/key match; use a reverse proxy for terminations |

If a run's event stream is inconsistent, `rheo history --run <id> --verify`
plus the audit evidence chain is the first place to look — every command is an
immutable, signed event.

---

See also:
- **Developer guide**: [DEVELOPMENT.md](DEVELOPMENT.md) (English) · [中文](DEVELOPMENT.zh.md)
- **Features**: [FEATURES.en.md](FEATURES.en.md) · [中文](FEATURES.md)
- **HTTP API reference**: [api/README.en.md](../api/README.en.md) · [中文](../api/README.md)
- **Product intro**: [product.en.md](product.en.md) · [中文](product.md)
