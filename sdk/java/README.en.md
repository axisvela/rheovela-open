> English · [中文](README.md)

# Rheovela Java Worker SDK (JDK 21, zero dependencies)

Plain-Java worker SDK for the Rheovela Worker HTTP API (`rheo serve`).
Uses only `java.net.http.HttpClient` from the JDK standard library (`java.base`) — no external dependencies.

## API flow

- `poll(instanceId)` — GET `/api/v1/work-items[?instance=]`
- `claim(id, lease)` — POST `/api/v1/work-items/{id}/claim` → lease token
- `complete(id, token)` — POST `/api/v1/work-items/{id}/complete`
- `fail(id, token, error)` — POST `/api/v1/work-items/{id}/fail`
- `processOnce(fn, lease)` — poll → claim → fn(item, token) → complete/fail → processed count

A non-2xx response from any call throws `Worker.WorkerException(status, body)`.
Items that cannot be claimed (already claimed by another worker) are skipped.

## Build & run

```powershell
javac -d out sdk/java/worker/Worker.java sdk/java/example/Example.java
java -cp out example.Example
```

Expected output:

```
processed=2
task-1: done
task-2: failed
```

## Usage

```java
import worker.Worker;

Worker w = new Worker("http://localhost:8080");

int processed = w.processOnce((item, token) -> {
    if (run(item, token)) {
        return Worker.Result.ok();
    }
    return Worker.Result.fail("boom");
}, "30s");

// low-level API
List<Worker.WorkItem> items = w.poll("instance-1");
String token = w.claim(items.get(0).id(), "30s");
w.complete(items.get(0).id(), token);   // or w.fail(id, token, "error")
```
