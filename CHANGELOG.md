# Changelog

RHEOVELA open锛坄github.com/axisvela/rheovela-open`锛孉pache-2.0锛夊彂甯冭褰曪細瀵瑰鍙戝竷灞傦紙鐗堟湰鍖?contracts銆丼DK銆乪xamples銆丆I銆丱penAPI锛夈€傛牸寮忥細鏈€鏂板湪鍓嶏紱姣忎釜鐗堟湰瀵瑰簲涓€涓?git tag锛坈ore/open 鍏变韩璇箟鐗堟湰 tag锛孍E 鍙嫭绔嬶級銆?

## [Unreleased]

### Added

- `contracts.FederationEnvelope` / `FederationHandoff` / `FederationSLA`: cross-org handoff wire contract (sender/receiver tenant, correlation, shared SLA context).

### Added

- `designer/collab.mjs`: EE collaboration transport adapter for the Designer
  CRDT (`sendOps`, `getDocument`, `heartbeat`, `presence`).

## [v1.68.0] - 2026-11-08

Coordinated release with core `v1.68.0`; no open contract or SDK changes.
Core adds the position-axis `RunAuthority` command guard.

## [v1.67.0] - 2026-11-08

Coordinated release with core `v1.67.0`; no open contract or SDK changes.
Core adds the SLA deadline calculator and SLA policy override hooks.

## [v1.66.0] - 2026-11-08

Coordinated release with core `v1.66.0`; no open contract or SDK changes.
Core adds the generic feed tenant resolver used by EE tenant-scoped projects.

## [v1.65.0] - 2026-11-08

### Added

- `scripts/release-local.ps1` / `scripts/release-local.sh`: build the core
  `rheo` release matrix from sibling checkouts without `CORE_REPO_TOKEN`,
  with optional Docker image and `gh release upload`.
- `designer/crdt.mjs`: dependency-free op-based CRDT document client for the
  Community Process Designer (LWW registers with actor tie-break, stage/
  transition/compensation registers, tombstones, snapshot merge, and a
  transport queue with replay until ack). Design drafts stay separate from
  published definitions; the EE collaboration service owns auth and persistence.

### Fixed

- The release workflow builds core from `Rheovela/backend` (the Go module root)
  instead of the repository root.

## [v1.64.0] - 2026-11-08

Coordinated release with core `v1.64.0`; no open contract or SDK changes.
Core adds the ORCHADYN auto-replan closure and BPMN callActivity subprocess
support (import 鈫?`Stage.SubWorkflow`, token-engine async spawn/link/鍥炲啓).

## [v1.63.0] - 2026-11-07

### Added

- The release workflow publishes the `rheo` server image to
  `ghcr.io/axisvela/rheo` (`:<version>` and `:latest`), built from the core
  `release/Dockerfile` at the same tag.

## [v1.62.0] - 2026-11-07

### Changed

- Coordinated release with core `v1.62.0`. Open has no code changes since
  `v1.60.0`; the `Stage.SubWorkflow` contract remains. Module tag `v3.15.0`
  unchanged. EE remains on its independent tag.

## [v1.61.0] - 2026-11-06

### Changed

- Release marker after `v1.60.0`; no open code changes in this version. Core/open
  release tag coordination continues; EE remains on its independent tag.

## [v1.60.0] - 2026-11-06

### Added

- `Stage.SubWorkflow` / `SubWorkflowRef` contract: optional declaration that a
  stage is executed by a child workflow, with `workflow_id`, `version`, and
  `${parentVar}` input mapping.
- Contract round-trip test for the new field.

### Changed

- Shared release tag with core `v1.60.0`; EE uses its own tag. Module tag
  `v3.15.0` unchanged.

## [v1.59.0] - 2026-11-06

### Changed

- Coordinated release with core `v1.59.0` and EE `v1.59.0`. Open has no code
  changes; the KINETOVELA backend now serves `/edges*` and the EE live drill
  passes. Module tag `v3.15.0` unchanged.

## [v1.58.0] - 2026-11-05

### Changed

- Coordinated release with core `v1.58.0` and EE `v1.58.0`. Open has no code
  changes; EE moves work-item adaptation to COORDIVELA (lease/fencing) and
  records live-peer contract findings.

## [v1.57.0] - 2026-11-05

### Changed

- Coordinated release with core `v1.57.0` and EE `v1.57.0`. Open has no code
  changes; EE adds audit CSV/JSONL export, migration approval detail/filter/
  batch/export UX, and SYMBIVELA/KINETOVELA peer drills.

## [v1.56.0] - 2026-11-05

### Changed

- Coordinated release with core `v1.56.0` and EE `v1.56.0`. Open has no code
  changes; EE adds migration console deep UI, local HA/edge drills, and
  Vault/K8s secret refs.

## [v1.55.0] - 2026-11-05

### Changed

- Coordinated release with core `v1.55.0` and EE `v1.55.0`. Open has no code
  changes; EE adds Keycloak admin-token refresh, SLO health, device-flow
  login, and audit explorer filters/export.

## [v1.54.0] - 2026-11-05

### Changed

- Coordinated release with core `v1.54.0` and EE `v1.54.0`. Open has no code
  changes; core adds unknown-effect reconciliation and engine/participant
  contract test hardening.

## [v1.53.0] - 2026-11-05

### Changed

- Coordinated release with core `v1.53.0` and EE `v1.53.0`. Open has no code
  changes in this release; core adds admin HTTP, configurable metrics windows,
  scheduler misfire/calendar, ORCHADYN trigger sources, HA leader, migration
  crash tests, and backup sink snapshots.

## [v1.52.0] - 2026-11-02

### Changed

- Coordinated M0 release: core replaces OIDC device flow/JWKS with local HTTP
  Basic auth and EE adds EIDOVELA/AEGIVELA enterprise authentication. Open has
  no runtime authentication changes; this release updates the roadmap reference
  to the M0鈥揗7 execution plan.

## [v1.51.0] - 2026-11-01

### Added

- OpenAPI paths and schemas for stage leases and worker sessions, plus optional
  `lease_token` / `session_token` on step and work-item claim bodies.
- Go SDK `ClaimStageLease` / `HeartbeatStageLease` / `ReleaseStageLease` and
  worker-session equivalents, plus lease-token claim options and step fields.
- Python, TypeScript, and Rust worker clients expose the same lease/session
  methods and claim-token options.
- Java SDK lease/session methods are implemented and compile with JDK 25.

## [v1.50.0] - 2026-11-01

### Changed

- Coordinated ecosystem release with core `v1.50.0` and EE `v1.50.0`. No open
  changes; core adds stage-level leases and worker sessions.

## [v1.49.1] - 2026-11-01

### Changed

- Coordinated ecosystem release with core `v1.49.1` and EE `v1.49.1`. No open
  changes; core ships a CI workflow fix.

## [v1.49.0] - 2026-11-01

### Changed

- Coordinated ecosystem release with core `v1.49.0` and EE `v1.49.0`. No open
  contract or SDK changes; core adds Valkey stream replay for events and feeds,
  feed route retention/archival, and PostgreSQL commit-ordered feed cursors.

## [v1.48.0] - 2026-10-31

### Changed

- Coordinated ecosystem release with core `v1.48.0` and EE `v1.48.0`. No open
  contract or SDK changes in this release; core adds tenant-aware feed routing
  and EE adds the Cockpit Live Activity feed.

## [v1.47.0] - 2026-10-31

### Changed

- Coordinated ecosystem release with core `v1.47.0` and EE `v1.47.0`. No open
  contract or SDK changes in this release; core adds durable feed routes and
  `/api/v1/feeds/*` replay endpoints.

## [v1.46.0] - 2026-10-31

### Changed

- Coordinated ecosystem release with core `v1.46.0` and EE `v1.46.0`. No open
  contract or SDK changes in this release; core adds feed routing and
  SSE/WebSocket actor/role/capability filters.

## [v1.45.0] - 2026-10-31

### Changed

- Coordinated ecosystem release with core `v1.45.0` and EE `v1.45.0`. No open
  contract or SDK changes in this release; core adds the configurable event bus
  (`local`/Valkey) and post-commit publishing.

## [v1.44.0] - 2026-10-31

### Added

- RHEO IR supports `approval_roles`, `tools`, `credential`, `completion`,
  `approvals`, and `completions` activity attributes with a conformance fixture.

### Fixed

- SDK workers pass the claim actor on completion, so role-targeted work items
  satisfy completion-time authorization.

- `/api/v1/metrics` exposes notification counts and its field names now match
  the response (`timers_pending`, `work_items_ready`, `effects_unknown`).

- `/api/v1/status` exposes pending/dead-lettered notification counts.

- Notifications expose retry/dead-letter metadata and a requeue endpoint.

- `api/workflow.schema.json` now includes the participant/completion Stage
  fields (`task_kind`, `payload`, `required_evidence`, `approval_roles`,
  `tool_scope`, `credential_ref`, `completion_mode`, counts) and assignment
  fields (`candidates`, `sla`, `budget`, `claim_policy`, `escalation`), and
  fixes the malformed stage `delegation` fragment.

### Changed

- Coordinated ecosystem release with core `v1.44.0` and EE `v1.44.0`.

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

### Added

- RHEO IR supports `approval_roles`, `tools`, `credential`, `completion`,
  `approvals`, and `completions` activity attributes with a conformance fixture.

### Fixed

- SDK workers pass the claim actor on completion, so role-targeted work items
  satisfy completion-time authorization.

- `/api/v1/metrics` exposes notification counts and its field names now match
  the response (`timers_pending`, `work_items_ready`, `effects_unknown`).

- `/api/v1/status` exposes pending/dead-lettered notification counts.

- Notifications expose retry/dead-letter metadata and a requeue endpoint.

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

闅?core `v1.1.0`锛圥ostgres 鏀寔锛夊悓姝ュ彂甯冿細鏂囨。琛ラ綈**杩愮淮鎵嬪唽**
锛坄docs/OPERATIONS.md`锛屽惈 Postgres 鍚庣灏忚妭锛変笌**寮€鍙戞墜鍐?*
锛坄docs/DEVELOPMENT.md`锛夛紱README 鏀逛负鑻辨枃涓轰富 + 涓枃閾炬帴
锛坄README.zh.md`锛夈€?

## [v0.9.0-beta] - 2026-08-08

棣栦釜 **beta** 鍙戝竷闈紙tag 涓?v1.14-core 鍚屾寚鍚戯級锛歋DK 瑕嗙洊 **5 绉嶈瑷€**
锛圙o / Python / TypeScript / Java / Rust锛夈€傛柊澧?**Java worker SDK**
锛坄sdk/java/`锛欽DK 21銆侀浂渚濊禆 `java.net.http.HttpClient`锛?
`Worker.poll/claim/complete/fail/processOnce`锛変笌 **Rust worker SDK**
锛坄sdk/rust/`锛歴td-only銆侀浂澶栭儴 crate銆乣std::net::TcpStream` 鏋佺畝 HTTP/1.1
client锛夛紝骞舵柊澧?**java / rust CI jobs**鈥斺€擿.github/workflows/ci.yml` 鐜版湁
4 涓?job锛坓o / python / java / rust锛夈€?

## [v1.14-core] - 2026-08-08

**Rust worker SDK**锛圵24.5锛宍sdk/rust/`锛歚Client` / `Worker` / `process_once`锛?
绂荤嚎鍙瀯寤猴級+ **Rust CI job**锛坄cargo build --offline` + `cargo run --offline`锛夈€?

## [v1.13-core] - 2026-08-08

**Worker SDK 鍚庡彴蹇冭烦**锛圵23.2锛夛細`sdk/worker.Worker` 鍦ㄩ暱浠诲姟鎵ц鏈熼棿浜庡悗鍙?
缁锛坄worker_test.go` 澧炲姞 heartbeat 鐢ㄤ緥锛夛紝閬垮厤闀夸换鍔＄绾﹁繃鏈熻閲嶆柊棰嗗彇銆?

## [v1.12-core] - 2026-08-08

**Event JSON Schema**锛圵21.1锛宍api/event.schema.json`锛宒raft-07锛岃鐩?**13 绉嶄簨浠?
绫诲瀷**锛? `api/schema_test.go` 涓€鑷存€ф祴璇曪紙`TestEventSchema`锛夈€?

## [v1.11-core] - 2026-08-08

**WorkflowDefinition JSON Schema**锛圵20.1锛宍api/workflow.schema.json`锛宒raft-07锛?
conformance 娴嬭瘯锛坄TestWorkflowSchemaValidatesSample` 绛夛級锛沗examples/README.md`
澹版槑绀轰緥 JSON 鍧囩鍚堣 schema銆?

## [v1.10-core] - 2026-08-08

Runnable examples 琛ラ綈锛氭柊澧?**Python worker example**锛坄examples/python-worker/main.py`锛変笌 **TypeScript worker example**锛坄examples/typescript-worker/worker-example.mjs`锛夛紝涓?Go example锛坴1.9-core锛夋瀯鎴愪笁璇█鍙繍琛屾牱渚嬶紙`examples/README.md`锛夈€?

## [v1.9-core] - 2026-08-08

Runnable **Go worker example**锛坄examples/go-worker/main.go` + `main_test.go`锛宨n-memory store锛夛紝绀轰緥娴佺▼鍙洿鎺ヨ繍琛屽苟绾冲叆 `go test`锛? 鍖?ok锛歝ontracts / sdk/worker / examples/go-worker锛夈€?

## [v1.8-core] - 2026-08-07

OpenAI 鍙戝竷闈㈡墿灞曪細**TypeScript worker SDK**锛坄sdk/typescript/worker.ts` + `worker.test.ts`锛岄浂渚濊禆銆丯ode 18+ `fetch`锛夈€?*OpenAPI 3.0 瑙勮寖**锛坄api/openapi.yaml`锛?00 琛岋級銆?*鍙岃 README**锛堜腑鏂?+ English 蹇€熷紑濮嬶級銆?

## [v1.7-core] - 2026-08-07

**Python worker SDK**锛坄sdk/python/worker.py`锛歴tdlib-only `Client`/`Worker`锛? **Python CI job**锛坄.github/workflows/ci.yml`锛宍unittest` 璺?`sdk/python` 娴嬭瘯锛夈€?

## [v1.6-core] - 2026-08-07

Contracts catalog 琛ラ綈 + 鍙戝竷璧勪骇锛?*SubprocessStarted 浜嬩欢娉ㄥ唽**锛圵7.1锛孠nownEventTypes 杈?**11 绉?*锛? open release assets锛坄examples/` 涓や釜 RHEO IR DSL 鏍蜂緥銆乣api/README.md` HTTP API reference銆乣docs/product.md` 浜у搧浠嬬粛銆乣sdk/worker/README.md` worker SDK 鎸囧崡锛學7.4锛夈€?

## [v1.5-core] - 2026-08-07

Contracts catalog 澧為暱锛?*CompensationExecuted 浜嬩欢娉ㄥ唽**锛圔8 / Wave 6锛孠nownEventTypes 10 绉嶏級銆?

## [v1.4-core] - 2026-08-07

Contracts catalog 澧為暱锛?*StageAssigned 浜嬩欢娉ㄥ唽**锛圵2.2锛屽紩鍏?`KnownEventTypes` 鐩綍锛屽垵濮?8 绉嶏級+ **Migrated 浜嬩欢娉ㄥ唽**锛圵5.1锛孠nownEventTypes 9 绉嶏級锛沗contracts/types_test.go` 澧炲姞瀵瑰簲 catalog round-trip 鏂█銆?

## [v1.0-core] - 2026-08-07

棣栦釜 open release锛?*鐗堟湰鍖?contracts**锛坄contracts/`锛宍schema_version=1`锛欵vent/WorkflowDefinition/Stage/Transition/RunContext/Actor 绛?+ schema round-trip 娴嬭瘯锛? **Go worker SDK alpha**锛坄sdk/worker/`锛歚WorkStore` 绔彛 + heartbeat/lease renewal/fencing token/structured failure锛學1.6锛夈€?

