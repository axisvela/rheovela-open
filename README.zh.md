> [English](README.md) · 中文

# RHEOVELA — 动态流程与持久化工作流平台

## RHEOVELA 是什么

RHEOVELA 是一个 **Dynamic Process & Durable Workflow Platform（动态流程与持久化工作流平台）**：把治理后的 Capability Plan 物化为**可持久、可恢复、可审批、可迁移、可审计的 Process Instance**。

**解决的问题**：企业里越来越多的长期工作由 Agent、人类、机器人共同推进，跨越分钟、小时、天甚至月。但 Agent 会话和单个 worker 都是短命的——模型进程退出、网络中断、worker 崩溃、人员换班或系统升级，都可能让"进行中的工作"丢失、重复或无法解释。传统 BPM 建模太静态，吸收不了动态的 Agent 计划；而裸的 Agent loop 又不理解长期责任、恢复与审计。

**RHEOVELA 的答案**：负责"长期工作如何可靠流转"——而不是做开放式规划，也不是跑单个 Agent 的推理循环。

### 核心特性

- **事件溯源 + 确定性内核**：任何崩溃后都能从事件流恢复一致状态；相同事件流必然折叠出相同结果
- **原子幂等命令管线**：重启、重复、乱序不产生重复业务效果
- **Work Item 统一人 / Agent / 服务 / 机器人**：claim / lease / fencing 防止双执行
- **证据链 + 签名**：谁被指派、谁执行、依据什么授权、产生了什么 effect，全程可审计
- **补偿 / 迁移 / replan / 子流程 / 挂起恢复 / 边缘同步 / checkpoint / legal hold / HA 租约锁**

### 多目标接口

- **CLI**（25+ 命令）：`workflow define|validate|diff|import-bpmn|export-bpmn`、`run open|step|close|suspend|resume|list`、`checkpoint`、`migrate`、`replan`、`audit`、`history`、`serve`、`watch`、`sync`、`partition`、`bench`、`dr`
- **HTTP Ops API**（40+ 端点）：instances / steps / workflows / work-items / audit / sync / events（SSE）/ health / status / metrics
- **MCP gateway**：`POST /mcp` — JSON-RPC 2.0 工具，供 Agent 调用
- **Worker HTTP API**：claim / heartbeat / complete / fail

## 这个仓库 — rheovela-open

**rheovela-open** 是 RHEOVELA 的**对外发布层**（Apache-2.0，当前 `v1.0.0-rc.1`）：版本化 contracts、5 语言 Worker SDK、HTTP Worker API、OpenAPI 与示例流程。内核（AGPL-3.0）位于 [rheovela](https://github.com/axisrobo/rheovela)；企业能力位于 [rheovela-ee](https://github.com/axisrobo/rheovela-ee)。完整功能见 [docs/FEATURES.md](docs/FEATURES.md)。

## 目录

| 目录 | 内容 |
|------|------|
| `contracts/` | 版本化 event/command/process schemas（权威契约，`schema_version=1`） |
| `sdk/` | worker SDK（5 种语言）— [Go](sdk/worker/) · [Python](sdk/python/) · [TypeScript](sdk/typescript/) · [Java](sdk/java/) · [Rust](sdk/rust/) |
| `api/` | HTTP API reference — [README（Ops API）](api/README.md) · [OpenAPI 3.0 规范](api/openapi.yaml) · [workflow.schema.json](api/workflow.schema.json) · [event.schema.json](api/event.schema.json) |
| `examples/` | 示例流程（RHEO IR DSL）— [README](examples/README.md) |
| `docs/` | **功能与特性** — [FEATURES.md](docs/FEATURES.md)（[English](docs/FEATURES.en.md)）· 产品介绍 — [product.md](docs/product.md)（[English](docs/product.en.md)） |

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

## 验证

```sh
go test ./...                      # Go SDK 测试
python -m unittest discover -s sdk/python -v   # Python SDK 测试
node --experimental-transform-types --test sdk/typescript/worker.test.ts   # TypeScript SDK 测试
javac -d out sdk/java/worker/Worker.java sdk/java/example/Example.java && java -cp out example.Example   # Java SDK
(cd sdk/rust && cargo build --offline && cargo run --offline)             # Rust SDK
```

CI（`.github/workflows/ci.yml`）跑 4 个 job：**go** / **python** / **java** / **rust**。
