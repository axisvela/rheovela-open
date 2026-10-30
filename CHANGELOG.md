# Changelog

RHEOVELA open（`github.com/axisvela/rheovela-open`，Apache-2.0）发布记录：对外发布层（版本化 contracts、SDK、examples、CI、OpenAPI）。格式：最新在前；每个版本对应一个 git tag（三仓库统一语义版本 tag）。

## [v1.42.0] - 2026-10-30

### Added

- OpenAPI: complete BPMN execution coverage (start/get/complete/fail/events/
  advance/compensate/choreography) plus BPMN state and request schemas; spec
  version moved to `v1.41.0`.
- OpenAPI: health/status/metrics, workflow definitions + BPMN import preview,
  and sync delta/pending/ack endpoints.
- Contracts: `AssigneeSpec` gains `candidates`, `sla`, `budget`,
  `claim_policy`, and `escalation` for the work assignment contract extension.
### Changed

- Coordinated ecosystem release with core `v1.42.0` and EE `v1.42.0`. The open `/v3` module remains `v3.12.0`.

## [v1.43.0] - 2026-10-30

### Added

- Contracts: `Stage` gains `task_kind`, `payload`, `required_evidence`, and
  `approval_roles` for Wave 2 participant contracts.
- OpenAPI: work items expose `kind`/`payload`/`required_evidence`/`evidence`;
  the complete endpoint accepts `evidence`.
- OpenAPI: work items expose `deadline`/`escalation`/`escalated_at`/`approval_roles`,
  and `GET /api/v1/work-items` accepts `overdue=true` for SLA breach queries.
- Contracts: `WorkItemEscalated` is a known audit event emitted by automatic
  SLA escalation.
- Contracts: `Stage` gains `tool_scope`, `credential_ref`, `completion_mode`,
  `required_approvals`, and `required_completions`; `WorkItemEscalated`,
  `WorkItemHandedOff`, `WorkItemTakenOver`, and `WorkItemCoCompleted` are known
  audit events.
- OpenAPI: work items expose budget/spent/tool scope/credential binding,
  completion mode, completions, and approval decisions; claim accepts
  actor/tools/credential; complete/fail accept cost; handoff/takeover endpoints.
- OpenAPI: notification list/ack and approval-delegation create/list/revoke
  endpoints for handoff/escalation delivery and delegated approvals. Claim
  `actor` matches `role:<name>` targets by role membership.
- Go/Python/TypeScript/Java/Rust SDKs: workers support claim guards
  (actor/tools/credential), completion `evidence`/`cost`, and the expanded
  work-item contract; HTTP clients forward both.

### Changed

- Coordinated ecosystem release with core `v1.43.0` and EE `v1.43.0`. The open
  `/v3` module is tagged `v3.13.0`; the Go SDK module is tagged `v3.2.0`.

## [v1.41.0] - 2026-10-30

### Added

- Java SDK `sdk/java/worker/Events.java`: durable WebSocket event client using
  the JDK built-in `java.net.http.WebSocket` and a file-backed pending queue.
- OpenAPI: stage/BPMN scope pause, `/sync/push`, filtered SSE and WebSocket
  event streams, `held` work-item state, and new pause/sync schemas. The spec
  is now OpenAPI 3.0.3-valid.
- Rust SDK `sdk/rust/src/events.rs`: std-only durable WebSocket event client
  (ws://) with the same cursor/pending-queue semantics.

### Changed

- Coordinated ecosystem release with core `v1.41.0` and EE `v1.41.0`. The open `/v3` module remains `v3.12.0`.

## [v1.40.0] - 2026-10-30

### Added

- Go SDK `sdk/go/events`: durable WebSocket event client with filtered
  `types`/`correlation_key`/`predicate` subscription, `after_seq` reconnect,
  exponential backoff, and a file-backed at-least-once pending queue that
  persists acks across restarts.
- Python SDK `sdk/python/events.py`: durable WebSocket event client with the
  same cursor/pending-queue semantics (`websockets` optional dependency, see
  `requirements-events.txt`).
- TypeScript SDK `sdk/typescript/events.ts`: durable WebSocket event client
  using the Node 22+ built-in WebSocket and a file-backed pending queue.

### Changed

- Contracts: `RunContext` gains `suspended_stages` for stage-region pause.
- Coordinated ecosystem release with core `v1.40.0` and EE `v1.40.0`. The open
  `/v3` module remains `v3.12.0`.

## [v1.39.0] - 2026-10-29

### Changed

- Coordinated ecosystem release with core `v1.39.0` and EE `v1.39.0`. No open code changes; the open `/v3` module remains `v3.12.0`.

## [v1.38.0] - 2026-10-28

### Changed

- Coordinated ecosystem release with core `v1.38.0` and EE `v1.38.0`. No open code changes; the open `/v3` module remains `v3.12.0`.

## [v1.37.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.37.0` and EE `v1.37.0`. No open code changes; the open `/v3` module remains `v3.12.0`.

## [v1.36.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.36.0` and EE `v1.36.0`. No open code changes; the open `/v3` module remains `v3.12.0`.

## [v1.35.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.35.0` and EE `v1.35.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.34.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.34.0` and EE `v1.34.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.33.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.33.0` and EE `v1.33.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.32.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.32.0` and EE `v1.32.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.31.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.31.0` and EE `v1.31.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.30.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.30.0` and EE `v1.30.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.29.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.29.0` and EE `v1.29.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.28.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.28.0` and EE `v1.28.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.27.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.27.0` and EE `v1.27.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.26.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.26.0` and EE `v1.26.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.25.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.25.0` and EE `v1.25.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.24.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.24.0` and EE `v1.24.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.23.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.23.0` and EE `v1.23.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.22.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.22.0` and EE `v1.22.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.21.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.21.0` and EE `v1.21.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.20.0] - 2026-10-27

### Changed

- Coordinated ecosystem release with core `v1.20.0` and EE `v1.20.0`. No open
  code changes; the open `/v3` module remains `v3.12.0`.

## [v1.19.0] - 2026-10-27

### Added

- `WorkflowDefinition.execution_mode`; `execution_mode=bpmn` opts into BPMN
  execution.
- Core BPMN graph compiler and deterministic token engine for start/end events,
  tasks, exclusive gateways (conditions/default flow), and parallel
  fork/join gateways.

### Changed

- Coordinated ecosystem release with core `v1.19.0` and EE `v1.19.0`. The open
  `/v3` module is tagged `v3.12.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.18.0] - 2026-10-27

### Added

- BPMN `messageEventDefinition.operationRef`, link-event `source`/`target`
  references, and `sequenceFlow.isImmediate`.

### Changed

- Coordinated ecosystem release with core `v1.18.0` and EE `v1.18.0`. The open
  `/v3` module is tagged `v3.11.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.17.0] - 2026-10-27

### Added

- BPMN definitions/process/activity `extensionElements` (structured tree),
  `documentation`, `import`, `relationship`, `auditing`/`monitoring`, and
  start-event `isInterrupting`.

### Changed

- Coordinated ecosystem release with core `v1.17.0` and EE `v1.17.0`. The open
  `/v3` module is tagged `v3.10.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.16.0] - 2026-10-27

### Added

- BPMN collaboration/conversation metadata: `collaboration`, `conversation`,
  `subConversation`, `callConversation`, `conversationLink`.
- BPMN complex-gateway `activationCondition`, multi-instance `behavior`
  references, and ad-hoc `completionCondition`.

### Changed

- Coordinated ecosystem release with core `v1.16.0` and EE `v1.16.0`. The open
  `/v3` module is tagged `v3.9.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.15.0] - 2026-10-27

### Added

- BPMN root definitions: `message`, `signal`, `error`, `escalation`,
  `interface`/`operation`, and `endPoint`.

### Changed

- Coordinated ecosystem release with core `v1.15.0` and EE `v1.15.0`. The open
  `/v3` module is tagged `v3.8.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.14.0] - 2026-10-27

### Added

- BPMN `ioSpecification`/`inputSet`/`outputSet` (process and activity level).
- Resource parameter bindings and resource assignment expressions.
- Participant `interfaceRef`/`endPointRef` and data-object `dataState`.

### Changed

- Coordinated ecosystem release with core `v1.14.0` and EE `v1.14.0`. The open
  `/v3` module is tagged `v3.7.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.13.0] - 2026-10-27

### Added

- BPMN resources and resource roles (`resource`, `performer`,
  `humanPerformer`, `potentialOwner`, `resourceRole`).
- Process-level `dataInput`/`dataOutput` metadata.
- BPMN `category`/`categoryValue`; `group.categoryValueRef` round-trips.
- `callChoreography` `calledChoreographyRef` and `participantAssociation`.

### Changed

- Coordinated ecosystem release with core `v1.13.0` and EE `v1.13.0`. The open
  `/v3` module is tagged `v3.6.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.12.0] - 2026-10-27

### Added

- BPMN DI completeness: plane `bpmnElement`/`resolution`, label bounds and
  `BPMNLabelStyle`, shape attributes (`isExpanded`, `isMarkerVisible`,
  `isHorizontal`, `isMessageVisible`, `participantBandKind`) and edge attributes
  (`sourceElement`, `targetElement`, `messageVisibleKind`).
- BPMN participant multiplicity; `messageFlow` `name`/`messageRef`.
- BPMN `callActivity`, `transaction`, `adHocSubProcess`, `eventSubProcess`,
  `sendTask`/`receiveTask`/`manualTask`, `calledElement`, `triggeredByEvent`,
  `isForCompensation`, and standard/multi-instance loop characteristics.
- BPMN data stores, data-store references, properties, item definitions,
  `dataObject.itemSubjectRef`/`isCollection`, root event definitions and
  `eventDefinitionRef`.
- BPMN text annotations, associations and groups; ad-hoc ordering/cancel
  attributes, transaction method and activity quantity attributes.

### Changed

- Coordinated ecosystem release with core `v1.12.0` and EE `v1.12.0`. The open
  `/v3` module is tagged `v3.5.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.11.0] - 2026-10-27

### Added

- Designer: render preserved BPMN DI shapes/edges, undo/redo with keyboard
  shortcuts, and import preview diffs before applying JSON/BPMN imports.

### Changed

- Coordinated ecosystem release with core `v1.11.0` and EE `v1.11.0`. The open
  `/v3` module is tagged `v3.4.1`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.10.0] - 2026-10-27

### Added

- Public RHEO IR v1.0 conformance corpus under `conformance/rheo-ir/v1.0/`,
  embedded through `conformance.RHEOIRv1` and covered by a structure test.
- Stable `RHEOIRxxx` parse error codes and v1.0 syntax boundaries documented in
  `docs/RHEO-IR.md`.
- `BPMNNode` gateway fields (`gateway_direction`, `event_gateway_type`,
  `instantiate`, `default_flow`), event fields (`cancel_activity`,
  `parallel_multiple`, `direction`, `event_definitions`) and
  `BPMNEventDefinition`; `BPMN.flows` and `BPMNFlow.name` preserve top-level
  sequence flows. Updated `api/workflow.schema.json`.
- `BPMNProcess`, `BPMNParticipant.process_ref`, `BPMNNode.process_id`,
  `BPMNChoreography`/`BPMNChoreographyTask`,
  `BPMNDataAssociation`/`BPMNAssignment` and the matching
  `BPMN.processes`/`choreographies`/`data_associations` collections.

### Changed

- Coordinated ecosystem release with core `v1.10.0` and EE `v1.10.0`. The open
  `/v3` module is tagged `v3.4.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.9.0] - 2026-10-27

### Documentation

- Added `docs/TESTING.md` with per-language test commands, CI job names, and
  observed counts; added `tools/test-all.ps1`/`tools/test-all.sh`.

### Added

- `WorkflowDefinition.bpmn.diagram` preserves BPMN DI shapes, bounds, edges and
  waypoints for import/export round-trips.
- RHEO IR `migration <activity> policy "preserve|ignore|compensate"` compiles to
  `Stage.migration_policy`; migration analysis honors the per-stage policy.
- RHEO IR `effect_key` and `timeout` now compile to `Stage.effect_key` and
  `Stage.timeout`.
- Designer can import BPMN XML through the core preview endpoint
  (`POST /api/v1/workflows/import-bpmn`) and load the result into the editor.

### Changed

- Coordinated ecosystem release with core `v1.9.0` and EE `v1.9.0`. The open
  `/v3` module is tagged `v3.3.0`; the Go SDK and Harmovela adapter module tags
  are unchanged.

## [v1.8.0] - 2026-10-27

### Added

- `WorkflowDefinition.compensations` carries declared activity/capability
  compensation actions, compiled from RHEO IR `compensate` statements.
- `WorkflowDefinition.bpmn` preserves subprocess internals, boundary/intermediate
  event kinds, gateway types, participants, message flows, lanes and data
  objects for full BPMN import/export round-trips.
- RHEO IR `delegate` and `evidence` declarations compile to
  `Stage.delegation` and `Stage.evidence`.
- Static Community Process Designer (`designer/`) for WorkflowDefinition JSON:
  stage/transition editing, per-stage assignee/evidence/delegation editing, DAG
  preview, validation, import/export, and delegation/evidence preservation. It
  can optionally load/publish/list through the core workflows API. Node model
  and core-client tests run in CI.

### Documentation

- Published the normative RHEO IR v1.0 specification in `docs/RHEO-IR.md`,
  covering grammar, compilation to `WorkflowDefinition`, validation, execution,
  BPMN projection, and reserved extensions.

## [v1.7.5] - 2026-10-26

### Added

- Python, TypeScript, Java, and Rust worker SDKs now match the Go SDK's
  v1.4.0 work-item surface: rich results, checkpoint/effect cursors, approval,
  cancel, claimable polling, and attempt metadata.
- New `sdk/go/harmovela` producer/observer adapter publishes Rheovela work-item
  lifecycle as Harmovela `task.*` events through an injected Publisher.

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.7.5`.
  No contracts or OpenAPI changes.

## [v1.7.4] - 2026-10-26

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.7.4`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.7.3] - 2026-10-26

### Documentation

- Coordinated ecosystem patch release with core and Rheovela-ee `v1.7.3`.

### Fixed

- Release workflow now requires `CORE_REPO_TOKEN` to check out the private core
  repository and fails with a clear preflight error when the secret is missing.

## [v1.7.2] - 2026-10-25

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.7.2`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.7.1] - 2026-10-25

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.7.1`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.7.0] - 2026-10-24

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.7.0`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.6.1] - 2026-10-24

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.6.1`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.6.0] - 2026-10-24

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.6.0`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.5.3] - 2026-10-24

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.5.3`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.5.2] - 2026-10-24

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.5.2`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.5.1] - 2026-10-24

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.5.1`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.5.0] - 2026-10-24

### Documentation

- Coordinated ecosystem release with Rheovela core and Rheovela-ee `v1.5.0`.
  No contracts, OpenAPI or SDK changes in this release.

## [v1.4.1] - 2026-10-24

### Documentation

- Added `docs/STATUS.md` and `docs/ROADMAP.md`.
- Aligned README/product/operations/development version references with
  `v1.4.0` and documented the v1.4.0 SDK feature matrix.

## [v1.4.0] - 2026-10-24

- Core adds the work-item retry policy (attempts / max_attempts): a failed
  item re-queues while attempts remain and becomes terminal at the bound.
- Core adds cancel (POST /api/v1/work-items/{id}/cancel); the Go SDK exposes
  CancelWorkItem.
- Durable worker SDK: `WorkItem`/`WorkResult` now carry `Result`,
  `CheckpointRef`, `EffectCursor`, and `Approval`; optional `ResultStore` and
  `ApprovalStore` ports; `Status == "approval"` requests human approval.
- Client methods: `GetWorkItem`, `CompleteWorkItemWithResult`,
  `RequestWorkItemApproval`, `ApproveWorkItem`, `RejectWorkItem`; claim response
  includes `work_item`.
- `HTTPWorkStore` polls claimable items (ready + expired claimed) and maps
  result/checkpoint/effect/approval fields.
- Work-item events registered in contracts/OpenAPI: `WorkItemApprovalRequested`,
  `WorkItemApproved`, `WorkItemRejected`.

## [v1.1.0] - 2026-08-13

随 core `v1.1.0`（Postgres 支持）同步发布：文档补齐**运维手册**
（`docs/OPERATIONS.md`，含 Postgres 后端小节）与**开发手册**
（`docs/DEVELOPMENT.md`）；README 改为英文为主 + 中文链接
（`README.zh.md`）。

## [v0.9.0-beta] - 2026-08-08

首个 **beta** 发布面（tag 与 v1.14-core 同指向）：SDK 覆盖 **5 种语言**
（Go / Python / TypeScript / Java / Rust）。新增 **Java worker SDK**
（`sdk/java/`：JDK 21、零依赖 `java.net.http.HttpClient`，
`Worker.poll/claim/complete/fail/processOnce`）与 **Rust worker SDK**
（`sdk/rust/`：std-only、零外部 crate、`std::net::TcpStream` 极简 HTTP/1.1
client），并新增 **java / rust CI jobs**——`.github/workflows/ci.yml` 现有
4 个 job（go / python / java / rust）。

## [v1.14-core] - 2026-08-08

**Rust worker SDK**（W24.5，`sdk/rust/`：`Client` / `Worker` / `process_once`，
离线可构建）+ **Rust CI job**（`cargo build --offline` + `cargo run --offline`）。

## [v1.13-core] - 2026-08-08

**Worker SDK 后台心跳**（W23.2）：`sdk/worker.Worker` 在长任务执行期间于后台
续租（`worker_test.go` 增加 heartbeat 用例），避免长任务租约过期被重新领取。

## [v1.12-core] - 2026-08-08

**Event JSON Schema**（W21.1，`api/event.schema.json`，draft-07，覆盖 **13 种事件
类型**）+ `api/schema_test.go` 一致性测试（`TestEventSchema`）。

## [v1.11-core] - 2026-08-08

**WorkflowDefinition JSON Schema**（W20.1，`api/workflow.schema.json`，draft-07）+
conformance 测试（`TestWorkflowSchemaValidatesSample` 等）；`examples/README.md`
声明示例 JSON 均符合该 schema。

## [v1.10-core] - 2026-08-08

Runnable examples 补齐：新增 **Python worker example**（`examples/python-worker/main.py`）与 **TypeScript worker example**（`examples/typescript-worker/worker-example.mjs`），与 Go example（v1.9-core）构成三语言可运行样例（`examples/README.md`）。

## [v1.9-core] - 2026-08-08

Runnable **Go worker example**（`examples/go-worker/main.go` + `main_test.go`，in-memory store），示例流程可直接运行并纳入 `go test`（3 包 ok：contracts / sdk/worker / examples/go-worker）。

## [v1.8-core] - 2026-08-07

OpenAI 发布面扩展：**TypeScript worker SDK**（`sdk/typescript/worker.ts` + `worker.test.ts`，零依赖、Node 18+ `fetch`）、**OpenAPI 3.0 规范**（`api/openapi.yaml`，700 行）、**双语 README**（中文 + English 快速开始）。

## [v1.7-core] - 2026-08-07

**Python worker SDK**（`sdk/python/worker.py`：stdlib-only `Client`/`Worker`）+ **Python CI job**（`.github/workflows/ci.yml`，`unittest` 跑 `sdk/python` 测试）。

## [v1.6-core] - 2026-08-07

Contracts catalog 补齐 + 发布资产：**SubprocessStarted 事件注册**（W7.1，KnownEventTypes 达 **11 种**）+ open release assets（`examples/` 两个 RHEO IR DSL 样例、`api/README.md` HTTP API reference、`docs/product.md` 产品介绍、`sdk/worker/README.md` worker SDK 指南，W7.4）。

## [v1.5-core] - 2026-08-07

Contracts catalog 增长：**CompensationExecuted 事件注册**（B8 / Wave 6，KnownEventTypes 10 种）。

## [v1.4-core] - 2026-08-07

Contracts catalog 增长：**StageAssigned 事件注册**（W2.2，引入 `KnownEventTypes` 目录，初始 8 种）+ **Migrated 事件注册**（W5.1，KnownEventTypes 9 种）；`contracts/types_test.go` 增加对应 catalog round-trip 断言。

## [v1.0-core] - 2026-08-07

首个 open release：**版本化 contracts**（`contracts/`，`schema_version=1`：Event/WorkflowDefinition/Stage/Transition/RunContext/Actor 等 + schema round-trip 测试）+ **Go worker SDK alpha**（`sdk/worker/`：`WorkStore` 端口 + heartbeat/lease renewal/fencing token/structured failure，W1.6）。
