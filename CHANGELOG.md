# Changelog

RHEOVELA open（`github.com/axisrobo/rheovela-open`，Apache-2.0）发布记录：对外发布层（版本化 contracts、SDK、examples、CI、OpenAPI）。格式：最新在前；每个版本对应一个 git tag。

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
