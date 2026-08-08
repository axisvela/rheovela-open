# RHEOVELA 功能与特性

> [English](FEATURES.en.md) · 中文

RHEOVELA 是一个 **Dynamic Process & Durable Workflow Platform（动态流程与持久化工作流平台）**：把治理后的 Capability Plan 物化为可持久、可恢复、可审批、可迁移、可审计的 Process Instance。它负责"长期工作如何可靠流转"，跨越人、Agent、机器人与企业系统。

本仓库（rheovela-open）是 Apache-2.0 对外发布层：版本化契约、SDK 与 API。内核实现位于 [rheovela](https://github.com/axisrobo/rheovela)（AGPL-3.0），企业能力位于 [rheovela-ee](https://github.com/axisrobo/rheovela-ee)（Enterprise）。

---

## 核心特性

### 1. 事件溯源 + 确定性内核
- 所有状态变更由已验证的 Command 产生不可变 Event（append-only）；投影可随时从 `definition snapshot + ordered events` 重建，结果完全一致。
- 纯函数折叠（Fold）不依赖网络/时钟/随机数；任意合法事件序列重放结果确定。
- **15 种事件类型**：RunOpened / StepEntered / StepCompleted / StepFailed / StepSkipped / RunClosed / StageAssigned / TimerFired / Migrated / SubprocessStarted / CompensationExecuted / ProcessSuspended / ProcessResumed 等。

### 2. 版本化定义与发布校验
- `definition_versions` 不可变：同一 digest 幂等，新内容产生新版本；运行实例固定绑定版本。
- 定义发布前校验：stages 非空且唯一、transition 引用已声明节点、**无环**、条件语法可解析。

### 3. 原子 Command 管线（幂等）
- 每个 Command 携带 idempotency key + actor + digest；**事件、投影、幂等键、outbox 同一事务**。
- 同 key 同 digest 重放返回相同结果；不同 digest 报冲突；并发重复被唯一约束拒绝。

### 4. 工作项与租约（Claim/Lease/Fencing）
- Work Item 可被人/Agent/服务/机器人领取：claim 获得 lease + fencing token；心跳续租；过期可重派；**旧 token 完成被拒**（防止双执行）。

### 5. Durable Timer
- 定时器持久化，进程重启后仍触发且**只触发一次**（原子 fire-and-mark）。支持 deadline / duration / business calendar。

### 6. Effect Ledger（副作用完整性）
- 外部副作用先登记 intent、后记录 outcome；同 idempotency key 不重复执行；unknown outcome 进入 reconciliation，不盲目重试。
- Effect Integrity 模型，而非"神话式 exactly-once"。

### 7. 恢复与故障注入
- **RHEO-Bench 10/10**：B1 重放 / B2 崩溃矩阵 / B3 重复与乱序 / B4 分支 DAG / B5 计时器 / B6 worker 丢失 / B7 effect 未知 / B8 补偿 / B9 迁移 / B10 撤销。
- Checkpoint（L2 恢复点）：事件位置 + 折叠快照，`RebuildProjection` 快路径。
- crash-matrix 测试：重启、acknowledged 命令不丢失、投影重建、seq 冲突。

### 8. 分支 / DAG / 子流程
- token 激活内核：条件分支未选节点标记 `Bypassed` 不阻塞完成；AND-join 满足全部依赖才就绪；条件错误进入 Waiting 而非静默分支。
- **子流程展开**：父-子关联（`parent_instance_id` + depth）、深度限制、预算/授权衰减。

### 9. 补偿 / 迁移 / Replan
- 补偿图：按成功顺序逆序执行，手动回退兜底；每动作产生 `CompensationExecuted` 审计事件。
- 迁移：dry-run 兼容性分析（增量为兼容，删除/条件变更拒绝），运行实例可在线迁移到新定义版本。
- Replan handoff：生成修订定义 → dry-run → 迁移，全程 idempotent。

### 10. 挂起 / 恢复
- `ProcessSuspended` / `ProcessResumed` 事件；挂起期间拒绝 step/close 命令，恢复后继续。

### 11. 证据链与签名
- `audit.Build` 从业务结果反查：定义 → 授权 → 分派 → 执行 → effect。
- 事件签名：HMAC 可选；**链式签名**（签名覆盖前序事件）可检测重排/删除。
- `history --verify` / `audit export` / `GET /api/v1/audit/{id}`。

### 12. 治理与保留
- legal hold 排除 retention 清理；retention 按 MaxAge 清理 closed 实例投影（事件保留）；分区注册与实例归属；`ha_locks` 租约锁（HA 基座）。

### 13. 边缘同步（Edge Sync）
- outbox 与事件同事务；`sync/delta`（增量拉取）→ `sync/ack`（确认）→ `sync/pending`（待发清单）；事件流为权威来源，outbox 丢失可重建。

### 14. 多目标接口
- **CLI**（25+ 命令）：workflow define/validate/diff/import-bpmn/export-bpmn、run open/step/close/suspend/resume/list、checkpoint、migrate/replan、audit、history、serve/watch、sync、partition、bench、dr、auth、ir。
- **HTTP Ops API**（40+ 端点）：instances / steps / suspend/resume / compensate / workflows / work-items / audit / sync / events(SSE) / health / status / metrics。
- **MCP gateway**：`POST /mcp` JSON-RPC 2.0，暴露 open_run / step / evidence 等 8 个工具，供 Agent 调用。
- **Worker API**：claim / heartbeat / complete / fail。

### 15. 身份
- `rheo auth`：OIDC Device Flow（RFC 8628）登录 + whoami。
- `identity.Identity` 端口（AEGIVELA 抽象）：静态提供者 / Keycloak（企业）。

### 16. 开放架构（open-core）
- 公共 `runtime` 包 = 内核契约面（Store/Service/Identity/Signer/审计），ee 与 SDK 无需 import `internal/*`。
- **5 语言 Worker SDK**：Go（嵌入）/ Python / TypeScript / Java / Rust；HTTP Worker API 客户端。
- 版本化 JSON Schema：`workflow.schema.json`、`event.schema.json`；OpenAPI：`openapi.yaml`。
- BPMN 2.0 子集 import/export（task/gateway/事件/条件，往返一致）。

---

## 快速开始

```sh
go build -o rheo ./cmd/rheo        # 或 release/build.ps1
rheo workflow define --file examples/expense.json
rheo run open expense --var amount=500
rheo step enter validate --as alice
rheo step exit validate --as alice
rheo run close --outcome done
rheo serve --addr :8080
```

详见 [docs/product.md](docs/product.md) 与 [docs/BETA.md](../Rheovela/docs/BETA.md)（Beta 发布手册与验收标准）。
