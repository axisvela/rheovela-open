# RHEOVELA 运维手册

> [English](OPERATIONS.md) · 中文

本手册覆盖基于内核（`rheo`，`github.com/axisrobo/rheovela`）部署的 RHEOVELA
单节点运维。`rheovela-open` 仓库提供集成所需的 contracts、Worker SDK 与 API
参考；运行时本体是 `rheo` 二进制。

> **范围**：单节点本地 / 私有化运维。多租户部署、身份与控制面位于
> `rheovela-ee`（`rheo-ee serve`）。HA / DR 边界见 [备份与灾备](#7-备份与灾备)。

---

## 1. 安装

### 1.1 下载发布二进制（推荐）

每个语义版本 Release 都会在 GitHub Releases 页面发布 5 个跨编译目标
以及 `SHA256SUMS` 校验清单：

| 资产 | 平台 |
|------|------|
| `rheo-windows-amd64.exe` | Windows amd64 |
| `rheo-linux-amd64` | Linux amd64 |
| `rheo-linux-arm64` | Linux arm64 |
| `rheo-darwin-amd64` | macOS amd64 |
| `rheo-darwin-arm64` | macOS arm64 |
| `SHA256SUMS` | 校验清单 |

首次运行前校验：

```sh
# macOS / Linux
sha256sum -c SHA256SUMS --ignore-missing rheo-linux-amd64
# Windows PowerShell
Get-FileHash -Algorithm SHA256 .\rheo-windows-amd64.exe
```

### 1.2 源码构建

需要 Go 1.25。单二进制构建（无 CGO——`modernc.org/sqlite`）：

```sh
go build -o rheo ./cmd/rheo
```

或用发布脚本跨编译 5 个目标（输出 `dist/SHA256SUMS`）：

```powershell
powershell -ExecutionPolicy Bypass -File release/build.ps1
```

查看版本：

```sh
rheo --version   # → rheo v1.0.0-rc.1（发布时注入）
```

### 1.3 数据目录

全部状态存放在一个 SQLite 文件中，默认路径 `~/.proc/proc.db`（WAL 模式）。
`serve` 可用 `--db <path>` 覆盖；其他命令视支持情况可用 `RHEO_DB`。务必备份
该文件——见 [备份与灾备](#7-备份与灾备)。

### 1.4 Postgres 后端（可选，ee）

`rheo-ee serve` 可以用 PostgreSQL 替代 SQLite：

```sh
rheo-ee serve --store postgres --db-url "postgres://user:pass@host:5432/rheo?sslmode=disable" --addr :8081
```

- 全部引擎状态存放在 Postgres 表中（与 SQLite 同构，`TIMESTAMPTZ`）。
- **备份 / 灾备**：Postgres 使用 `pg_dump` / `pg_restore`；基于文件的 `rheo dr`
  命令仅限 SQLite。
- core `rheo` CLI 仍面向 SQLite；Postgres 是 ee / server 的选项。

---

## 2. 首次运行：定义 / 打开 / 推进 / 关闭

```sh
# 创建并选择项目
rheo project new quickstart      # → Project #1 created: quickstart
rheo project use 1

# 导入工作流定义（JSON；或用 `rheo ir tojson <file>.rheo`）
rheo workflow define --file expense.json

# 带变量与 actor 打开运行
rheo run open expense-reimbursement --var amount=1200 --as alice

# 逐 stage 推进
rheo step enter submit --as alice
rheo step exit submit
rheo step enter dept_approve --as bob
rheo step exit dept_approve

# 查看，然后关闭
rheo view                       # TUI
rheo history --run <run-id>     # 事件日志
rheo run close --outcome done
```

每条命令都会追加一条不可变、带签名的事件；运行状态由事件流折叠得出——
绝不靠直接改状态。

---

## 3. 运行 HTTP 服务

`rheo serve` 在同一 mux 上暴露 HTTP Ops API、Worker API、MCP gateway
以及可选的 SSE 事件流：

```sh
rheo serve [--addr :8080] [--db <path>] [--verify-key <hex>] \
           [--scheduler-interval <duration>] [--scheduler-off] \
           [--tls-cert <file> --tls-key <file>] [--readonly]
```

| 参数 | 说明 |
|------|------|
| `--addr` | 监听地址（默认 `:8080`） |
| `--db` | SQLite 数据库路径 |
| `--verify-key <hex>` | 审计签名校验的 HMAC 密钥 |
| `--scheduler-interval <d>` | 到期 timer 轮询间隔（默认 `1s`） |
| `--scheduler-off` | 关闭 durable timer 调度器 |
| `--tls-cert` / `--tls-key` | HTTPS 服务（必须成对提供） |
| `--readonly` | 只读观测面：关闭调度器，变更方法一律 403 |

存活与概览：

```sh
curl http://localhost:8080/api/v1/health    # {"status":"ok","db":"ok"}
curl http://localhost:8080/api/v1/status    # {instances, active, done, failed, ...}
curl http://localhost:8080/api/v1/metrics   # {events, instances, timers_pending, ...}
```

完整路由参考：`api/README.md`（[中文](../api/README.md) / [English](../api/README.en.md)）。OpenAPI 3.0 规范：`api/openapi.yaml`。

### 3.1 MCP gateway

`POST /mcp` 使用 JSON-RPC 2.0（MCP 工具：`open_run`、`enter_step`、
`complete_step`、`fail_step`、`skip_step`、`close_run`、`list_instances`、
`get_evidence`）。适合把 Agent / 助手接入运行时。

```sh
curl -X POST http://localhost:8080/mcp -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

### 3.2 TLS

用成对的 `--tls-cert`/`--tls-key` 开启 HTTPS（单独给 cert 或 key 会被拒绝）。
生产环境建议由反向代理（nginx/Caddy）终止 TLS，并把监听地址钉在回环 / 内网接口。

### 3.3 只读模式

加 `--readonly` 启动即把服务变成纯观测面：调度器关闭，所有
`POST`/`PUT`/`PATCH`/`DELETE` 被拒（`403 {"error":"read-only mode"}`）。
适合读副本、审计人员或不允许变更状态的看板。

---

## 4. 运行 Worker

工作项通过 Worker HTTP API 领取与结算：

| Method | Path | 用途 |
|--------|------|------|
| `GET` | `/api/v1/work-items?instance=<id>` | 列出 ready 工作项 |
| `POST` | `/api/v1/work-items/{id}/claim` | 领取 → `{token, lease, lease_until}` |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | 续租 `{token, lease}` |
| `POST` | `/api/v1/work-items/{id}/complete` | 完成 `{token}` |
| `POST` | `/api/v1/work-items/{id}/fail` | 失败 `{token, error}` |

`rheovela-open` 提供 5 种语言的 Worker SDK。每个都提供
poll → claim → run → complete/fail 循环；fencing token 在租约过期并被其他
worker 重新领取后防止重复执行。

```sh
# Go（嵌入）        → sdk/worker（WorkStore 端口；core runtime.WorkerBridge）
# Python            → sdk/python  （stdlib-only，Python 3.9+）
# TypeScript        → sdk/typescript（零依赖，Node 18+ fetch）
# Java              → sdk/java（JDK 21，零依赖）
# Rust              → sdk/rust（std-only，零 crate，离线构建）
```

示例：把任一 worker 指向服务地址并启动其循环：

```python
import worker
w = worker.Worker(worker.Client("http://localhost:8080"), fn, lease="30s")
while True:
    if w.process_once() == 0:
        time.sleep(2)
```

各语言环境与代码示例：
- [Go](../sdk/worker/README.en.md) · [Python](../sdk/python/README.en.md) ·
  [TypeScript](../sdk/typescript/README.en.md) · [Java](../sdk/java/README.en.md) ·
  [Rust](../sdk/rust/README.en.md)

端到端示例运行：[examples/README.en.md](../examples/README.en.md)。

---

## 5. 监控与审计

### 5.1 运维端点

| 端点 | 说明 |
|------|------|
| `GET /api/v1/health` | 存活（DB ping） |
| `GET /api/v1/status` | 各状态实例数、未发送 outbox、分区、legal hold |
| `GET /api/v1/metrics` | 引擎计数器：events、待触发 timer、ready 工作项、unknown 效果 |
| `GET /api/v1/events?stream=<id>&after_seq=<n>` | SSE 事件流（推送面） |
| `GET /api/v1/sync/delta?stream=<id>&after_seq=<n>` | 增量事件快照（edge 拉取面） |

### 5.2 审计

```sh
rheo history --run <run-id> --verify          # 逐事件 [valid]/[INVALID]
rheo audit export <run-id> --format json      # 证据链导出 JSON
rheo audit export <run-id> --verify-key <hex> # 带签名校验的导出
```

写侧加 `--signing-key <hex>`、服务端加 `--verify-key <hex>`（或环境变量
`RHEO_SIGNING_KEY`）时，每条事件都经 HMAC-SHA256 签名；链式签名可检测
事件重排或删除。

### 5.3 实时事件流

```sh
rheo watch <run-id>            # 每 500ms 以 SSE data: 行流式输出事件
```

---

## 6. 备份与灾备

整个节点状态就是单个 SQLite 文件，因此 DR 是文件层面的故事。

- **文件内 RPO = 0** —— 每条命令在单个事务内提交（WAL）；崩溃最多丢一个未提交事务。
- **RTO ≈ `rheo dr restore <backup>` + 重开** —— 文件复制加重启，无重放或索引重建。

### 三步 restore 演练

```sh
# 1) 备份并验证
rheo dr backup /safe/proc.db.bak
rheo dr verify /safe/proc.db.bak        # 必须打印 OK: true

# 2) 模拟丢失（先停节点）
#    del /your/proc.db            (Windows)
#    rm  /your/proc.db            (POSIX)

# 3) 恢复并重开
rheo dr restore /safe/proc.db.bak
rheo dr verify /your/proc.db
rheo serve
```

### 定时备份

按 RPO 预算的节奏定时备份，并在同一任务里验证每次备份：

```sh
rheo dr backup backups/daily.db.bak && rheo dr verify backups/daily.db.bak
```

备份务必存放在与线上 DB 不同的磁盘 / 对象存储。若 `verify` 未输出
`OK: true`，不要用该文件运行节点。

> **HA 说明**：`ha_locks` 提供租约锁（fencing）原语，但**多节点 leader
> election、复制与自动故障转移尚未实现**。请把每个节点当作独立的持久化单元，
> 逐节点备份。

---

## 7. Day-2 运维速查

| 任务 | 命令 |
|------|------|
| 列出运行（按状态 / 项目过滤） | `rheo run list [--status active] [--project 1]` |
| 查看运行详情 + stage 时间线 | `rheo run show <run-id>` |
| 挂起 / 恢复运行 | `rheo run suspend <run-id> --reason <text>` / `rheo run resume <run-id>` |
| 补偿已关闭 / 失败的运行 | `POST /api/v1/instances/{id}/compensate` |
| 查看待发送同步记录 | `rheo sync pending` / `rheo sync ack <id>` |
| 管理分区 | `rheo partition list/register/assign/instances` |
| L2 恢复检查点 | `rheo checkpoint create <run-id>` / `rheo checkpoint list <run-id>` |
| 热迁移运行 | `rheo migrate <run-id> --to <workflow-id>` |
| 重规划运行中实例 | `rheo replan <run-id> --stages a,b,c [--dry-run]` |
| 节点基准测试 | `rheo bench --concurrency 8 --duration 30s --report bench.json` |
| 校验定义 | `rheo workflow validate <file>` |
| 导入 / 导出 BPMN | `rheo workflow import-bpmn <file>` / `rheo workflow export-bpmn <file>` |

完整 CLI 参考：core 仓库 `docs/api.md`（在 `github.com/axisrobo/rheovela`）。

---

## 8. 故障排查

| 症状 | 检查点 |
|------|--------|
| `rheo serve` 起不来 | DB 被占用（已有 serve / CLI 打开）？其他进程锁文件？ |
| `dr verify` 失败 | **不要**用该文件恢复 / 运行；备份不完整或损坏 |
| 审计出现 `[INVALID]` | 写侧签名密钥与读侧验证密钥不一致 |
| Worker 领取总是失败 | 租约太短 / 其他 worker 持项；调大租约或加密心跳 |
| Timer 从不触发 | 传了 `--scheduler-off`，或调度间隔过长 |
| 变更型 HTTP 调用返回 403 | 服务以 `--readonly` 启动 |
| TLS 背后连接重置 | 确认 cert/key 配对；由反向代理终止 TLS |

若某个运行的事件流不一致，先看 `rheo history --run <id> --verify` 与审计
证据链——每条命令都是不可变、带签名的事件。

---

另见：
- **开发手册**：[DEVELOPMENT.md](DEVELOPMENT.md)（English）· [中文](DEVELOPMENT.zh.md)
- **功能特性**：[FEATURES.md](FEATURES.md)（[English](FEATURES.en.md)）
- **HTTP API 参考**：[api/README.md](../api/README.md)（[English](../api/README.en.md)）
- **产品介绍**：[product.md](product.md)（[English](product.en.md)）
