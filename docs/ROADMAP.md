# Rheovela-open Public Roadmap

> Last updated: 2026-10-25 · current release `v1.7.2`（coordinated ecosystem release；功能基线 `v1.4.0`）
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
| Durable worker (v1.4.0) | Go SDK：result、checkpoint、effect cursor、approval、retry、cancel、claimable polling |

## Next

1. **SDK parity for v1.4.0**
   - Python / TypeScript / Java / Rust 对齐 approval、cancel、retry、result、checkpoint、effect cursor、claimable polling
   - 统一各语言 worker 的能力矩阵与测试覆盖
2. **.NET SDK**
   - 第六种语言 worker/client SDK
3. **Harmovela adapter**
   - 跨组织协调与观察 fan-out 的 open 侧适配
4. **BPMN / RHEO IR**
   - 完整 RHEO IR 规范发布到 open
   - BPMN import/export 的标准覆盖扩展（子流程、消息流等）
5. **Community Process Designer**
   - 社区版流程设计器与 OpenAPI/contracts 对齐

## Release Rules

- 使用语义版本 tag（如 `v1.4.0`），不使用 `v1.x-core` 开发里程碑 tag。
- `/v3` Go module 的模块 tag 与 release tag 对应关系由内部发布流程维护。
- 每个 release 更新 `CHANGELOG.md`，并保持 contracts/OpenAPI 与 core 同步。
