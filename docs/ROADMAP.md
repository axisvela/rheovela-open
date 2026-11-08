# Rheovela-open Public Roadmap

> Last updated: 2026-11-08 · current release `v1.68.0`（core/open 共享；EE 独立 tag；work-item 功能参考基线 `v1.4.0`）
>
> 本文是 open 的对外公开路线图。core 内核路线图见 core `docs/ROADMAP.md`；
> 企业能力与内部发布一致性由 Rheovela-ee 私有仓库 `docs/ROADMAP.md` 跟踪。

## Product Boundaries

open 负责开放契约、SDK、社区 Designer 与 adapter，不实现 durable runtime 内核或企业协调控制面。core 必须保持无需 EE 即可独立运行；企业产品消费 core/open 的接口，不改变已有社区能力和许可证。

| 里程碑 | 产品归属 | Open 配套范围 |
|---|---|---|
| M0 · 认证边界 | core / EE 各自负责 | 契约与客户端说明，不实现运行时认证 |
| M1 · 图形 Dashboard | EE | 社区 Designer、模型解析复用；企业实例看板由 EE 提供 |
| M2 · SLA / business calendar | EE | 按需提供数据契约与 SDK；日历计算和 SLA 策略管理由 EE 提供 |
| M3 · Local → Coordinated | EE | 按需提供交接契约；core 保留独立运行和最小安全交接原语，EE 编排切换与回滚 |
| M4 · Connector marketplace | EE | 按需公开 manifest、SDK、示例；registry 和安装/启停/配置管理由 EE 提供 |
| M5 · Process mining / analytics | EE | 按需提供分析数据与客户端契约；分析查询、conformance、报表由 EE 提供 |
| M6 · CRDT 协同编辑 | EE 主导、open 配套 | Designer 文档模型、CRDT 客户端、合并/重连；鉴权、租户隔离、共享持久化、在线状态与协同服务由 EE 提供 |
| M7 · Federated | EE 主导、open 配套 | 跨组织消息契约、SDK、Harmovela/LIMENORA adapter；组织信任配置、协调与运营由 EE 提供 |

M6 编辑的是设计稿，不直接修改已发布的不可变定义或运行实例；发布与迁移仍须走 core 的校验、版本和审批/审计规则。M7 的开放协议和 adapter 不等于完整企业联邦服务。以上是职责边界，不是交付或生产验收声明。

Core 公共接口与边界见 [core Roadmap](https://github.com/axisvela/rheovela/blob/main/docs/ROADMAP.md)；企业产品范围、依赖与验收 gate 由 Rheovela-ee 私有仓库 `docs/ROADMAP.md` 和 `docs/BACKLOG-EXECUTION-PLAN.md` 维护。

## Completed

| Area | Delivered |
|------|-----------|
| Contracts | Phase 0 domain schema、事件目录、`schema_version=1` |
| Worker SDKs | Go / Python / TypeScript / Java / Rust |
| HTTP API | OpenAPI 3.0、Worker HTTP API、work-item approval/retry/cancel |
| Schemas | WorkflowDefinition JSON Schema、Event JSON Schema |
| Examples | Go / Python / TypeScript runnable workers、RHEO IR samples |
| SDK parity for v1.4.0 | Go / Python / TypeScript / Java / Rust SDK：result、checkpoint、effect cursor、approval、retry、cancel、claimable polling（v1.7.5） |
| Harmovela adapter | `sdk/go/harmovela` producer/observer adapter：将 Rheovela work-item 生命周期映射为 Harmovela `task.*` 事件（v1.7.5）；仅 producer/observer，consume 方向由 core 负责，内置 transport 不包含 |
| Community Process Designer | 静态 `designer/`：WorkflowDefinition 编辑、校验、JSON 导入导出、delegation/evidence 保留、BPMN DI 渲染、undo/redo、导入差异预览；可选直连 core API；CRDT 客户端核心（op-based LWW、快照合并、离线重放队列） |
| RHEO IR participant attributes | v1.44.0：`approval_roles`/`tools`/`credential`/`completion`/`approvals`/`completions` 活动属性与 conformance fixture |
| Participant/assignment schema | v1.44.0：`api/workflow.schema.json` 覆盖 `task_kind`/`payload`/`required_evidence`/`approval_roles`/`tool_scope`/`credential_ref`/`completion_mode`/counts 与 candidates/SLA/budget/claim policy/escalation |
| Coordinated releases | v1.45.0–v1.51.0：core 事件总线、feed 路由、durable feed routes、tenant-aware feeds、P3 replay/retention、CI 修复、stage-level lease；open 在 v1.51.0 新增 stage/session lease SDK 与 OpenAPI |
| Current release | v1.68.0（core/open 共享）：open 无代码变更；core 增加 position-axis `RunAuthority` 命令门；契约保持 v1.60.0 的 Stage.SubWorkflow / SubWorkflowRef。实现、验证与生产验收分开记录，详见 [STATUS.md](STATUS.md) |

## Next

> 内部执行顺序与依赖见 Rheovela-ee `docs/BACKLOG-EXECUTION-PLAN.md`。M6/M7 是 EE 主导的企业能力，open 仅负责上表中的客户端、契约与 adapter 面；认证由 EE 的 AEGIVELA/EIDOVELA 承担。

1. **BPMN / RHEO IR**
   - RHEO IR v1.0 规范已发布（[RHEO-IR.md](RHEO-IR.md)）；delegation/evidence/migration policy/effect_key/timeout 已支持
   - BPMN import/export 的标准覆盖扩展（子流程、消息流等）
2. **SDK parity**
   - 跟随 core 的 work-item/event API 演进（notification/feed/event-bus 相关接口按需加入 SDK 与 OpenAPI）
   - ✅ OpenAPI + Go/Python/TypeScript/Java/Rust：stage/session lease、`lease_token`/`session_token`
3. **M6 / M7 配套**
   - M6：Designer 文档模型与 CRDT 客户端（`designer/crdt.mjs` 已落地 op-based LWW、快照合并、离线重放；鉴权、租户隔离、共享持久化与在线状态仍属 EE），保持设计稿、发布版本和运行实例分离
   - M7：跨组织消息契约、SDK 与 adapter；不在 open 实现组织信任管理或联邦控制面

## Release Rules

- 使用语义版本 tag（如 `v1.4.0`），不使用 `v1.x-core` 开发里程碑 tag。
- core 与 open 使用同一 release tag；EE 可独立打 tag。`/v3` Go module 的模块 tag 与 release tag 对应关系由内部发布流程维护；当前 `v1.63.0` 未切新 module tag，根模块仍为 `v3.15.0`、`sdk/go/v3.2.0`、adapter `sdk/go/harmovela/v3.0.0`。本地 tag 存在不等于远端发布或无 replace 模块解析已通过。
- 每个 release 更新 `CHANGELOG.md`，并保持 contracts/OpenAPI 与 core 同步。
