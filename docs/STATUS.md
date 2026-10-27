# Rheovela-open Status (v1.19.0)

> Last updated: 2026-10-27 · current release `v1.19.0`（feature release；功能基线 `v1.4.0`）· license Apache-2.0
>
> 本文是 open 的对外公开状态。内部发布一致性、企业能力缺口与详细 backlog
> 由 Rheovela-ee 私有仓库 `docs/STATUS.md` 跟踪。

## Current Release

| Item | State |
|------|-------|
| Release tag | `v1.19.0`（feature release；功能基线 `v1.4.0`） |
| Contracts | `schema_version=1`；16 种事件（含 `WorkItemApprovalRequested` / `WorkItemApproved` / `WorkItemRejected`） |
| OpenAPI | `api/openapi.yaml`；覆盖 instances、steps、work-items、suspend/resume、approval、retry/cancel、audit、sync、events、health/status/metrics |
| JSON Schema | `api/workflow.schema.json`、`api/event.schema.json` |
| Worker SDKs | Go / Python / TypeScript / Java / Rust |
| Examples | Go / Python / TypeScript worker examples；expense/purchase RHEO IR samples |

## SDK Feature Matrix

| SDK | Path | Work-item v1.4.0 features |
|-----|------|---------------------------|
| Go client + worker | `sdk/go/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling |
| Python worker | `sdk/python/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling |
| TypeScript worker | `sdk/typescript/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling |
| Java worker | `sdk/java/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling |
| Rust worker | `sdk/rust/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling |

Go SDK 的 work-item 客户端方法包括 `GetWorkItem`、`CompleteWorkItemWithResult`、
`RequestWorkItemApproval`、`ApproveWorkItem`、`RejectWorkItem`、`CancelWorkItem`。

## Verification (2026-10-27, v1.19.0 gate)

```sh
# root module
go test ./... -count=1        # 4 packages ok

# Go SDK module
cd sdk/go
go test ./... -count=1        # 2 packages ok
```

Exact top-level test counts are tracked by CI; historical 15/29 figures are not
the current baseline.

Python / TypeScript / Java / Rust 由各自 CI job 独立验证，没有跨语言统一测试框架。

Release evidence: [SDK v1.4.0 parity](testing/2026-10-26-sdk-v14-parity/README.md)
and [Harmovela adapter](testing/2026-10-26-harmovela-adapter/README.md).

## Public Limitations

| Area | Boundary |
|------|----------|
| Process Designer | 静态社区设计器 `designer/`：编辑/校验/导入导出、BPMN DI 渲染、undo/redo、导入差异预览；可选 load/publish/list 直连 core workflows API |
| Harmovela adapter | producer/observer adapter in `sdk/go/harmovela` (since v1.7.5)；内置 WebSocket/gRPC/NATS transports 不包含；consumer direction 由 core 负责 |
| BPMN / RHEO IR | core 提供 BPMN import/export 子集，保留 gateway/event 语义、root definitions、activities/loops、data stores/IO、resources/roles、categories、collaboration/conversation、choreography、artifacts、data associations、extensionElements/documentation/import/relationship、auditing/monitoring、isInterrupting，以及完整 DI；RHEO IR v1.0 规范已发布（[RHEO-IR.md](RHEO-IR.md)），公开 conformance corpus 与稳定 `RHEOIRxxx` 错误码见 [`conformance/`](../conformance/README.md) |

内部发布细节（`/v3` 模块 tag、三仓库 release tag 对齐）不在公开文档展开。
