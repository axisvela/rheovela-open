# Rheovela-open

## RHEOVELA 是什么

RHEOVELA 是一个 **Dynamic Process & Durable Workflow Platform（动态流程与持久化工作流平台）**。

**解决的问题**：企业里越来越多的长期工作由 Agent、人类、机器人共同推进，跨越分钟、小时、天甚至月。但 Agent 会话和单个 worker 都是短命的——模型进程退出、网络中断、worker 崩溃、人员换班或系统升级，都可能让"进行中的工作"丢失、重复或无法解释。传统 BPM 建模太静态，吸收不了动态的 Agent 计划；而裸的 Agent loop 又不理解长期责任、恢复与审计。

**RHEOVELA 的答案**：把治理后的 Capability Plan 物化为**可持久、可恢复、可审批、可迁移、可审计的 Process Instance**，负责"长期工作如何可靠流转"——而不是做开放式规划，也不是跑单个 Agent 的推理循环。

- **事件溯源 + 确定性内核**：任何崩溃后都能从事件流恢复一致状态
- **原子幂等命令管线**：重启、重复、乱序不产生重复业务效果
- **Work Item 统一人 / Agent / 服务 / 机器人**：claim / lease / fencing 防止双执行
- **证据链 + 签名**：谁被指派、谁执行、依据什么授权、产生了什么 effect，全程可审计
- **补偿 / 迁移 / 子流程 / 挂起恢复 / 边缘同步 / checkpoint / legal hold**

## 这个仓库是什么

**rheovela-open** 是 RHEOVELA 的**对外发布层**（Apache-2.0，当前 `v0.9.0-beta`）：版本化 contracts、5 语言 Worker SDK、HTTP Worker API、OpenAPI 与示例流程。完整功能见 [docs/FEATURES.md](docs/FEATURES.md)。

- 内核（AGPL-3.0）：https://github.com/axisrobo/rheovela
- 企业版（Enterprise）：https://github.com/axisrobo/rheovela-ee

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

**rheovela-open** is the **public release layer of RHEOVELA** (Apache-2.0, current `v0.9.0-beta`): versioned contracts, 5-language Worker SDKs, the HTTP Worker API, OpenAPI spec and example workflows. Full feature list in [docs/FEATURES.en.md](docs/FEATURES.en.md).

- Core (AGPL-3.0): https://github.com/axisrobo/rheovela
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
