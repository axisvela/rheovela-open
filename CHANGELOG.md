# Changelog

RHEOVELA open（`github.com/axisrobo/rheovela-open`，Apache-2.0）发布记录：对外发布层（版本化 contracts、SDK、examples、CI、OpenAPI）。格式：最新在前；每个版本对应一个 git tag。

## [v0.9.0-beta] - 2026-08-08

首个 **beta** 发布面（tag 与 v1.14-core 同指向）：SDK 覆盖 **5 种语言**
（Go / Python / TypeScript / Java / Rust）。新增 **Java worker SDK**
（`sdk/java/`：JDK 21、零依赖 `java.net.http.HttpClient`，
`Worker.poll/claim/complete/fail/processOnce`）与 **Rust worker SDK**
（`sdk/rust/`：std-only、零外部 crate、`std::net::TcpStream` 极简 HTTP/1.1
client），并新增 **java / rust CI jobs**——`.github/workflows/ci.yml` 现有
4 个 job（go / python / java / rust）。

## [v1.14-core] - 2026-08-08

**Rust worker SDK**（W24.5，`sdk/rust/`：`Client` / `Worker` / `process_once`，
离线可构建）+ **Rust CI job**（`cargo build --offline` + `cargo run --offline`）。

## [v1.13-core] - 2026-08-08

**Worker SDK 后台心跳**（W23.2）：`sdk/worker.Worker` 在长任务执行期间于后台
续租（`worker_test.go` 增加 heartbeat 用例），避免长任务租约过期被重新领取。

## [v1.12-core] - 2026-08-08

**Event JSON Schema**（W21.1，`api/event.schema.json`，draft-07，覆盖 **13 种事件
类型**）+ `api/schema_test.go` 一致性测试（`TestEventSchema`）。

## [v1.11-core] - 2026-08-08

**WorkflowDefinition JSON Schema**（W20.1，`api/workflow.schema.json`，draft-07）+
conformance 测试（`TestWorkflowSchemaValidatesSample` 等）；`examples/README.md`
声明示例 JSON 均符合该 schema。

## [v1.10-core] - 2026-08-08

Runnable examples 补齐：新增 **Python worker example**（`examples/python-worker/main.py`）与 **TypeScript worker example**（`examples/typescript-worker/worker-example.mjs`），与 Go example（v1.9-core）构成三语言可运行样例（`examples/README.md`）。

## [v1.9-core] - 2026-08-08

Runnable **Go worker example**（`examples/go-worker/main.go` + `main_test.go`，in-memory store），示例流程可直接运行并纳入 `go test`（3 包 ok：contracts / sdk/worker / examples/go-worker）。

## [v1.8-core] - 2026-08-07

OpenAI 发布面扩展：**TypeScript worker SDK**（`sdk/typescript/worker.ts` + `worker.test.ts`，零依赖、Node 18+ `fetch`）、**OpenAPI 3.0 规范**（`api/openapi.yaml`，700 行）、**双语 README**（中文 + English 快速开始）。

## [v1.7-core] - 2026-08-07

**Python worker SDK**（`sdk/python/worker.py`：stdlib-only `Client`/`Worker`）+ **Python CI job**（`.github/workflows/ci.yml`，`unittest` 跑 `sdk/python` 测试）。

## [v1.6-core] - 2026-08-07

Contracts catalog 补齐 + 发布资产：**SubprocessStarted 事件注册**（W7.1，KnownEventTypes 达 **11 种**）+ open release assets（`examples/` 两个 RHEO IR DSL 样例、`api/README.md` HTTP API reference、`docs/product.md` 产品介绍、`sdk/worker/README.md` worker SDK 指南，W7.4）。

## [v1.5-core] - 2026-08-07

Contracts catalog 增长：**CompensationExecuted 事件注册**（B8 / Wave 6，KnownEventTypes 10 种）。

## [v1.4-core] - 2026-08-07

Contracts catalog 增长：**StageAssigned 事件注册**（W2.2，引入 `KnownEventTypes` 目录，初始 8 种）+ **Migrated 事件注册**（W5.1，KnownEventTypes 9 种）；`contracts/types_test.go` 增加对应 catalog round-trip 断言。

## [v1.0-core] - 2026-08-07

首个 open release：**版本化 contracts**（`contracts/`，`schema_version=1`：Event/WorkflowDefinition/Stage/Transition/RunContext/Actor 等 + schema round-trip 测试）+ **Go worker SDK alpha**（`sdk/worker/`：`WorkStore` 端口 + heartbeat/lease renewal/fencing token/structured failure，W1.6）。
