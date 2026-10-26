> English · [中文](README.md)

# Rust Worker SDK — `sdk/rust`

Stdlib-only HTTP client for the core Worker API exposed by `rheo serve` (see
`api/README.md` for the contract). Uses only `std::net::TcpStream` and
`std::io`; no external crates, so builds work offline
(`cargo build --offline`).

## Worker API routes

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/work-items[?instance=<id>]` | list work items (global poll = claimable items) |
| `GET` | `/api/v1/work-items/{id}` | get one work item with result/approval |
| `POST` | `/api/v1/work-items/{id}/claim` | claim (returns fencing token) |
| `POST` | `/api/v1/work-items/{id}/complete` | complete with optional result/cursors |
| `POST` | `/api/v1/work-items/{id}/request-approval` | request human approval |
| `POST` | `/api/v1/work-items/{id}/approve` | approve a pending item |
| `POST` | `/api/v1/work-items/{id}/reject` | reject a pending item |
| `POST` | `/api/v1/work-items/{id}/cancel` | cancel an item |
| `POST` | `/api/v1/work-items/{id}/fail` | fail with error message |

Any non-2xx response returns `WorkerError { status, message }`.

## Types

`WorkItem` parses `id`, `instance_id`, `activity_id`, `state`, `result`,
`checkpoint_ref`, `effect_cursor`, `attempts`, `max_attempts`, `last_error`, and
the nested `approval` (`state`, `pending_stage`, `final_args`, `reason`,
`requested_by`, `requested_at`, `decided_by`, `decided_at`, `comment`,
`checkpoint_ref`, `effect_cursor`). `result` and `final_args` are **raw JSON
strings** embedded as-is in request bodies, never re-encoded.

`WorkResult` is the worker callback outcome; `WorkStatus` is `Success`,
`Failure`, or `Approval`:

```rust
use rheovela_worker::{ApprovalRequest, WorkResult};

WorkResult::success(); // successful completion without a result
WorkResult::success_with(
    Some(r#"{"ok":true}"#.to_string()),
    Some("ckpt-1".to_string()),
    Some("cur-1".to_string()),
);
WorkResult::failure("boom");
WorkResult::approval(ApprovalRequest {
    actor: "agent".to_string(),
    reason: "over limit".to_string(),
    final_args: Some(r#"{"amount":10}"#.to_string()),
    checkpoint_ref: String::new(),
    effect_cursor: String::new(),
});
```

`ApprovalRequest` has `actor`, `reason`, `final_args`, `checkpoint_ref`, and
`effect_cursor`.

## Client

```rust
use rheovela_worker::{ApprovalRequest, Client, Worker};

let w = Worker::new(Client::new("http://127.0.0.1:8080"));

let items = w.poll(None)?;              // claimable items globally
let filtered = w.poll(Some("inst-1"))?;
let item = w.get("wi1")?;               // includes approval/result

let token = w.claim("wi1", "30s")?;

// Complete with a raw JSON result and durable-execution references.
w.complete_with_result(
    "wi1", &token,
    Some(r#"{"ok":true}"#),
    Some("ckpt-1"),
    Some("cur-1"),
)?;

// Token-only completion remains compatible.
w.complete("wi1", &token)?;

w.request_approval("wi1", &token, &ApprovalRequest {
    actor: "agent".to_string(),
    reason: "review".to_string(),
    final_args: None,
    checkpoint_ref: "ckpt-1".to_string(),
    effect_cursor: "cur-1".to_string(),
})?;
w.approve("wi1", "human", "ok", Some(r#"{"approved":true}"#))?;
w.reject("wi2", "human", "no")?;
w.cancel("wi3")?;
w.fail("wi4", &token, "boom")?;
```

`complete_with_result` only sends non-empty optional fields (`result`,
`checkpoint_ref`, `effect_cursor`), so a token-only body keeps working with
older servers.

## Worker loop

`process_once` wraps polling, claiming, executing and settling into one pass.
The callback returns a `WorkResult`:

```rust
use rheovela_worker::{ApprovalRequest, Client, WorkResult, Worker};

let w = Worker::new(Client::new("http://127.0.0.1:8080"));

let processed = w.process_once(|item| {
    if item.activity_id == "dangerous" {
        return WorkResult::approval(ApprovalRequest {
            actor: "agent".to_string(),
            reason: "needs review".to_string(),
            final_args: None,
            checkpoint_ref: String::new(),
            effect_cursor: String::new(),
        });
    }
    WorkResult::success_with(
        Some(r#"{"ok":true}"#.to_string()),
        Some("ckpt-1".to_string()),
        Some("cur-1".to_string()),
    )
}, "30s")?;
```

- `success`: complete with `result`, `checkpoint_ref`, `effect_cursor`; when
  none are present it falls back to token-only `complete`.
- `failure`: fail with `error` (default `"failure"`).
- `approval`: request approval; a missing `approval` request fails the item.
  Missing checkpoint/effect references on the `ApprovalRequest` are filled from
  the `WorkResult`.
- Claim conflicts (already claimed) are skipped.
- A stale/expired token on settle is silently ignored (`WorkerError`).

## Build & run

```powershell
cd sdk/rust
cargo test --offline
cargo run --offline
```

Expected output:

```
processed=2
ok
```

`main.rs` starts an in-process fake Worker API, exercises success-with-result,
request-approval, approve/reject/cancel, and a single-item `get`, then compares
the recorded endpoints and bodies against the expected sequence; it panics on a
mismatch (CI runs it directly, so the job fails).
