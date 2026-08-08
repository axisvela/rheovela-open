> English · [中文](README.md)

# Rheovela Rust Worker SDK (std-only, zero dependencies)

Plain-Rust worker SDK for the Rheovela Worker HTTP API (`rheo serve`).
Uses only the Rust standard library — a minimal HTTP/1.1 client built on
`std::net::TcpStream` and `std::io`. No external crates, so builds work
offline (`cargo build --offline`).

## API flow

- `poll(instance)` — GET `/api/v1/work-items[?instance=]`
- `claim(id, lease)` — POST `/api/v1/work-items/{id}/claim` → fencing token
- `complete(id, token)` — POST `/api/v1/work-items/{id}/complete`
- `fail(id, token, error)` — POST `/api/v1/work-items/{id}/fail`
- `process_once(fn, lease)` — poll → claim → fn(item) → complete/fail → processed count

A non-2xx response from any call returns `WorkerError { status, message }`.
Items that cannot be claimed (already claimed by another worker) are skipped.

## Build & run

```powershell
cd sdk/rust
cargo build --offline
cargo run --offline
```

Expected output:

```
processed=2
task-1: done
task-2: failed
```

## Usage

```rust
use rheovela_worker::{Client, Worker};

let worker = Worker::new(Client::new("http://localhost:8080"));

let processed = worker.process_once(
    |item| {
        if run(item) {
            Ok(())
        } else {
            Err("boom".to_string())
        }
    },
    "30s",
)?;

// low-level API
let items = worker.poll(Some("instance-1"))?;
let token = worker.claim(&items[0].id, "30s")?;
worker.complete(&items[0].id, &token)?; // or worker.fail(&id, &token, "error")
```
