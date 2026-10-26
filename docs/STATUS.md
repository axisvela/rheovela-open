# Rheovela-open Status (v1.7.3)

> Last updated: 2026-10-26 · current release `v1.7.3`（coordinated ecosystem release；功能基线 `v1.4.0`）· license Apache-2.0
>
> 本文是 open 的对外公开状态。内部发布一致性、企业能力缺口与详细 backlog
> 由 Rheovela-ee 私有仓库 `docs/STATUS.md` 跟踪。

## Current Release

| Item | State |
|------|-------|
| Release tag | `v1.7.3`（coordinated ecosystem release；功能基线 `v1.4.0`） |
| Contracts | `schema_version=1`；16 种事件（含 `WorkItemApprovalRequested` / `WorkItemApproved` / `WorkItemRejected`） |
| OpenAPI | `api/openapi.yaml`；覆盖 instances、steps、work-items、suspend/resume、approval、retry/cancel、audit、sync、events、health/status/metrics |
| JSON Schema | `api/workflow.schema.json`、`api/event.schema.json` |
| Worker SDKs | Go / Python / TypeScript / Java / Rust |
| Examples | Go / Python / TypeScript worker examples；expense/purchase RHEO IR samples |

## SDK Feature Matrix

| SDK | Path | Work-item v1.4.0 features |
|-----|------|---------------------------|
| Go client + worker | `sdk/go/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling |
| Python worker | `sdk/python/` | 基础：claim / heartbeat / complete / fail；v1.4.0 功能对齐待补 |
| TypeScript worker | `sdk/typescript/` | 基础：claim / heartbeat / complete / fail；v1.4.0 功能对齐待补 |
| Java worker | `sdk/java/` | 基础：claim / heartbeat / complete / fail；v1.4.0 功能对齐待补 |
| Rust worker | `sdk/rust/` | 基础：claim / heartbeat / complete / fail；v1.4.0 功能对齐待补 |

Go SDK 的 work-item 客户端方法包括 `GetWorkItem`、`CompleteWorkItemWithResult`、
`RequestWorkItemApproval`、`ApproveWorkItem`、`RejectWorkItem`、`CancelWorkItem`。

## Verification (2026-10-26, v1.7.3 gate)

```sh
# root module
go test ./... -count=1        # 3 packages ok

# Go SDK module
cd sdk/go
go test ./... -count=1        # 2 packages ok
```

Exact top-level test counts are tracked by CI; historical 15/29 figures are not
the current baseline.

Python / TypeScript / Java / Rust 由各自 CI job 独立验证，没有跨语言统一测试框架。

## Public Limitations

| Area | Boundary |
|------|----------|
| Non-Go SDK parity | Python / TypeScript / Java / Rust 尚未暴露 v1.4.0 的 approval、cancel、retry、result、checkpoint、effect cursor |
| .NET SDK | 尚未发布 |
| Process Designer | 未包含在本仓库 |
| Harmovela adapter | 尚未在 open 落地 |
| BPMN / RHEO IR | core 提供子集实现；完整规范与扩展仍在路线图中 |

内部发布细节（`/v3` 模块 tag、三仓库 release tag 对齐）不在公开文档展开。
