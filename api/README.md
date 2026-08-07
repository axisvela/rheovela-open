# HTTP Ops API — Reference

RHEOVELA core 的单节点 HTTP Ops API 由 `rheo serve` 暴露（`internal/server`，
core `docs/api.md`）。所有路由挂载在 `/api/v1` 下，返回 `application/json`。

> 定位：本地运维 / 集成工具面，**无 TLS / auth shim**。多租户、鉴权与管控面在
> `rheovela-ee`（`X-Tenant-ID` 中间件 + tenant-scoped service）。

## 启动

```sh
rheo serve [--addr :8080] [--db <path>] [--verify-key <hex>] [--scheduler-interval <duration>] [--scheduler-off]
```

| 参数 | 说明 |
|------|------|
| `--addr` | 监听地址（默认 `:8080`） |
| `--db` | SQLite 数据库路径（默认 `~/.proc/proc.db`） |
| `--verify-key <hex>` | HMAC 校验密钥：审计端点签名校验（`GET /api/v1/audit/{id}`） |
| `--scheduler-interval <duration>` | 到期 timer 调度轮询间隔（默认 `1s`） |
| `--scheduler-off` | 关闭调度循环 |

写入路径与 CLI 共用同一 `application.Service`，因此 HTTP 打开的运行与
`run_contexts` 投影原子一致，并受 `idempotency_key` 去重保护。

## 路由总览

| Method | Path | 描述 |
|--------|------|------|
| `GET` | `/api/v1/instances?project=<id>` | 列出流程实例（可按项目过滤） |
| `POST` | `/api/v1/instances` | 打开一个运行（open run） |
| `GET` | `/api/v1/instances/{id}` | 获取单个实例 |
| `POST` | `/api/v1/instances/{id}/steps` | 步骤动作（enter/complete/fail/skip/assign） |
| `GET` | `/api/v1/audit/{id}` | 实例的证据链（含签名校验） |

### 错误格式

所有错误返回 JSON：`{"error": "<message>"}`。`404` 表示实例不存在，`400` 表示
请求体非法 / 非法动作 / 非法状态转换，`500` 表示服务器错误。

---

## `GET /api/v1/instances`

列出流程实例。`project` 查询参数可选；为空时返回全部实例。

**响应 200**：`ProcessInstance[]`（按 `started_at` 倒序）

```json
[
  {
    "instance_id": "R-<uuid>",
    "project_id": "1",
    "definition_id": "PurchaseApproval",
    "definition_version": 1,
    "status": "active",
    "opened_by": "cli-user",
    "variables": {"amount": "1500"},
    "depth": 0,
    "started_at": "2026-08-07T12:00:00Z"
  }
]
```

`ProcessInstance` 字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `instance_id` | string | 实例 ID（stream ID） |
| `project_id` | string | 所属项目 |
| `definition_id` | string | 工作流定义 ID |
| `definition_version` | int | 运行固定的定义版本 |
| `status` | string | `active` / `done` / `failed` / `cancelled` 等 |
| `opened_by` | string | 打开运行的 actor |
| `variables` | map | 运行条件变量（`--var`） |
| `parent_instance_id` | string\|null | 子流程的父实例（subprocess） |
| `depth` | int | 子流程深度（父为 0） |
| `started_at` / `closed_at` | string\|null | 时间戳（RFC3339） |

---

## `POST /api/v1/instances`

打开一个运行。

**请求体**

```json
{
  "workflow": "PurchaseApproval",
  "project": "1",
  "actor": "cli-user",
  "variables": {"amount": "1500"},
  "idempotency_key": "k-<uuid>"
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `workflow` | ✅ | 工作流定义 ID（缺少则 400） |
| `project` |  | 项目 ID |
| `actor` |  | actor ID（写入事件的 `actor_id`） |
| `variables` |  | 条件变量映射 |
| `idempotency_key` |  | 幂等键；省略时服务端自动生成 UUID |

**响应 201**：`ProcessInstance`（见上）。同键重试返回既有实例，不产生重复事件。

---

## `GET /api/v1/instances/{id}`

获取单个实例。

**响应 200**：`ProcessInstance`。**404**：实例不存在。

---

## `POST /api/v1/instances/{id}/steps`

对实例执行一个步骤动作。

**请求体**

```json
{
  "action": "enter",
  "stage": "validate",
  "actor": "cli-user",
  "assigned_to": "bob",
  "error": "",
  "idempotency_key": "k-<uuid>"
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `action` | ✅ | `enter` \| `complete` \| `fail` \| `skip` \| `assign` |
| `stage` | ✅ | 阶段 ID |
| `actor` |  | 执行 actor |
| `assigned_to` |  | 仅 `assign` 使用：指派对象 |
| `error` |  | 仅 `fail` 使用：失败原因 |
| `idempotency_key` |  | 幂等键；省略时自动生成 |

**响应 200**：`{"status": "ok"}`

**错误**：实例不存在 → 404；非法 action / 非法状态转换（`ErrInvalidTransition`）
→ 400。

---

## `GET /api/v1/audit/{id}`

返回实例的**证据链**（`internal/audit`）：plan → events → assignment → execution，
并附带 effect records。启动时带 `--verify-key` 会对每条事件做签名校验。

**响应 200**：

```json
{
  "instance_id": "R-<uuid>",
  "definition_id": "PurchaseApproval",
  "definition_version": 1,
  "status": "active",
  "events": [
    {
      "seq": 1,
      "type": "RunOpened",
      "actor_id": "cli-user",
      "wall_time": "2026-08-07T12:00:00Z",
      "signature": "…",
      "signature_valid": true,
      "payload": {"workflow_id": "PurchaseApproval"}
    }
  ],
  "effects": [
    {
      "idempotency_key": "effect:invoice/R-<uuid>",
      "target": "R-<uuid>",
      "state": "applied",
      "evidence": "…"
    }
  ],
  "stage_executions": [
    {
      "stage_id": "validate",
      "assigned_to": "bob",
      "assigned_by": "cli-user",
      "executed_by": ["cli-user"],
      "completed_by": "cli-user",
      "status": "completed"
    }
  ]
}
```

签名策略：无 verifier 视为全部有效；有 verifier 但事件签名为空 → `signature_valid:
false`；有签名则按 `Verify` 结果判定。**404**：实例不存在。

---

## 租户考虑

- **core（本层）**：单租户、本地运维面，不做鉴权与租户隔离。
- **`rheovela-ee`**：在 core 之上增加 `X-Tenant-ID` 中间件、tenant-scoped
  service 与 `{id}` 路由属主 guard（跨租户一律 404）。若需多租户，请使用 ee 的
  `rheo-ee serve`，而非本层。

## 完整 JSON schema

契约型（`WorkflowDefinition` / `Event` / `ProcessInstance`）定义见
`contracts/types.go`（`schema_version=1`）。CLI 完整命令与 Go API 见 core
`docs/api.md`。
