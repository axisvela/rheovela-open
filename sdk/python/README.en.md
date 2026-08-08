> English · [中文](README.md)

# Python Worker SDK — `sdk/python`

Stdlib-only (`urllib` / `json`) HTTP client for the core Worker API exposed by
`rheo serve` (see `api/README.md` and the Worker API routes below). No
third-party dependencies; works with Python 3.9+.

## Worker API routes

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/work-items?instance=<id>` | list work items |
| `POST` | `/api/v1/work-items/{id}/claim` | claim (returns fencing token) |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | renew lease |
| `POST` | `/api/v1/work-items/{id}/complete` | complete (token-guarded) |
| `POST` | `/api/v1/work-items/{id}/fail` | fail with error message |

Errors are returned as `{"error": "..."}` with a 4xx status and raised as
`worker.WorkerError`.

## Client

```python
import worker

client = worker.Client("http://127.0.0.1:8080")

items = client.poll()                      # list of work items
claim = client.claim(items[0]["id"], lease="30s")
token = claim["token"]
client.heartbeat(items[0]["id"], token)
client.complete(items[0]["id"], token)     # or client.fail(id, token, "boom")
```

## Worker loop

`Worker` wraps polling, claiming, executing and settling into one pass:

```python
import worker

def fn(item):
    # do the work; return ("success", None) or ("failure", "reason")
    return ("success", None)

client = worker.Client("http://127.0.0.1:8080")
w = worker.Worker(client, fn, lease="30s")

while True:
    if w.process_once() == 0:   # nothing ready, back off
        import time
        time.sleep(2)
```

- Items whose `claim` conflicts (already claimed, 409) are skipped.
- A stale/expired token on settle is silently ignored (`WorkerError`).
- Filter by instance with `w.process_once(instance_id)`.

## Tests

```sh
python -m py_compile sdk/python/worker.py
python -m unittest discover -s sdk/python -v
```
