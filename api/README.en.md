> English · [中文](README.md)

# HTTP Ops API — Reference

RHEOVELA core's single-node HTTP Ops API is exposed by `rheo serve`
(`internal/server`, core `docs/api.md`). All routes are mounted under `/api/v1` and
return `application/json`.

> Purpose: local operations / integration tooling surface — **no TLS / auth shim**.
> Multi-tenancy, authentication and the control plane live in `rheovela-ee`
> (`X-Tenant-ID` middleware + tenant-scoped service).

## Starting the server

```sh
rheo serve [--addr :8080] [--db <path>] [--verify-key <hex>] [--scheduler-interval <duration>] [--scheduler-off]
```

| Flag | Description |
|------|-------------|
| `--addr` | listen address (default `:8080`) |
| `--db` | SQLite database path (default `~/.proc/proc.db`) |
| `--verify-key <hex>` | HMAC verification key: signature verification for audit endpoints (`GET /api/v1/audit/{id}`) |
| `--scheduler-interval <duration>` | polling interval for expired timers (default `1s`) |
| `--scheduler-off` | disable the scheduler loop |

The write path shares the same `application.Service` as the CLI, so runs opened over
HTTP are atomically consistent with the `run_contexts` projection and protected by
`idempotency_key` deduplication.

## Route overview

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/instances?project=<id>` | list process instances (optionally filtered by project) |
| `POST` | `/api/v1/instances` | open a run |
| `GET` | `/api/v1/instances/{id}` | get a single instance |
| `POST` | `/api/v1/instances/{id}/steps` | step actions (enter/complete/fail/skip/assign) |
| `GET` | `/api/v1/audit/{id}` | instance evidence chain (with signature verification) |

### Error format

All errors return JSON: `{"error": "<message>"}`. `404` means the instance does not
exist, `400` means an invalid request body / invalid action / invalid state
transition, `500` means a server error.

---

## `GET /api/v1/instances`

Lists process instances. The `project` query parameter is optional; when omitted, all
instances are returned.

**Response 200**: `ProcessInstance[]` (ordered by `started_at` descending)

```json
[
  {
    "instance_id": "R-<uuid>",
    "project_id": "1",
    "definition_id": "PurchaseApproval",
    "definition_version": 1,
    "status": "active",
    "opened_by": "cli-user",
    "variables": {"amount": "1500"},
    "depth": 0,
    "started_at": "2026-08-07T12:00:00Z"
  }
]
```

`ProcessInstance` fields:

| Field | Type | Description |
|------|------|-------------|
| `instance_id` | string | instance ID (stream ID) |
| `project_id` | string | owning project |
| `definition_id` | string | workflow definition ID |
| `definition_version` | int | definition version pinned by the run |
| `status` | string | `active` / `done` / `failed` / `cancelled`, etc. |
| `opened_by` | string | actor who opened the run |
| `variables` | map | run condition variables (`--var`) |
| `parent_instance_id` | string\|null | parent instance of a subprocess |
| `depth` | int | subprocess depth (0 for parents) |
| `started_at` / `closed_at` | string\|null | timestamps (RFC3339) |

---

## `POST /api/v1/instances`

Opens a run.

**Request body**

```json
{
  "workflow": "PurchaseApproval",
  "project": "1",
  "actor": "cli-user",
  "variables": {"amount": "1500"},
  "idempotency_key": "k-<uuid>"
}
```

| Field | Required | Description |
|------|----------|-------------|
| `workflow` | ✓ | workflow definition ID (400 if missing) |
| `project` |  | project ID |
| `actor` |  | actor ID (written to the event's `actor_id`) |
| `variables` |  | condition variable map |
| `idempotency_key` |  | idempotency key; if omitted the server auto-generates a UUID |

**Response 201**: `ProcessInstance` (see above). Retrying with the same key returns
the existing instance without producing duplicate events.

---

## `GET /api/v1/instances/{id}`

Gets a single instance.

**Response 200**: `ProcessInstance`. **404**: instance does not exist.

---

## `POST /api/v1/instances/{id}/steps`

Performs one step action on an instance.

**Request body**

```json
{
  "action": "enter",
  "stage": "validate",
  "actor": "cli-user",
  "assigned_to": "bob",
  "error": "",
  "idempotency_key": "k-<uuid>"
}
```

| Field | Required | Description |
|------|----------|-------------|
| `action` | ✓ | `enter` \| `complete` \| `fail` \| `skip` \| `assign` |
| `stage` | ✓ | stage ID |
| `actor` |  | acting actor |
| `assigned_to` |  | used by `assign` only: the assignee |
| `error` |  | used by `fail` only: failure reason |
| `idempotency_key` |  | idempotency key; auto-generated if omitted |

**Response 200**: `{"status": "ok"}`

**Errors**: instance not found → 404; invalid action / invalid state transition
(`ErrInvalidTransition`) → 400.

---

## `GET /api/v1/audit/{id}`

Returns the instance's **evidence chain** (`internal/audit`): plan → events →
assignment → execution, plus effect records. When started with `--verify-key`, every
event's signature is verified.

**Response 200**:

```json
{
  "instance_id": "R-<uuid>",
  "definition_id": "PurchaseApproval",
  "definition_version": 1,
  "status": "active",
  "events": [
    {
      "seq": 1,
      "type": "RunOpened",
      "actor_id": "cli-user",
      "wall_time": "2026-08-07T12:00:00Z",
      "signature": "…",
      "signature_valid": true,
      "payload": {"workflow_id": "PurchaseApproval"}
    }
  ],
  "effects": [
    {
      "idempotency_key": "effect:invoice/R-<uuid>",
      "target": "R-<uuid>",
      "state": "applied",
      "evidence": "…"
    }
  ],
  "stage_executions": [
    {
      "stage_id": "validate",
      "assigned_to": "bob",
      "assigned_by": "cli-user",
      "executed_by": ["cli-user"],
      "completed_by": "cli-user",
      "status": "completed"
    }
  ]
}
```

Signature policy: with no verifier, all events are treated as valid; with a verifier,
an event with an empty signature → `signature_valid: false`; an event with a signature
is judged by the `Verify` result. **404**: instance does not exist.

---

## Tenant considerations

- **core (this layer)**: single-tenant, local operations surface; no authentication
  or tenant isolation.
- **`rheovela-ee`**: adds an `X-Tenant-ID` middleware, tenant-scoped services and an
  `{id}` route ownership guard (cross-tenant requests always return 404) on top of
  core. For multi-tenancy, use ee's `rheo-ee serve` instead of this layer.

## Full JSON schemas

Contract types (`WorkflowDefinition` / `Event` / `ProcessInstance`) are defined in
`contracts/types.go` (`schema_version=1`). JSON Schemas (draft-07) live in
`workflow.schema.json` and `event.schema.json`, and can be validated with any JSON
Schema validator or `go test ./api/ -v`; the OpenAPI 3.0 spec for the HTTP routes is
in `openapi.yaml`. Full CLI commands and the Go API are in core `docs/api.md`.
