# Examples — RHEOVELA 示例流程

本目录包含 RHEOVELA 的示例流程，以 **RHEO IR（Intermediate Representation）DSL**
形式呈现，对应 core 的 `internal/rheoir` 解析器（Wave 3.3）。IR 可编译为
`WorkflowDefinition`（`schema_version=1`），再通过 core CLI 导入并运行。

> 说明：`rheo workflow define` 当前导入的是 **JSON** 形式的 `WorkflowDefinition`。
> `.rheo` 文件是面向作者的 IR 源文本，由 core 的 `internal/rheoir`（`Parse` →
> `ToWorkflowDefinition`）编译为 JSON。下面每个示例都附了编译产物与运行步骤。

## 文件一览

| 文件 | 内容 | 阶段 |
|------|------|------|
| `purchase-approval.rheo` | 采购审批：发票校验 → 成本中心审批 → ERP 过账，按金额分叉，过账可补偿 | validate → approve / post |
| `expense-approval.rheo` | 费用报销：校验 → 经理审批 → 财务复核 → 打款，超限进入财务复核，打款可补偿 | validate → approve → financeCheck → pay |

两个示例都演示了三种活动类型（`AgentTask` / `HumanTask` / `ServiceTask`）、条件
分支（`when`）与补偿（`compensate`），其中 `ServiceTask` 声明了 `effect_key`
（effect ledger 幂等键模板）。

## RHEO IR DSL 语法摘要

DSL 由 `process` 声明、`activity`、`transition`、`compensate` 四类行构成（行内 `//`
注释会被解析器剥离）。

```
process <Name> v<N> {
  activity <id> : <AgentTask|HumanTask|ServiceTask> [capability "<cap>"] [role "<role>"] [effect_key "<template>"] [timeout <dur>]
  transition <from> -> <to> [when <condition>]
  compensate <activity-id> with capability "<cap>"
  compensate <activity-id> with <activity-id>
}
```

### `process`

声明流程名称与版本：

```
process PurchaseApproval v1 {
```

### `activity`

声明一个活动（映射为 `WorkflowDefinition.stages`）：

| 属性 | 含义 |
|------|------|
| `<AgentTask|HumanTask|ServiceTask>` | 活动类型：agent 任务 / 人工任务 / 服务任务 |
| `capability "<cap>"` | 执行该活动所需的能力名（`AssigneeSpec.required`） |
| `role "<role>"` | 指派角色（`AssigneeSpec.role`） |
| `effect_key "<template>"` | 外部效果幂等键模板（如 `invoice/${id}`），供 effect ledger 使用 |
| `timeout <dur>` | 超时时长（`30m` / `2h` / `5d` 等） |

示例：

```
activity validate : AgentTask capability "invoice.validate"
activity approve  : HumanTask role "cost-center-owner"
activity post     : ServiceTask capability "erp.invoice.post" effect_key "invoice/${id}"
```

### `transition`

声明活动间的流转，可选 `when` 条件：

```
transition <from> -> <to> [when <var> <op> <literal>]
```

条件为 `var op literal` 形式，`op` 支持 `>=` `<=` `==` `!=` `>` `<`（core
`internal/kernel` 的 `Evaluate`）。变量值在 `rheo run open --var k=v` 时传入。

### `compensate`

为已成功活动声明补偿动作（`internal/compensation` 按逆序构建补偿计划）：

```
compensate <activity-id> with capability "<cap>"   // 通过能力名补偿
compensate <activity-id> with <activity-id>        // 通过另一个活动补偿
```

## 如何运行

前置：安装 core CLI（`github.com/axisrobo/rheovela`，`cmd/rheo` → `rheo`/`rheo.exe`）。

### 1. 编译 IR → JSON

把 `.rheo` 编译为 `WorkflowDefinition` JSON（core 的 `internal/rheoir` 提供
`Parse` + `ToWorkflowDefinition`）。`purchase-approval.rheo` 对应的 JSON：

```json
{
  "workflow_id": "PurchaseApproval",
  "title": "PurchaseApproval",
  "source": "rheo-ir",
  "stages": [
    {"id": "validate", "title": "validate", "assignee_spec": {"kind": "capability", "required": ["invoice.validate"]}, "inputs": [], "outputs": []},
    {"id": "approve", "title": "approve", "assignee_spec": {"kind": "role", "role": "cost-center-owner"}, "inputs": [], "outputs": []},
    {"id": "post", "title": "post", "assignee_spec": {"kind": "capability", "required": ["erp.invoice.post"]}, "inputs": [], "outputs": []}
  ],
  "transitions": [
    {"from": "validate", "to": "approve", "condition": "amount >= 1000"},
    {"from": "validate", "to": "post", "condition": "amount < 1000"}
  ]
}
```

`expense-approval.rheo` 对应的 JSON：

```json
{
  "workflow_id": "ExpenseApproval",
  "title": "ExpenseApproval",
  "source": "rheo-ir",
  "stages": [
    {"id": "validate", "title": "validate", "assignee_spec": {"kind": "capability", "required": ["expense.validate"]}, "inputs": [], "outputs": []},
    {"id": "approve", "title": "approve", "assignee_spec": {"kind": "role", "role": "manager"}, "inputs": [], "outputs": []},
    {"id": "financeCheck", "title": "financeCheck", "assignee_spec": {"kind": "role", "role": "finance-team"}, "inputs": [], "outputs": []},
    {"id": "pay", "title": "pay", "assignee_spec": {"kind": "capability", "required": ["payroll.pay"]}, "inputs": [], "outputs": []}
  ],
  "transitions": [
    {"from": "validate", "to": "approve"},
    {"from": "approve", "to": "financeCheck", "condition": "amount >= 1000"},
    {"from": "financeCheck", "to": "pay"}
  ]
}
```

> 注意：IR 中的 `effect_key` 与 `compensate` 目前是 IR 层的丰富语义，尚未落入
> `WorkflowDefinition` JSON schema（`contracts.Stage` / `contracts.Transition`
> 尚无对应字段）；它们由 core 的 effect ledger / compensation 计划器消费。

> 所有示例 `WorkflowDefinition` JSON 均符合 `api/workflow.schema.json`
> （draft-07），可用任意 JSON Schema 校验器或 `go test ./api/ -v` 的
> `TestWorkflowSchemaValidatesSample` 验证。

### 2. 定义并运行

以 `PurchaseApproval` 为例（Windows PowerShell / 类 Unix shell 均可）：

```sh
# 创建并选中项目
rheo project new purchase-approval
rheo project list
rheo project use <id>

# 导入编译后的定义
rheo workflow define --file purchase-approval.json

# 打开一个运行（金额 ≥ 1000 → 走审批分支）
rheo run open PurchaseApproval --var amount=1500 --as alice

# 逐步推进
rheo step enter validate --as alice
rheo step exit  validate --as alice
rheo step enter approve  --as alice
rheo step exit  approve  --as alice
rheo step enter post     --as alice
rheo step exit  post     --as alice

# 查看状态（TUI / HTML / JSON）
rheo view --format tui

# 关闭运行
rheo run close --outcome done
```

若 `amount=500`（`< 1000`），`validate` 完成后将直接进入 `post` 分支：

```sh
rheo run open PurchaseApproval --var amount=500 --as alice
rheo step enter validate --as alice
rheo step exit  validate --as alice
rheo step enter post     --as alice
rheo step exit  post     --as alice
rheo run close --outcome done
```

### 3. 常用 CLI 命令速查

| 命令 | 作用 |
|------|------|
| `rheo workflow define --file <json>` | 导入工作流定义 |
| `rheo workflow list` / `rheo workflow show <id>` | 查看定义 |
| `rheo run open <workflow-id> [--var k=v]... [--as <actor>]` | 打开运行 |
| `rheo step enter\|exit\|fail\|skip <stage-id> [--as <actor>]` | 推进/失败/跳过阶段 |
| `rheo view [--format tui\|html\|json]` | 查看运行状态 |
| `rheo run close --outcome done\|fail` | 关闭运行 |
| `rheo history --run <id>` | 查看事件日志 |
| `rheo serve` | 启动 HTTP Ops API |

详细 CLI 文档见 core 仓库 `docs/api.md`。

## Go Worker 示例（SDK 端到端）

`go-worker/` 是一个可运行的 Go worker 示例，演示 worker SDK
（`sdk/worker`）的完整处理循环：poll → claim → run → complete/fail。它使用一个
内存中的 `fakeStore`（实现 `worker.WorkStore`），不依赖 core。

### 运行

```sh
go run ./examples/go-worker/
```

预期输出：

```
processed=2
  task-1: done
  task-2: failed
```

`task-1` 的工作函数返回 `success` → `done`；`task-2` 返回 `failure` → `failed`。

### 测试

```sh
go test ./examples/go-worker/ -v   # TestExampleWorkerFlow
```

### 对接真实 core

示例中的 `fakeStore` 是 `sdk/worker.WorkStore` 端口的一个最小实现。生产环境中
该端口由 rheovela core（AGPL 内核）的 `runtime` 包提供：`runtime.WorkerBridge`
实现了 `WorkStore`，将 `PollReady` / `Claim` / `Heartbeat` / `Complete` / `Fail`
映射到 core 的 Worker HTTP API（`rheo serve`）。接入时只需把 `worker.New`
的第一个参数从 `fakeStore` 替换为 `runtime.WorkerBridge` 实例即可，其余处理逻辑不变。

## Python Worker 示例（SDK 端到端）

`python-worker/main.py` 是一个可运行的 Python worker 示例，演示 `sdk/python/worker.py`
的完整处理循环：poll → claim → complete/fail。它启动一个本地 fake Worker-API 服务器
（stdlib `http.server`，内存 dict 存储），进程结束后自动关闭。仅依赖 Python 标准库。

示例复用 `sdk/python/worker.py` 的 `Client` / `Worker`（通过 `sys.path` 引入），
`task-1` 返回 `success` → `done`；`task-2` 返回 `failure` → `failed`。

### 运行

```sh
python examples/python-worker/main.py
```

预期输出：

```
processed=2
  task-1: done
  task-2: failed
```

### 对接真实 core

示例中的 `FakeWorkerServer` 实现了 Worker HTTP API（`GET /api/v1/work-items`、
`POST .../claim` / `complete` / `fail`）。生产环境中该 API 由 core 的 `rheo serve`
提供，把 `Client(srv.base_url)` 换成 `Client("http://localhost:8080")` 即可。

## TypeScript Worker 示例（SDK 端到端）

`typescript-worker/worker-example.mjs` 是一个可运行的 ESM 脚本（Node 18+），演示
`Worker.processOnce()` 的完整处理循环：poll → claim → complete/fail。它 mock
`globalThis.fetch` 指向一个内存中的 Worker-API 响应，无需任何运行时依赖。

> Node 无法直接 `import` `.ts` 文件（需 transform），因此脚本内联了一份与
> `sdk/typescript/worker.ts` 对应的 `Client` / `Worker` 最小实现，并注明镜像自 SDK。

`task-1` 返回 `success` → `done`；`task-2` 返回 `failure` → `failed`。

### 运行

```sh
node examples/typescript-worker/worker-example.mjs
```

预期输出：

```
processed=2
  task-1: done
  task-2: failed
```

### 对接真实 core

示例中 mock 的 `fetch` 返回 Worker HTTP API 的响应。生产环境中把
`new Client("http://fake")` 换成 `new Client("http://localhost:8080")` 并移除
`globalThis.fetch` 的 mock 即可对接 core 的 `rheo serve`。
