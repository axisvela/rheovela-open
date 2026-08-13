# RHEOVELA 开发手册

> [English](DEVELOPMENT.md) · 中文

本手册面向在 `rheovela-open` 仓库内工作的开发者：扩展 contracts、基于
Worker SDK 构建、编写示例流程，并把 5 种语言 SDK 接入运行中的 core 节点。

> **内核在别处。** 引擎、存储、CLI（`rheo`）与 HTTP 服务位于
> `github.com/axisrobo/rheovela`（AGPL-3.0）。`rheovela-open`（Apache-2.0）
> 是公共层：contracts、SDK、API 参考、示例与 CI。企业能力位于
> `github.com/axisrobo/rheovela-ee`。本手册覆盖开放层；core 文档以
> `core:` 标注。

---

## 1. 仓库结构

| 路径 | 内容 |
|------|------|
| `contracts/` | 版本化 Go 类型——`Event`、`WorkflowDefinition`、`Stage`、`Transition`、`RunContext`（`schema_version=1`） |
| `sdk/worker/` | Go Worker SDK——`WorkStore` 端口 + `Worker` 循环 |
| `sdk/python/` | Python Worker SDK（stdlib-only，Python 3.9+） |
| `sdk/typescript/` | TypeScript Worker SDK（零依赖，Node 18+ `fetch`） |
| `sdk/java/` | Java Worker SDK（JDK 21，零依赖） |
| `sdk/rust/` | Rust Worker SDK（std-only，零 crate） |
| `api/` | HTTP API 参考、OpenAPI 3.0 规范、JSON Schema + schema 测试 |
| `examples/` | 示例流程（RHEO IR DSL）与可运行的 worker 示例 |
| `docs/` | 特性、产品介绍、**运维手册**（`OPERATIONS.md`）与本文档 |
| `.github/workflows/ci.yml` | CI：go / python / java / rust 四个 job |

依赖方向：`rheovela-open`（Apache）→ `rheovela`（AGPL core）→
`rheovela-ee`（Enterprise）。Core import open 的 contracts；社区 SDK 永不受
AGPL 传染。

---

## 2. Contracts（`contracts/`）

`contracts/types.go` 是权威的版本化契约面（`SchemaVersion = "1"`）。修改契约
是跨仓库变更——core 引擎与 ee 都消费这些类型。

关键类型：

| 类型 | 用途 |
|------|------|
| `Event` | 不可变事件（`type`、`actor_id`、`payload`、`wall_time`、可选 `signature`） |
| `WorkflowDefinition` | `workflow_id`、`stages[]`、`transitions[]` |
| `Stage` | `id`、`title`、`assignee`/`assignee_spec`、`inputs`、`outputs`、可选 `gate` |
| `AssigneeSpec` | `kind`：`actor` / `role` / `capability` / `open` |
| `Transition` | `from` → `to`，可选 `condition` 表达式 |
| `RunContext` | 运行投影：状态、当前 stage、变量、已完成 stages |
| `KnownEventTypes` | 引擎折叠的事件类型目录 |

### 新增事件类型

1. 在 `KnownEventTypes` 中加入名称（如 `"ProcessPaused"`）。
2. 在 core 仓库更新引擎折叠（`internal/engine`）以处理该类型。
3. 更新 `api/event.schema.json`，使 JSON Schema 覆盖新类型。
4. 在 `contracts/types_test.go` 新增 / 扩展往返测试。
5. 只有**破坏性**变更才提升 `SchemaVersion`；增量化字段保持 v1。

### JSON Schema + OpenAPI

- `api/workflow.schema.json` —— `WorkflowDefinition`（draft-07）
- `api/event.schema.json` —— `Event`（覆盖全部 13 种事件类型）
- `api/openapi.yaml` —— HTTP Ops API + Worker API 路由
- `api/schema_test.go` —— 保持 schema 与 `contracts` 一致并校验内置示例定义

按 schema 校验：

```sh
go test ./api/ -v                 # schema 一致性 + 示例校验
```

---

## 3. Worker SDK——如何构建 worker

5 个 SDK 都针对 core 的 Worker HTTP API 实现同一契约：

| SDK | 位置 | 运行时 | 循环 API |
|-----|------|--------|----------|
| Go | `sdk/worker` | Go 1.25 | `Worker.ProcessOnce(ctx)` |
| Python | `sdk/python/worker.py` | Python 3.9+（stdlib） | `Worker.process_once()` |
| TypeScript | `sdk/typescript/worker.ts` | Node 18+（fetch） | `Worker.processOnce()` |
| Java | `sdk/java/worker/Worker.java` | JDK 21 | `Worker.processOnce(fn, lease)` |
| Rust | `sdk/rust/` | Rust std（离线） | `Worker::process_once(fn, lease)` |

各 SDK 调用的 Worker HTTP API：

| Method | Path | 用途 |
|--------|------|------|
| `GET` | `/api/v1/work-items?instance=<id>` | 列出 ready 工作项 |
| `POST` | `/api/v1/work-items/{id}/claim` | 领取 → `{token, lease, lease_until}` |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | 续租 `{token, lease}` |
| `POST` | `/api/v1/work-items/{id}/complete` | 完成 `{token}` |
| `POST` | `/api/v1/work-items/{id}/fail` | 失败 `{token, error}` |

Worker 就是一个**循环**：poll ready 项 → claim（fencing token）→ 运行你的函数
→ 成功 complete / 失败 fail。SDK 负责协议，你提供工作函数。

### Go（嵌入，本地 / edge）

Go SDK 定义 `WorkStore` 端口。生产环境中 core 提供 `runtime.WorkerBridge`
（在本地 broker 上实现 `WorkStore`）；你也可以针对任意后端实现这 5 个方法。

```go
db, _ := runtime.OpenDB("~/.proc/proc.db")
st := runtime.NewStore(db)
bridge := runtime.NewWorkerBridge(st)

fn := func(ctx context.Context, item worker.WorkItem) worker.WorkResult {
    // do the work...
    return worker.WorkResult{Status: "success", Outcome: "ok"}
}

w := worker.New(bridge, fn, 30*time.Second)
for {
    n, _ := w.ProcessOnce(context.Background())
    if n == 0 { time.Sleep(2 * time.Second) }
}
```

### Python / TypeScript / Java / Rust（HTTP）

把 `Client` 指向 `rheo serve` 地址；`Worker` 循环在语言间完全一致：

```python
import worker
w = worker.Worker(worker.Client("http://localhost:8080"), fn, lease="30s")
while True:
    if w.process_once() == 0:
        time.sleep(2)
```

各语言完整文档与代码：
- [Go](../sdk/worker/README.en.md) · [Python](../sdk/python/README.en.md) ·
  [TypeScript](../sdk/typescript/README.en.md) · [Java](../sdk/java/README.en.md) ·
  [Rust](../sdk/rust/README.en.md)

### 并发与 fencing

- `claim` 返回 **fencing token**；只有持有者才能 `complete`/`fail`。
- 租约过期后其他 worker 可重新领取——过期 token 会被拒绝（防止双执行）。
- 长任务请调用 `heartbeat` 续租，避免任务中途租约过期。

---

## 4. 构建与测试

```sh
# Go（contracts、schemas、Go SDK）
go test ./...

# Python SDK
python -m py_compile sdk/python/worker.py
python -m unittest discover -s sdk/python -v

# TypeScript SDK（Node 18+；Node 22.7+ 使用 --experimental-transform-types）
node --experimental-transform-types --test sdk/typescript/worker.test.ts

# Java SDK
javac -d out sdk/java/worker/Worker.java sdk/java/example/Example.java
java -cp out example.Example

# Rust SDK（离线构建）
cd sdk/rust && cargo build --offline && cargo run --offline
```

`go test ./api/ -v` 额外校验 JSON Schema 与 OpenAPI 规范。

---

## 5. 示例流程

`examples/` 包含 RHEO IR DSL 源文件以及可运行的 worker 示例。

### RHEO IR DSL

`.rheo` 文件声明 `process`、`activity`、`transition`、`compensate` 行，可编译为
`WorkflowDefinition` JSON（经 core `rheo ir tojson`）：

```
process PurchaseApproval v1 {
  activity validate : AgentTask capability "invoice.validate"
  activity approve  : HumanTask role "cost-center-owner"
  activity post     : ServiceTask capability "erp.invoice.post" effect_key "invoice/${id}"
  transition validate -> approve when amount >= 1000
  transition validate -> post    when amount < 1000
  compensate post with capability "erp.invoice.void"
}
```

用 core CLI 编译 + 定义 + 运行：

```sh
rheo ir tojson examples/purchase-approval.rheo --output purchase-approval.json
rheo workflow define --file purchase-approval.json
rheo run open PurchaseApproval --var amount=1500 --as alice
# ... 依次推进 validate/approve/post，然后 close
```

### 可运行的 worker 示例

- `examples/go-worker/` —— 带内存 `fakeStore` 的 Go worker（`go run ./examples/go-worker/`）
- `examples/python-worker/` —— 面向本地 fake Worker API 的 Python worker（`python examples/python-worker/main.py`）
- `examples/typescript-worker/` —— 带 mock `fetch` 的 TS worker（`node examples/typescript-worker/worker-example.mjs`）

每个都展示完整的 poll → claim → complete/fail 循环，以及如何把 fake 后端换成
真实 `rheo serve`（`Client("http://localhost:8080")`）。

---

## 6. CI

`.github/workflows/ci.yml` 在每次 push/PR 时运行 4 个独立 job：

| Job | 运行内容 |
|-----|----------|
| `test` | `go test ./...`（contracts + schemas + Go SDK） |
| `python` | 编译 + `unittest`（`sdk/python`） |
| `java` | `javac` + 运行 `example.Example` |
| `rust` | `cargo build --offline` + `cargo run --offline` |

推送前保持 SDK job 全绿；没有跨语言测试框架——各 SDK 独立按共享的 Worker API
契约验证。

---

## 7. 发布与版本

按**语义版本 tag**（`v1.0.0`、`v0.9.0-beta`、`v1.0.0-rc.1`）版本化。开发里程碑
tag（`v1.x-core`）**不**产生 GitHub Release。

| 产物 | 位置 | 如何 |
|------|------|------|
| Contracts（`schema_version=1`） | `contracts/` | 仅破坏性变更时提升 |
| `rheo` 二进制（5 目标 + SHA256SUMS） | core 仓库 Releases | 语义版本 tag 触发 CI |
| OpenAPI + JSON Schema | `api/` | 重新生成 + `go test ./api/` 校验 |
| CHANGELOG | `CHANGELOG.md` | 每个发布加一条 |

发布流程：

1. 冻结变更；确认 `go test ./...` 与所有 SDK job 通过。
2. 打语义版本 tag 并推送（core 仓库构建二进制；open 仓库的文档 / contracts
   使用同名 tag）。
3. 更新 `CHANGELOG.md` 与文档中的版本戳。

---

## 8. 故障排查（开发者）

| 症状 | 可能原因 / 修复 |
|------|-----------------|
| `api/` 的 `go test ./...` 失败 | Schema 与 `contracts` 漂移；`go test ./api/ -v` 并更新 JSON Schema |
| 新事件类型不折叠 | 更新了 `KnownEventTypes` 但 core `internal/engine` 缺对应 fold（core 仓库） |
| Worker `claim` 返回 409 | 其他 worker 持项；调大租约或加密心跳 |
| Node 18 无法运行 `worker.test.ts` | 用 `tsc`/`esbuild` 转译，或 Node 22.7+ 加 `--experimental-transform-types` |
| Java 示例编译失败 | JDK 需 21+（`java.net.http`）；检查 `javac` classpath |
| Rust 离线构建失败 | 引入了外部 crate；保持 `sdk/rust` std-only |

---

另见：
- **运维手册**：[OPERATIONS.md](OPERATIONS.md)（English）· [中文](OPERATIONS.zh.md)
- **功能特性**：[FEATURES.md](FEATURES.md)（[English](FEATURES.en.md)）
- **HTTP API 参考**：[api/README.md](../api/README.md)（[English](../api/README.en.md)）
- **core CLI/Go API**（在 `github.com/axisrobo/rheovela`）：`docs/api.md`
