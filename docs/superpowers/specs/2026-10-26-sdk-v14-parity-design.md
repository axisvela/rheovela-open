# Open SDK v1.4.0 Parity

- Date: 2026-10-26
- Status: Approved design, pending implementation
- Owner: open
- Related: `docs/STATUS.md` SDK matrix, `api/openapi.yaml` Worker API

## 1. Context

The Go worker/client SDK exposes the v1.4.0 durable-work-item surface:
typed results, checkpoint/effect references, approval requests, attempt fields,
cancel, and claimable polling. Python, TypeScript, Java, and Rust currently
expose only claim/heartbeat/complete/fail.

This change aligns all four SDKs with the same HTTP contract.

## 2. Goals

- Add the full v1.4.0 client surface to Python, TypeScript, Java, and Rust.
- Make each Worker callback return a rich `WorkResult` (success/failure/approval).
- Persist result, checkpoint, and effect cursor on successful completion.
- Support request-approval, approve, reject, and cancel.
- Expose work-item attempts/max-attempts/last-error/approval fields.
- Keep Java/Rust dependency-free; arbitrary JSON is carried as raw JSON text.
- Add per-language tests and CI coverage (including TypeScript).

## 3. Non-goals

- .NET SDK.
- Harmovela adapter.
- Full JSON parser/object model for Java/Rust.
- New server endpoints or retry endpoint (retry remains server-driven via
  failed/attempts/max_attempts).
- Token exchange/auth changes.

## 4. Shared Contract

WorkItem fields:

```text
id, instance_id, activity_id, state,
result, checkpoint_ref, effect_cursor,
attempts, max_attempts, last_error,
approval
```

Approval fields:

```text
state, pending_stage, final_args, reason,
requested_by, requested_at, decided_by, decided_at, comment,
checkpoint_ref, effect_cursor
```

WorkResult:

```text
status: "success" | "failure" | "approval"
error
outcome
result
checkpoint_ref
effect_cursor
approval (ApprovalRequest)
```

ApprovalRequest:

```text
actor, reason, final_args, checkpoint_ref, effect_cursor
```

Client methods (language-idiomatic names):

| Operation | HTTP |
| --- | --- |
| list/poll (global claimable when no instance) | `GET /api/v1/work-items[?instance=]` |
| get | `GET /api/v1/work-items/{id}` |
| claim | `POST .../{id}/claim` |
| heartbeat | `POST .../{id}/heartbeat` |
| complete (token-only compatible) | `POST .../{id}/complete` |
| complete with result/cursors | `POST .../{id}/complete` |
| request approval | `POST .../{id}/request-approval` |
| approve | `POST .../{id}/approve` |
| reject | `POST .../{id}/reject` |
| cancel | `POST .../{id}/cancel` |
| fail | `POST .../{id}/fail` |

Retry: no client endpoint. `fail` lets the server requeue within
`attempts < max_attempts`; WorkItem exposes the counters and `last_error`.

## 5. Worker Behavior

All SDK workers take a callback returning `WorkResult` (breaking change from
the old success/failure tuple):

- `success`: call complete with result/checkpoint/effect when present.
- `failure`: call fail with the error.
- `approval`: call request-approval; missing approval request fails the item.
- Heartbeat continues during the callback where the language SDK supports it
  (Python/TS/Java/Rust may keep the existing synchronous model unless already
  present).
- Claim conflicts/lease loss are skipped/failed as today.

## 6. Language-Specific Mapping

### Python

- `@dataclass WorkItem`, `Approval`, `WorkResult`, `ApprovalRequest`.
- `Client.get_work_item`, `complete(..., result=..., checkpoint_ref=..., effect_cursor=...)`,
  `request_approval`, `approve`, `reject`, `cancel`.
- `Worker` callback returns `WorkResult`; helpers
  `WorkResult.success/failure/approval`.
- Tests: `unittest` fake HTTP server.

### TypeScript

- Interfaces `WorkItem`, `Approval`, `WorkResult`, `ApprovalRequest`.
- `Client.getWorkItem`, `complete` overload/options, `requestApproval`,
  `approve`, `reject`, `cancel`.
- `WorkFn` returns `WorkResult`; helpers `success`, `failure`, `requestApproval`.
- Tests: `node --test --experimental-strip-types` (Node 22+) in CI.

### Java

- `record WorkItem(..., String result, String checkpointRef, String effectCursor,
  int attempts, int maxAttempts, String lastError, Approval approval)`.
- `record Result(String status, String error, String outcome, String result,
  String checkpointRef, String effectCursor, ApprovalRequest approval)` with
  factories `ok`, `success`, `fail`, `approval`.
- Raw `result`/`finalArgs` are JSON strings embedded as raw JSON in requests.
- Client methods `get`, `complete(...)`, `requestApproval`, `approve`, `reject`,
  `cancel`.
- Example/fake server updated to exercise result, approval, cancel and assert.

### Rust

- Structs `WorkItem`, `Approval`, `WorkResult`, `ApprovalRequest`.
- Raw `result`/`final_args` as `String` JSON.
- `Client`/`Worker` methods `get`, `complete_with_result`,
  `request_approval`, `approve`, `reject`, `cancel`.
- `process_once` callback returns `WorkResult`; update `main.rs` fake server and
  add `#[cfg(test)]` coverage.
- CI runs `cargo test --offline` and `cargo run --offline`.

## 7. Testing and CI

- Python: compile + unittest.
- TypeScript: new CI job with Node 22 and `node --test --experimental-strip-types`.
- Java: compile + run example assertions.
- Rust: `cargo test --offline` + `cargo run --offline`.
- Go SDK remains unchanged and continues to pass.

## 8. Backward Compatibility

- The Worker callback signature changes in Python/TS/Java/Rust; examples and
  tests are updated. This is an intentional SDK breaking change for the v1.4.0
  feature line.
- HTTP API is unchanged.
- Go SDK remains the reference implementation.

## 9. Acceptance Criteria

1. All four SDKs expose get/list/claim/heartbeat/complete-with-result/
   request-approval/approve/reject/cancel/fail.
2. WorkItem exposes result/checkpoint/effect/attempts/max/last-error/approval.
3. Workers handle success/failure/approval rich results.
4. Java/Rust stay dependency-free and transport arbitrary JSON as raw text.
5. Tests pass for each language; CI runs all four (TypeScript included).
6. Docs/STATUS/ROADMAP reflect full v1.4.0 parity.

## 10. Out of Scope

- .NET SDK.
- Harmovela adapter.
- Full JSON object model in Java/Rust.
- Server-side retry API changes.
