> English · [中文](README.md)

# Examples — RHEOVELA Sample Workflows

This directory contains RHEOVELA sample workflows written in the **RHEO IR
(Intermediate Representation) DSL**, matching core's `internal/rheoir` parser
(Wave 3.3). The IR compiles to a `WorkflowDefinition` (`schema_version=1`), which can
then be imported and run via the core CLI.

> Note: `rheo workflow define` currently imports the `WorkflowDefinition` as **JSON**.
> `.rheo` files are author-facing IR source text, compiled to JSON by core's
> `internal/rheoir` (`Parse` → `ToWorkflowDefinition`). Each example below includes
> the compiled output and the run steps.

## File overview

| File | Content | Stages |
|------|---------|--------|
| `purchase-approval.rheo` | Purchase approval: invoice validation → cost-center approval → ERP posting; branches by amount, posting is compensable | validate → approve / post |
| `expense-approval.rheo` | Expense reimbursement: validation → manager approval → finance review → payment; over-limit goes through finance review, payment is compensable | validate → approve → financeCheck → pay |

Both examples demonstrate the three activity types (`AgentTask` / `HumanTask` /
`ServiceTask`), conditional branching (`when`) and compensation (`compensate`); the
`ServiceTask` declares an `effect_key` (effect ledger idempotency-key template).

## RHEO IR DSL syntax summary

The DSL consists of four kinds of lines — `process` declaration, `activity`,
`transition`, `compensate` (inline `//` comments are stripped by the parser).

```
process <Name> v<N> {
  activity <id> : <AgentTask|HumanTask|ServiceTask> [capability "<cap>"] [role "<role>"] [effect_key "<template>"] [timeout <dur>]
  transition <from> -> <to> [when <condition>]
  compensate <activity-id> with capability "<cap>"
  compensate <activity-id> with <activity-id>
}
```

### `process`

Declares the process name and version:

```
process PurchaseApproval v1 {
```

### `activity`

Declares an activity (mapped to `WorkflowDefinition.stages`):

| Attribute | Meaning |
|-----------|---------|
| `<AgentTask|HumanTask|ServiceTask>` | activity type: agent task / human task / service task |
| `capability "<cap>"` | capability required to execute this activity (`AssigneeSpec.required`) |
| `role "<role>"` | assigned role (`AssigneeSpec.role`) |
| `effect_key "<template>"` | external effect idempotency-key template (e.g. `invoice/${id}`), used by the effect ledger |
| `timeout <dur>` | timeout duration (`30m` / `2h` / `5d`, etc.) |

Example:

```
activity validate : AgentTask capability "invoice.validate"
activity approve  : HumanTask role "cost-center-owner"
activity post     : ServiceTask capability "erp.invoice.post" effect_key "invoice/${id}"
```

### `transition`

Declares flow between activities, with an optional `when` condition:

```
transition <from> -> <to> [when <var> <op> <literal>]
```

Conditions take the form `var op literal`; `op` supports `>=` `<=` `==` `!=` `>`
`<` (core `internal/kernel`'s `Evaluate`). Variable values are passed in via
`rheo run open --var k=v`.

### `compensate`

Declares compensation actions for already-succeeded activities
(`internal/compensation` builds the plan in reverse order):

```
compensate <activity-id> with capability "<cap>"   // compensate via a capability
compensate <activity-id> with <activity-id>        // compensate via another activity
```

## How to run

Prerequisite: install the core CLI
(`github.com/axisrobo/rheovela`, `cmd/rheo` → `rheo`/`rheo.exe`).

### 1. Compile IR → JSON

Compile `.rheo` to a `WorkflowDefinition` JSON (core's `internal/rheoir` provides
`Parse` + `ToWorkflowDefinition`). The JSON corresponding to `purchase-approval.rheo`:

```json
{
  "workflow_id": "PurchaseApproval",
  "title": "PurchaseApproval",
  "source": "rheo-ir",
  "stages": [
    {"id": "validate", "title": "validate", "assignee_spec": {"kind": "capability", "required": ["invoice.validate"]}, "inputs": [], "outputs": []},
    {"id": "approve", "title": "approve", "assignee_spec": {"kind": "role", "role": "cost-center-owner"}, "inputs": [], "outputs": []},
    {"id": "post", "title": "post", "assignee_spec": {"kind": "capability", "required": ["erp.invoice.post"]}, "inputs": [], "outputs": []}
  ],
  "transitions": [
    {"from": "validate", "to": "approve", "condition": "amount >= 1000"},
    {"from": "validate", "to": "post", "condition": "amount < 1000"}
  ]
}
```

The JSON corresponding to `expense-approval.rheo`:

```json
{
  "workflow_id": "ExpenseApproval",
  "title": "ExpenseApproval",
  "source": "rheo-ir",
  "stages": [
    {"id": "validate", "title": "validate", "assignee_spec": {"kind": "capability", "required": ["expense.validate"]}, "inputs": [], "outputs": []},
    {"id": "approve", "title": "approve", "assignee_spec": {"kind": "role", "role": "manager"}, "inputs": [], "outputs": []},
    {"id": "financeCheck", "title": "financeCheck", "assignee_spec": {"kind": "role", "role": "finance-team"}, "inputs": [], "outputs": []},
    {"id": "pay", "title": "pay", "assignee_spec": {"kind": "capability", "required": ["payroll.pay"]}, "inputs": [], "outputs": []}
  ],
  "transitions": [
    {"from": "validate", "to": "approve"},
    {"from": "approve", "to": "financeCheck", "condition": "amount >= 1000"},
    {"from": "financeCheck", "to": "pay"}
  ]
}
```

> Note: the `effect_key` and `compensate` in the IR are currently rich IR-layer
> semantics that have not yet landed in the `WorkflowDefinition` JSON schema
> (`contracts.Stage` / `contracts.Transition` have no corresponding fields yet); they
> are consumed by core's effect ledger / compensation planner.

> All example `WorkflowDefinition` JSONs conform to `api/workflow.schema.json`
> (draft-07), and can be validated with any JSON Schema validator or with
> `TestWorkflowSchemaValidatesSample` from `go test ./api/ -v`.

### 2. Define and run

Using `PurchaseApproval` as an example (works in both Windows PowerShell and
Unix-like shells):

```sh
# Create and select a project
rheo project new purchase-approval
rheo project list
rheo project use <id>

# Import the compiled definition
rheo workflow define --file purchase-approval.json

# Open a run (amount ≥ 1000 → takes the approval branch)
rheo run open PurchaseApproval --var amount=1500 --as alice

# Advance step by step
rheo step enter validate --as alice
rheo step exit  validate --as alice
rheo step enter approve  --as alice
rheo step exit  approve  --as alice
rheo step enter post     --as alice
rheo step exit  post     --as alice

# View state (TUI / HTML / JSON)
rheo view --format tui

# Close the run
rheo run close --outcome done
```

With `amount=500` (`< 1000`), after `validate` completes the run goes straight to the
`post` branch:

```sh
rheo run open PurchaseApproval --var amount=500 --as alice
rheo step enter validate --as alice
rheo step exit  validate --as alice
rheo step enter post     --as alice
rheo step exit  post     --as alice
rheo run close --outcome done
```

### 3. Common CLI commands at a glance

| Command | Purpose |
|---------|---------|
| `rheo workflow define --file <json>` | import a workflow definition |
| `rheo workflow list` / `rheo workflow show <id>` | inspect definitions |
| `rheo run open <workflow-id> [--var k=v]... [--as <actor>]` | open a run |
| `rheo step enter\|exit\|fail\|skip <stage-id> [--as <actor>]` | advance/fail/skip a stage |
| `rheo view [--format tui\|html\|json]` | view run state |
| `rheo run close --outcome done\|fail` | close a run |
| `rheo history --run <id>` | view the event log |
| `rheo serve` | start the HTTP Ops API |

Detailed CLI docs live in the core repo `docs/api.md`.

## Go Worker example (end-to-end SDK)

`go-worker/` is a runnable Go worker example demonstrating the worker SDK's
(`sdk/worker`) full processing loop: poll → claim → run → complete/fail. It uses an
in-memory `fakeStore` (implementing `worker.WorkStore`) and does not depend on core.

### Run

```sh
go run ./examples/go-worker/
```

Expected output:

```
processed=2
  task-1: done
  task-2: failed
```

`task-1`'s work function returns `success` → `done`; `task-2` returns `failure` →
`failed`.

### Tests

```sh
go test ./examples/go-worker/ -v   # TestExampleWorkerFlow
```

### Connecting to real core

The `fakeStore` in the example is a minimal implementation of the
`sdk/worker.WorkStore` port. In production this port is provided by the rheovela core
(AGPL kernel) `runtime` package: `runtime.WorkerBridge` implements `WorkStore`, mapping
`PollReady` / `Claim` / `Heartbeat` / `Complete` / `Fail` to core's Worker HTTP API
(`rheo serve`). To integrate, simply replace the first argument of `worker.New` from
`fakeStore` to a `runtime.WorkerBridge` instance; the rest of the processing logic
stays the same.

## Python Worker example (end-to-end SDK)

`python-worker/main.py` is a runnable Python worker example demonstrating the full
processing loop of `sdk/python/worker.py`: poll → claim → complete/fail. It starts a
local fake Worker-API server (stdlib `http.server`, in-memory dict storage) which
shuts down automatically when the process exits. Only the Python standard library is
used.

The example reuses `sdk/python/worker.py`'s `Client` / `Worker` (imported via
`sys.path`); `task-1` returns `success` → `done`; `task-2` returns `failure` →
`failed`.

### Run

```sh
python examples/python-worker/main.py
```

Expected output:

```
processed=2
  task-1: done
  task-2: failed
```

### Connecting to real core

The `FakeWorkerServer` in the example implements the Worker HTTP API
(`GET /api/v1/work-items`, `POST .../claim` / `complete` / `fail`). In production this
API is provided by core's `rheo serve`; replace `Client(srv.base_url)` with
`Client("http://localhost:8080")`.

## TypeScript Worker example (end-to-end SDK)

`typescript-worker/worker-example.mjs` is a runnable ESM script (Node 18+)
demonstrating `Worker.processOnce()`'s full processing loop: poll → claim →
complete/fail. It mocks `globalThis.fetch` to point at an in-memory Worker-API
response and needs no runtime dependencies.

> Node cannot `import` `.ts` files directly (they require transformation), so the
> script inlines a minimal `Client` / `Worker` implementation matching
> `sdk/typescript/worker.ts`, noted as mirrored from the SDK.

`task-1` returns `success` → `done`; `task-2` returns `failure` → `failed`.

### Run

```sh
node examples/typescript-worker/worker-example.mjs
```

Expected output:

```
processed=2
  task-1: done
  task-2: failed
```

### Connecting to real core

The mock `fetch` in the example returns Worker HTTP API responses. In production,
replace `new Client("http://fake")` with `new Client("http://localhost:8080")` and
remove the `globalThis.fetch` mock to talk to core's `rheo serve`.
