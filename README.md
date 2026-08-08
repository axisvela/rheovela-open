# Rheovela-open

RHEOVELA 对外发布层（Apache-2.0，当前 **v0.9.0-beta**）：产品介绍、版本化 contracts、SDK（5 种语言：Go/Python/TypeScript/Java/Rust）、API reference 与 examples。

- 仓库：https://github.com/axisrobo/rheovela-open
- 内核（AGPL）：https://github.com/axisrobo/rheovela
- 企业版：https://github.com/axisrobo/rheovela-ee

## 目录

| 目录 | 内容 |
|------|------|
| `contracts/` | 版本化 event/command/process schemas（权威契约，`schema_version=1`） |
| `sdk/` | worker SDK（5 种语言）— [Go](sdk/worker/) · [Python](sdk/python/) · [TypeScript](sdk/typescript/) · [Java](sdk/java/) · [Rust](sdk/rust/) |
| `api/` | HTTP API reference — [README（Ops API）](api/README.md) · [OpenAPI 3.0 规范](api/openapi.yaml) · [workflow.schema.json](api/workflow.schema.json) · [event.schema.json](api/event.schema.json) |
| `examples/` | 示例流程（RHEO IR DSL）— [README](examples/README.md) |
| `docs/` | **功能与特性** — [FEATURES.md](docs/FEATURES.md)（[English](docs/FEATURES.en.md)）· 产品介绍 — [product.md](docs/product.md) |

## 快速开始（Worker SDK）

用任意语言 SDK 对接 core 的 Worker HTTP API（`rheo serve`，路由见
[api/README.md](api/README.md) 与 [api/openapi.yaml](api/openapi.yaml)）：

- Go：`sdk/worker/`（`WorkStore` 端口 + `Worker` 循环）
- Python：`sdk/python/`（stdlib-only，`worker.Client` / `worker.Worker`）
- TypeScript：`sdk/typescript/`（零依赖，Node 18+ `fetch`，`Client` / `Worker`）
- Java：`sdk/java/`（JDK 21，零依赖 `java.net.http.HttpClient`，`Worker`）
- Rust：`sdk/rust/`（std-only，零外部 crate，`Client` / `Worker`）

```ts
// sdk/typescript 用法
import { Client, Worker } from "./sdk/typescript/worker.ts";
const client = new Client("http://127.0.0.1:8080");
const w = new Worker(client, async (item) => {
  // do the work...
  return { status: "success" };
});
await w.processOnce(); // poll → claim → fn → complete/fail
```

---

# Rheovela-open

The public release layer of RHEOVELA (Apache-2.0): product intro, versioned
contracts, SDKs (Go/Python/TypeScript/Java/Rust), API reference and examples.

- Repo: https://github.com/axisrobo/rheovela-open
- Core (AGPL): https://github.com/axisrobo/rheovela
- Enterprise: https://github.com/axisrobo/rheovela-ee

## Layout

| Path | Content |
|------|---------|
| `contracts/` | Versioned event/command/process schemas (authoritative contract, `schema_version=1`) |
| `sdk/` | Worker SDKs (5 languages) — [Go](sdk/worker/) · [Python](sdk/python/) · [TypeScript](sdk/typescript/) · [Java](sdk/java/) · [Rust](sdk/rust/) |
| `api/` | HTTP API reference — [README (Ops API)](api/README.md) · [OpenAPI 3.0 spec](api/openapi.yaml) · [workflow.schema.json](api/workflow.schema.json) · [event.schema.json](api/event.schema.json) |
| `examples/` | Example workflows (RHEO IR DSL) — [README](examples/README.md) |
| `docs/` | **Features** — [FEATURES.md](docs/FEATURES.md) ([中文](docs/FEATURES.md) / [English](docs/FEATURES.en.md)) · Product intro — [product.md](docs/product.md) |

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

CI（`.github/workflows/ci.yml`）跑 4 个 job：**go** / **python** / **java** / **rust**。
