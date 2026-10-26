> English · [中文](README.md)

# Java Worker SDK — `sdk/java`

Stdlib-only HTTP client for the core Worker API exposed by `rheo serve` (see
`api/README.md` for the contract). Uses `java.net.http.HttpClient`; no external
dependencies. Built for JDK 21 (source-compatible with JDK 11+).

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

Errors are returned as `{"error": "..."}` with a non-2xx status and raised as
`Worker.WorkerException(status, body)`.

## Types

`WorkItem` parses `id`, `instance_id`, `activity_id`, `state`, `result`,
`checkpoint_ref`, `effect_cursor`, `attempts`, `max_attempts`, `last_error`, and
the nested `approval` (`state`, `pending_stage`, `final_args`, `reason`,
`requested_by`, `requested_at`, `decided_by`, `decided_at`, `comment`,
`checkpoint_ref`, `effect_cursor`). `result` and `final_args` are **raw JSON
strings** embedded as-is in request bodies, never re-encoded.

`Result` is the worker callback outcome, with status `success`, `failure`, or
`approval`:

```java
Worker.Result.ok();                              // legacy; success without a result
Worker.Result.success("{\"ok\":true}", "ckpt-1", "cur-1");
Worker.Result.fail("boom");
Worker.Result.approval(new Worker.ApprovalRequest(
        "agent", "over limit", "{\"amount\":10}", "", ""));
```

`ApprovalRequest` has `actor`, `reason`, `final_args`, `checkpoint_ref`, and
`effect_cursor`.

## Client

```java
import java.util.List;
import worker.Worker;

Worker w = new Worker("http://127.0.0.1:8080");

List<Worker.WorkItem> items = w.poll(null);     // claimable items globally
List<Worker.WorkItem> filtered = w.poll("inst-1");
Worker.WorkItem item = w.get("wi1");            // includes approval/result

String token = w.claim("wi1", "30s");

// Complete with a raw JSON result and durable-execution references.
w.complete("wi1", token, "{\"ok\":true}", "ckpt-1", "cur-1");

// Token-only completion remains compatible.
w.complete("wi1", token);

w.requestApproval("wi1", token, new Worker.ApprovalRequest(
        "agent", "review", null, "ckpt-1", "cur-1"));
w.approve("wi1", "human", "ok", "{\"approved\":true}");
w.reject("wi2", "human", "no");
w.cancel("wi3");
w.fail("wi4", token, "boom");
```

`complete` only sends non-empty optional fields (`result`, `checkpoint_ref`,
`effect_cursor`), so a token-only body keeps working with older servers.

## Worker loop

`processOnce` wraps polling, claiming, executing and settling into one pass. The
callback returns a `Result`:

```java
int processed = w.processOnce((item, token) -> {
    try {
        if ("dangerous".equals(item.activity_id())) {
            return Worker.Result.approval(new Worker.ApprovalRequest(
                    "agent", "needs review", null, null, null));
        }
        return Worker.Result.success("{\"ok\":true}", "ckpt-1", "cur-1");
    } catch (Exception e) {
        return Worker.Result.fail(e.getMessage());
    }
}, "30s");
```

- `success`: complete with `result`, `checkpoint_ref`, `effect_cursor`.
- `failure` (or unknown status): fail with `error` (default `"failure"`).
- `approval`: request approval; a missing `approval` request fails the item.
  Missing checkpoint/effect references on the `ApprovalRequest` are filled from
  the `Result`.
- Claim conflicts (already claimed) are skipped.
- A stale/expired token on settle is silently ignored (`WorkerException`).

## Build & run

```powershell
javac -d out sdk/java/worker/Worker.java sdk/java/example/Example.java
java -cp out example.Example
```

Expected output:

```
ok
```

`Example` starts an in-process fake Worker API, exercises success-with-result,
request-approval, approve/reject/cancel, and a single-item `get`, then compares
the recorded endpoints and bodies against the expected sequence; it throws on a
mismatch (CI runs it directly).
