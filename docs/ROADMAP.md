# Rheovela-open Public Roadmap

> Last updated: 2026-10-27 · current release `v1.13.0`（feature release；功能基线 `v1.4.0`）
>
> 本文是 open 的对外公开路线图。core 内核路线图见 core `docs/ROADMAP.md`；
> 企业能力与内部发布一致性由 Rheovela-ee 私有仓库 `docs/ROADMAP.md` 跟踪。

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
| Community Process Designer | 静态 `designer/`：WorkflowDefinition 编辑、校验、JSON 导入导出、delegation/evidence 保留、BPMN DI 渲染、undo/redo、导入差异预览；可选直连 core API |

## Next

1. **BPMN / RHEO IR**
   - RHEO IR v1.0 规范已发布（[RHEO-IR.md](RHEO-IR.md)）；delegation/evidence/migration policy/effect_key/timeout 已支持
   - BPMN import/export 的标准覆盖扩展（子流程、消息流等）

## Release Rules

- 使用语义版本 tag（如 `v1.4.0`），不使用 `v1.x-core` 开发里程碑 tag。
- `/v3` Go module 的模块 tag 与 release tag 对应关系由内部发布流程维护。
- 每个 release 更新 `CHANGELOG.md`，并保持 contracts/OpenAPI 与 core 同步。
