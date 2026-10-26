> English · [中文](README.md)

# Python Worker SDK — `sdk/python`

Stdlib-only (`urllib` / `json`) HTTP client for the core Worker API exposed by
`rheo serve` (see `api/README.md` and the Worker API routes below). No
third-party dependencies; works with Python 3.9+.

## Worker API routes

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/work-items[?instance=<id>]` | list work items (global poll = claimable items) |
| `GET` | `/api/v1/work-items/{id}` | get one work item with result/approval |
| `POST` | `/api/v1/work-items/{id}/claim` | claim (returns fencing token) |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | renew lease |
| `POST` | `/api/v1/work-items/{id}/complete` | complete with optional result/cursors |
| `POST` | `/api/v1/work-items/{id}/request-approval` | request human approval |
| `POST` | `/api/v1/work-items/{id}/approve` | approve a pending item |
| `POST` | `/api/v1/work-items/{id}/reject` | reject a pending item |
| `POST` | `/api/v1/work-items/{id}/cancel` | cancel an item |
| `POST` | `/api/v1/work-items/{id}/fail` | fail with error message |

Errors are returned as `{"error": "..."}` with a 4xx status and raised as
`worker.WorkerError`.

## Types

`WorkItem` parses `id`, `instance_id`, `activity_id`, `state`, `result`,
`checkpoint_ref`, `effect_cursor`, `attempts`, `max_attempts`, `last_error`, and
the nested `approval` (`state`, `pending_stage`, `final_args`, `reason`,
`requested_by`, `requested_at`, `decided_by`, `decided_at`, `comment`,
`checkpoint_ref`, `effect_cursor`).

```python
import worker

item = worker.WorkItem(
    id="wi1", instance_id="i1", activity_id="a1", state="ready",
)
```

`WorkResult` carries the worker callback outcome. Use the helpers:

```python
worker.WorkResult.success(result=None, checkpoint_ref="", effect_cursor="")
worker.WorkResult.failure("boom")
worker.WorkResult.request_approval(
    worker.ApprovalRequest(actor="agent", reason="over limit")
)
```

`ApprovalRequest` has `actor`, `reason`, `final_args`, `checkpoint_ref`, and
`effect_cursor`.

## Client

```python
import worker

client = worker.Client("http://127.0.0.1:8080")

items = client.poll()                          # claimable items globally
items = client.poll(instance_id="inst-1")      # filter by instance
item = client.get_work_item("wi1")             # includes approval/result

claim = client.claim("wi1", lease="30s")
token = claim["token"]
client.heartbeat("wi1", token)

# Complete with a typed result and durable-execution references.
client.complete("wi1", token, result={"ok": True},
                checkpoint_ref="ckpt-1", effect_cursor="cur-1")

# Token-only completion remains compatible.
client.complete("wi1", token)

client.request_approval("wi1", token,
                        worker.ApprovalRequest(actor="agent", reason="review"))
client.approve("wi1", "human", comment="ok", final_args={"approved": True})
client.reject("wi2", "human", "no")
client.cancel("wi3")
client.fail("wi4", token, "boom")
```

`complete` only sends non-empty optional fields (`result`, `checkpoint_ref`,
`effect_cursor`), so a token-only body keeps working with older servers.

## Worker loop

`Worker` wraps polling, claiming, executing and settling into one pass. The
callback **must return a `WorkResult`** (breaking change from the tuple form):

```python
import worker

def fn(item):
    try:
        if item.activity_id == "dangerous":
            return worker.WorkResult.request_approval(
                worker.ApprovalRequest(actor="agent", reason="needs review")
            )
        return worker.WorkResult.success(
            result={"ok": True}, checkpoint_ref="ckpt-1",
        )
    except Exception as exc:
        return worker.WorkResult.failure(str(exc))

client = worker.Client("http://127.0.0.1:8080")
w = worker.Worker(client, fn, lease="30s")

while True:
    if w.process_once() == 0:   # nothing ready, back off
        import time
        time.sleep(2)
```

- `success` completes with `result`, `checkpoint_ref`, `effect_cursor`.
- `failure` (or any unknown status) fails with `error` (default `"failure"`).
- `approval` requests approval; if `approval` is missing the item is failed.
  Missing checkpoint/effect refs are copied from the `WorkResult` onto the
  `ApprovalRequest`.
- Claim conflicts (already claimed, 409) are skipped.
- A stale/expired token on settle is silently ignored (`WorkerError`).
- Filter by instance with `w.process_once(instance_id)`.

## Tests

```sh
python -m py_compile sdk/python/worker.py
python -m unittest discover -s sdk/python -v
```
