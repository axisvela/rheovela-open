> English · [中文](README.zh.md)

# Rheovela-open

## What is RHEOVELA

RHEOVELA is a **Dynamic Process & Durable Workflow Platform**.

**The problem**: More and more long-running work in an enterprise is carried forward by agents, humans and robots together, spanning minutes, hours, days or months. Yet agent sessions and individual workers are ephemeral — a model process exiting, a network outage, a worker crash, a shift change or a system upgrade can all lose, duplicate or render unexplainable the work "in progress". Traditional BPM modeling is too static to absorb dynamic agent plans, while a bare agent loop has no notion of long-lived responsibility, recovery or audit.

**The answer**: RHEOVELA materializes governed Capability Plans into **persistent, recoverable, approvable, migratable and auditable Process Instances** — it owns "how long-running work reliably flows", not open-ended planning, and not a single agent's inference loop.

- **Event sourcing + deterministic kernel**: consistent state is recovered from the event stream after any crash
- **Atomic idempotent command pipeline**: restarts, duplicates and reordering never cause duplicate business effects
- **Work Items unify humans / agents / services / robots**: claim / lease / fencing prevents double execution
- **Evidence chain + signatures**: who was assigned, who executed, under what grant, what effects — fully auditable
- **Compensation / migration / subprocess / suspend-resume / edge sync / checkpoint / legal hold**

## What this repository is

**rheovela-open** is the **public release layer of RHEOVELA** (Apache-2.0, current `v1.0.0-rc.1`): versioned contracts, 5-language Worker SDKs, the HTTP Worker API, OpenAPI spec and example workflows. Full feature list in [docs/FEATURES.en.md](docs/FEATURES.en.md).

- Core (AGPL-3.0): https://github.com/axisrobo/rheovela
- Enterprise: https://github.com/axisrobo/rheovela-ee

## Layout

| Path | Content |
|------|---------|
| `contracts/` | Versioned event/command/process schemas (authoritative contract, `schema_version=1`) |
| `sdk/` | Worker SDKs (5 languages) — [Go](sdk/worker/) · [Python](sdk/python/) · [TypeScript](sdk/typescript/) · [Java](sdk/java/) · [Rust](sdk/rust/) |
| `api/` | HTTP API reference — [README (Ops API)](api/README.md) · [OpenAPI 3.0 spec](api/openapi.yaml) · [workflow.schema.json](api/workflow.schema.json) · [event.schema.json](api/event.schema.json) |
| `examples/` | Example workflows (RHEO IR DSL) — [README](examples/README.md) |
| `docs/` | **Features** — [FEATURES.en.md](docs/FEATURES.en.md) ([中文](docs/FEATURES.md)) · Product intro — [product.en.md](docs/product.en.md) ([中文](docs/product.md)) |

## Quick start (Worker SDK)

Talk to the core Worker HTTP API (`rheo serve`; routes documented in
[api/README.md](api/README.md) and [api/openapi.yaml](api/openapi.yaml)) from
any language SDK:

- Go: `sdk/worker/` (`WorkStore` port + `Worker` loop)
- Python: `sdk/python/` (stdlib-only, `worker.Client` / `worker.Worker`)
- TypeScript: `sdk/typescript/` (zero-dependency, Node 18+ `fetch`, `Client` / `Worker`)
- Java: `sdk/java/` (JDK 21, zero deps `java.net.http.HttpClient`, `Worker`)
- Rust: `sdk/rust/` (std-only, zero external crates, `Client` / `Worker`)

```ts
import { Client, Worker } from "./sdk/typescript/worker.ts";
const client = new Client("http://127.0.0.1:8080");
const w = new Worker(client, async (item) => {
  // do the work...
  return { status: "success" };
});
await w.processOnce(); // poll → claim → fn → complete/fail
```

## Verification

```sh
go test ./...                      # Go SDK tests
python -m unittest discover -s sdk/python -v   # Python SDK tests
node --experimental-transform-types --test sdk/typescript/worker.test.ts   # TypeScript SDK tests
javac -d out sdk/java/worker/Worker.java sdk/java/example/Example.java && java -cp out example.Example   # Java SDK
(cd sdk/rust && cargo build --offline && cargo run --offline)             # Rust SDK
```

CI (`.github/workflows/ci.yml`) runs 4 jobs: **go** / **python** / **java** / **rust**.
