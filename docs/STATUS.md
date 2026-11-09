# Rheovela-open Status (v1.72.0)

> Last updated: 2026-11-08 · current release `v1.72.0`（core/open 共享；EE 独立 tag；work-item 功能参考基线 `v1.4.0`）· license Apache-2.0
>
> 本文是 open 的对外公开状态。内部发布一致性、企业能力缺口与详细 backlog
> 由 Rheovela-ee 私有仓库 `docs/STATUS.md` 跟踪。

## Current Release

| Item | State |
|------|-------|
| Release tag | `v1.72.0`（2026-11-08，core/open 共享；EE 独立 tag；本版 open 无变更，core 修复 CLI 版本回退） |
| Module tags | 根模块保持 `v3.16.0`；`sdk/go/v3.2.0`；adapter `sdk/go/harmovela/v3.0.0`（本地 tag 已核对；不代表本次验证了远端发布/模块代理） |
| Contracts | `AssigneeSpec` candidates/SLA/budget/claim_policy/escalation；`Stage` task_kind/payload/required_evidence/approval_roles/tool_scope/credential_ref/completion_mode/required_approvals/required_completions/**sub_workflow**； `schema_version=1`；事件含 `WorkItemApprovalRequested`/`WorkItemApproved`/`WorkItemRejected`/`WorkItemEscalated`/`WorkItemHandedOff`/`WorkItemTakenOver`/`WorkItemCoCompleted` |
| OpenAPI | `api/openapi.yaml`；覆盖 instances、steps、work-items、handoff/takeover、notifications、approval-delegations、suspend/resume、approval、retry/cancel、audit、sync、events、health/status/metrics |
| JSON Schema | `api/workflow.schema.json`、`api/event.schema.json` |
| Worker SDKs | Go / Python / TypeScript / Java / Rust |
| Examples | Go / Python / TypeScript worker examples；expense/purchase RHEO IR samples |

> Release automation (2026-11-07): the `v1.63.0` tag/commit is pushed, but the
> `release` workflow failed because `CORE_REPO_TOKEN` is not configured; core CI
> and EE privacy jobs are also blocked by GitHub account billing. Binaries and
> the GHCR image were not published; re-run after configuring the secret and
> billing.

`v1.44.0` added RHEO IR participant attributes, SDK claim-actor propagation on
completion, notification retry/dead-letter metadata, and participant/assignment
fields in `api/workflow.schema.json`. `v1.45.0`–`v1.50.0` were coordinated
releases with no open contract or SDK changes; core added the configurable event
bus, feed routing, durable feed routes, tenant-aware feeds, P3 replay/retention,
a CI workflow fix, and stage-level leases.

**Released in v1.51.0:** OpenAPI documents stage lease and worker session
routes, and step/claim bodies accept `lease_token` / `session_token`. The Go,
Python, TypeScript, Java, and Rust SDKs expose the lease/session methods.

`v1.52.0`–`v1.59.0` are coordinated releases without open code changes;
the current release records the EE KINETOVELA live drill. The `v1.4.0`
reference names the work-item feature set, not a freeze of later SDK/API work.

## SDK Feature Matrix

| SDK | Path | Work-item v1.4.0 features |
|-----|------|---------------------------|
| Go client + worker + events | `sdk/go/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling；`events` 包提供 WebSocket + 持久化离线队列 |
| Python worker + events | `sdk/python/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling；`events.py` 提供 WS + 持久化离线队列 |
| TypeScript worker + events | `sdk/typescript/` | 完整：`Result`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling；`events.ts` 使用 Node 内置 WebSocket + 持久化队列 |
| Java worker + events | `sdk/java/` | 完整：`Result`、`Evidence`、`Cost`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling、claim guards；`Events.java` 使用 JDK WebSocket + 持久化队列 |
| Rust worker + events | `sdk/rust/` | 完整：`Result`、`Evidence`、`Cost`、`CheckpointRef`、`EffectCursor`、approval、retry、cancel、claimable polling、claim guards；`events` 模块提供 std-only WS + 持久化队列 |

Go SDK 的 work-item 客户端方法包括 `GetWorkItem`、`CompleteWorkItemWithResult`、
`RequestWorkItemApproval`、`ApproveWorkItem`、`RejectWorkItem`、`CancelWorkItem`。

## Historical Verification (2026-11-01, v1.50.0 gate)

The commands below preserve the recorded v1.50.0 gate; they do not verify
the later v1.51.0 lease/session SDK changes or the current checkout. No SDK
test suite was rerun for this documentation reconciliation.

```sh
# root module
go test ./... -count=1        # 4 packages ok

# Go SDK module
cd sdk/go
go test ./... -count=1        # 2 packages ok
```

Exact current test counts require a fresh run; historical 15/29 figures and
the package counts above are not a new v1.59.0 verification claim.

Python / TypeScript / Java / Rust 由各自 CI job 独立验证，没有跨语言统一测试框架。

Release evidence: [SDK v1.4.0 parity](testing/2026-10-26-sdk-v14-parity/README.md)
and [Harmovela adapter](testing/2026-10-26-harmovela-adapter/README.md).

## Public Limitations

| Area | Boundary |
|------|----------|
| Process Designer | 静态社区设计器 `designer/`：编辑/校验/导入导出、BPMN DI 渲染、undo/redo、导入差异预览；可选 load/publish/list 直连 core workflows API；CRDT 客户端核心（`designer/crdt.mjs`：LWW op merge、快照合并、离线重放队列） |
| Harmovela adapter | producer/observer adapter in `sdk/go/harmovela` (since v1.7.5)；内置 WebSocket/gRPC/NATS transports 不包含；consumer direction 由 core 负责 |
| BPMN / RHEO IR | core 提供 BPMN import/export 子集，保留 gateway/event 语义、root definitions、activities/loops、data stores/IO、resources/roles、categories、collaboration/conversation、choreography、artifacts、data associations、extensionElements/documentation/import/relationship、auditing/monitoring、isInterrupting，以及完整 DI；RHEO IR v1.0 规范已发布（[RHEO-IR.md](RHEO-IR.md)），公开 conformance corpus 与稳定 `RHEOIRxxx` 错误码见 [`conformance/`](../conformance/README.md) |

内部发布细节（`/v3` 模块 tag、三仓库 release tag 对齐）不在公开文档展开。
