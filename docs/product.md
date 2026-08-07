# RHEOVELA — Product Introduction

> RHEOVELA：**Dynamic Process & Durable Workflow Platform**。
> 将治理后的 **Capability Plan** 物化为可持久、可恢复、可审计的 **Process Instance**。

## 定位

RHEOVELA 是一个面向「动态流程 + 持久工作流」的运行时平台。核心思路：

1. **以能力（capability）为契约**：流程由活动构成，活动按 `AgentTask` /
   `HumanTask` / `ServiceTask` 声明所需能力与角色，而不是绑定具体实现。
2. **事件溯源（event sourcing）**：一切状态变化都是追加写的事件；当前状态由纯函数
   折叠事件流得到，重放即重建，天然可审计、可调试。
3. **确定性内核**：激活状态机（AND-join / XOR-split / 旁路）+ typed condition AST，
   相同事件流必然得到相同结果。
4. **面向治理的变更**：结构变化不靠「运行时改图」，而走**版本、迁移或审批**
   （versioned definitions + migration dry-run + live migration）。
5. **持久化保证**：幂等命令管线、effect ledger（外部副作用记账）、补偿、计时器、
   worker 租约/fencing token，保证重启 / 重试 / 并发下不产生重复业务效果。

## 三仓库结构

| 仓库 | License | 定位 |
|------|---------|------|
| [`rheovela-open`](https://github.com/axisrobo/rheovela-open)（本仓库） | Apache-2.0 | 对外发布层：版本化 `contracts/`（`schema_version=1`）、`sdk/`（worker SDK）、`api/`（HTTP API reference）、`examples/`、`docs/` |
| [`rheovela`](https://github.com/axisrobo/rheovela) | AGPL-3.0 | 内核：store / kernel / engine / application / scheduler / broker / effects / compensation / migration / server / CLI（`cmd/rheo`），并暴露公共 `runtime` 包 |
| `rheovela-ee` | Enterprise | 企业层：多租户、HA、audit/evidence explorer、migration console、企业 IdP（AEGIVELA） |

**依赖方向**：`rheovela-open` → `rheovela`（core import open 的 contracts）→
`rheovela-ee`。Core 不包含企业/闭源功能；contracts 放 Apache 下，社区 worker/client
SDK 不受 AGPL 传染。

## 关键概念

### 事件溯源与确定性内核

- 事件追加写（`events` 表），每流服务端分配聚合序号 `seq`，防止重复/乱序。
- `internal/engine` 以纯函数折叠事件（`RunOpened`、`StepEntered`、
  `StepCompleted`、`StepFailed`、`StepSkipped`、`RunClosed`、`StageAssigned`、
  `TimerFired`、`Migrated`、`CompensationExecuted` 等），重建运行状态。
- 投影（`run_contexts` / `process_instances`）与事件流同事务写入，可随时
  `RebuildProjection`。

### 幂等性

- 所有命令（open / step / close / assign / migrate）携带 `idempotency_key`。
- 幂等键在命令事务内检查/写入（`idempotency_keys`），请求摘要检测冲突，重复提交不
  产生重复效果。

### Effect Ledger（效果账本）

- 外部副作用（调用 ERP、打款、过账）先记 intent、后记 outcome
  （`effect_records(idempotency_key, target, request_digest, state, evidence)`）。
- 未知结果进入 reconciliation；效果按幂等键去重，保证外部动作恰好一次。

### 补偿（Compensation）

- RHEO IR 用 `compensate <activity> with ...` 声明补偿动作。
- `internal/compensation` 按**逆序**构建补偿计划并执行（`CompensationExecuted`
  事件），已成功的活动在失败路径上可被回滚；无声明动作走 manual fallback。

### 迁移（Migration）

- 定义版本化存储（`definition_versions`），运行实例固定 `definition_version`。
- 迁移先 dry-run（`internal/migration.Analyze`）：**增量变更兼容，删除节点/边不兼容
  被拒绝**；兼容路径通过 `MigrateInstance` + `Migrated` 事件 live 迁移，投影与重放
  尊重迁移。

### 证据（Evidence）

- 每个事件带 `actor_id`、签名（HMAC-SHA256，`--signing-key`）与时间戳。
- `internal/audit` 构建证据链：plan → events → assignment → execution，并反查
  effect records；HTTP 端点在带 `--verify-key` 时做签名校验。

## 数据模型

SQLite（`modernc.org/sqlite`，纯 Go 无 CGO，WAL）13 张表：事件流、项目、定义版本、
流程实例、幂等键、计时器、工作项、效果记录、阶段执行等。契约类型（`Event` /
`WorkflowDefinition` / `ProcessInstance` / `RunContext`）在 `contracts/` 中版本化。

## 快速开始

RHEOVELA 的内核与 CLI 位于 core 仓库（AGPL）。本仓库提供契约、SDK 与示例。

1. 构建/安装 core CLI：`rheo`（`cmd/rheo`，Windows `rheo.exe`）。
2. 查看示例：`examples/README.md` — 用 `rheo workflow define --file <json>` 导入
   `examples/` 中的流程，用 `rheo run open <workflow-id> --var k=v` 打开运行。
3. 编写 worker：`sdk/worker/README.md` — 实现 `WorkStore` 端口（或嵌入 core 的
   `runtime.WorkerBridge`），用 `Worker` / `ProcessOnce` 消费工作项。
4. 对接 HTTP API：`api/README.md` — `rheo serve` 暴露 `/api/v1/instances` 等路由。

完整 CLI / Go API 见 core `docs/api.md`，架构见 core `docs/architecture.md`。
