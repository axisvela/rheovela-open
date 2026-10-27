# RHEO IR v1.0 Specification

Status: published. This document is the normative specification for the RHEO IR
text format accepted by Rheovela core (`internal/rheoir`). The compiled target is
`WorkflowDefinition` (`schema_version=1`).

The key words MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY are to be interpreted
as described in RFC 2119.

## 1. Scope

RHEO IR is an author-facing, line-oriented DSL for declaring a process,
activities, transitions and compensation actions. It is not BPMN: BPMN is an
import/export projection, while RHEO IR is the native source format.

This specification covers the v1.0 grammar implemented in core. Future
extensions are reserved in section 10 and MUST NOT be inferred from this
document.

## 2. Conformance

An implementation conforms to this specification if it:

1. accepts every valid program in section 4;
2. rejects malformed programs with a stable `RHEOIRxxx` code and the offending
   1-based line (section 7);
3. compiles a parsed process to `WorkflowDefinition` as defined in section 6;
4. preserves compensation declarations for the compensation planner.

A public corpus of valid and invalid programs with expected output lives in
[`conformance/rheo-ir/v1.0/`](../conformance/rheo-ir/v1.0/manifest.json).

## 3. Lexical Rules

- The source is UTF-8 text and MUST NOT start with a byte-order mark (BOM).
- Lines are separated by `\n` (a trailing `\r` is ignored as whitespace).
- Blank lines are ignored.
- `//` starts a line comment. A `//` inside a single- or double-quoted value is
  not a comment. There are no block comments.
- Whitespace separates tokens. Indentation is insignificant.
- `ident` is any non-whitespace token that does not contain `{`, `}`, `:`, `->`
  or `//`. Implementations SHOULD use `[A-Za-z_][A-Za-z0-9_.-]*`.
- A `value` is a single whitespace-delimited token, optionally wrapped in double
  or single quotes. The v1.0 parser removes surrounding double quotes only;
  single quotes are preserved literally. Quoted values MUST NOT contain
  whitespace, and backslash escape sequences are not interpreted.
- A `duration` is a decimal number followed by one of `d`, `h`, `m`, `s`, `ms`,
  `us`, `ns`. Fractional values are allowed (for example `1.5h`).

## 4. Grammar

```abnf
program       = process-decl *( WS declaration ) WS
process-decl  = "process" WS ident [ WS version ] WS "{" [ WS "}" ]
version       = "v" 1*DIGIT
declaration   = activity / transition / compensate / delegate / evidence / migration
activity      = "activity" WS ident WS ":" WS kind *( WS attribute )
kind          = ident
attribute     = "capability" WS value
              / "role" WS value
              / "effect_key" WS value
              / "timeout" WS duration
transition    = "transition" WS ident WS "->" WS ident [ WS "when" WS condition ]
compensate    = "compensate" WS ident WS "with" WS ( "capability" WS value / ident )
delegate      = "delegate" WS ident WS "to" WS ("actor"/"role"/"capability") WS value
                *( WS ("scopes" WS value / "budget" WS integer / "evidence" WS value) )
evidence      = "evidence" WS ident WS "require" WS value
migration     = "migration" WS ident WS "policy" WS value
condition     = 1*( any non-newline character )
```

Declarations are line-oriented and MUST NOT share a line with the process block
or with another declaration. `process P { }` is syntactically recognized but is
always rejected because the block contains no activities. Declarations after
the closing `}` are invalid.

### 4.1 Examples

```rheo
process PurchaseApproval v1 {
  activity validate : AgentTask capability "invoice.validate" timeout 30m
  activity approve  : HumanTask role "cost-center-owner"
  activity post     : ServiceTask capability "erp.invoice.post" effect_key "invoice/${id}"

  transition validate -> approve when amount >= 1000
  transition validate -> post when amount < 1000

  compensate post with capability "erp.invoice.reverse"
  delegate approve to role "cost-center-lead" scopes "approve,reject" budget 500
  evidence approve require "approval_receipt"
}
```

## 5. Semantics

### 5.1 Process

`process <name> [v<version>] { ... }` declares exactly one process.

- `<name>` becomes `WorkflowDefinition.workflow_id` and `.title`.
- `v<version>` is parsed into the IR AST (`Process.Version`). It is not
  represented in `WorkflowDefinition`; definition versioning is assigned by
  `rheo workflow define` / the store.
- The compiled definition has `source = "rheo-ir"`.

### 5.2 Activity

`activity <id> : <kind> [attributes]` declares one stage.

- `<id>` MUST be unique within the process. Duplicate ids are rejected by
  `rheo workflow validate` after compilation.
- `<kind>` is a free-form token. The canonical kinds are `AgentTask`,
  `HumanTask` and `ServiceTask`; other kinds are preserved in the IR AST but
  compile to an open assignee.
- `capability "<cap>"` sets `AssigneeSpec.kind = "capability"` and
  `AssigneeSpec.required = ["<cap>"]`.
- `role "<role>"` sets `AssigneeSpec.kind = "role"` and `AssigneeSpec.role`.
- When both `capability` and `role` are present, `role` takes precedence.
- When neither is present, `AssigneeSpec.kind = "open"`.
- `effect_key "<template>"` and `timeout <duration>` compile to
  `Stage.effect_key` and `Stage.timeout`.
- `inputs` and `outputs` are emitted as empty arrays by the compiler.

### 5.3 Transition

`transition <from> -> <to> [when <condition>]` declares a directed edge.

- `<from>` and `<to>` are activity ids. Referential integrity is checked by
  `rheo workflow validate`; the parser does not require the targets to be
  declared earlier or at all.
- `<condition>` is the remainder of the line after `when`, with surrounding
  whitespace trimmed. It is copied verbatim to `Transition.condition`.
- Conditions are evaluated by core `internal/kernel` and support `>=`, `<=`,
  `==`, `!=`, `>`, `<` against run variables supplied with `rheo run open --var`.
- A transition without `when` is unconditional.

### 5.4 Compensation

`compensate <activity-id> with ...` declares how to compensate an activity that
has already succeeded.

- `with capability "<cap>"` records a capability-based compensation.
- `with <activity-id>` records an activity-based compensation.
- Compensation declarations compile to `WorkflowDefinition.compensations`. The
  compensation planner (`internal/compensation`) consumes them and executes in
  reverse completion order.
- Compensating an activity that never succeeded is a no-op; compensation MUST
  NOT fabricate a return to the original external state.

### 5.5 Delegation and evidence

`delegate <activity> to actor|role|capability "<value>" [scopes "<csv>"] [budget <int>] [evidence "<duty>"]`
attaches an authority delegation to a stage.

- `scopes` is a comma-separated list copied to `Delegation.scopes`.
- `budget` is a non-negative integer copied to `Delegation.budget`.
- `evidence` may be repeated and copies duties to `Delegation.evidence`.
- `evidence <activity> require "<duty>"` appends standalone evidence duties to
  `Stage.evidence`.
- Delegation/evidence are metadata for authorization and audit adapters; the
  runtime does not grant scopes or collect evidence automatically.

### 5.6 Migration policy

`migration <activity> policy "preserve|ignore|compensate"` annotates how a stage
may be removed by a live migration.

- `preserve` always blocks removal.
- `ignore` allows removal without a compensation requirement.
- `compensate` uses the migration compensation mode (`manual`/`auto`) and
  requires a declared compensation for `auto`.
- An absent policy uses the migration-wide compensation mode.

## 6. Compilation to WorkflowDefinition

| IR element | WorkflowDefinition field |
|------------|--------------------------|
| `process <name>` | `workflow_id`, `title` |
| `activity <id> : <kind>` | `stages[].id`, `stages[].title` (title = id) |
| `capability "<cap>"` | `stages[].assignee_spec = {kind:"capability", required:["<cap>"]}` |
| `role "<role>"` | `stages[].assignee_spec = {kind:"role", role:"<role>"}` |
| no capability/role | `stages[].assignee_spec = {kind:"open"}` |
| `transition a -> b` | `transitions[] = {from:"a", to:"b"}` |
| `when <condition>` | `transitions[].condition` |
| `effect_key` / `timeout` | `stages[].effect_key`, `stages[].timeout` |
| `compensate ...` | `compensations[] = {for_activity, compensate_with/capability}` |
| `delegate ...` | `stages[].delegation = {kind, delegatee, scopes, budget, evidence}` |
| `evidence ... require` | `stages[].evidence[]` |
| `migration ... policy` | `stages[].migration_policy` |

The compiler MUST emit `inputs: []` and `outputs: []` for every stage.

Compiled JSON example:

```json
{
  "workflow_id": "PurchaseApproval",
  "title": "PurchaseApproval",
  "source": "rheo-ir",
  "stages": [
    {"id": "validate", "title": "validate", "assignee_spec": {"kind": "capability", "required": ["invoice.validate"]}, "inputs": [], "outputs": []},
    {"id": "approve", "title": "approve", "assignee_spec": {"kind": "role", "role": "cost-center-owner"}, "inputs": [], "outputs": []}
  ],
  "transitions": [
    {"from": "validate", "to": "approve", "condition": "amount >= 1000"}
  ]
}
```

## 7. Validation and Diagnostics

The parser returns `ErrParse`-wrapped `*ParseError` values. Every error carries a
stable code and the 1-based source line:

```
rheo ir parse error [RHEOIR007] line 3: malformed activity line: "activity"
```

| Code | Condition |
|------|-----------|
| `RHEOIR001` | missing `process` declaration |
| `RHEOIR002` | first non-comment line is not a process declaration |
| `RHEOIR003` | malformed process declaration (missing `{` or trailing tokens) |
| `RHEOIR004` | unterminated process block |
| `RHEOIR005` | process has no activities |
| `RHEOIR006` | unknown declaration keyword inside the block |
| `RHEOIR007` | malformed activity (`activity <id> : <kind>` required) |
| `RHEOIR008` | missing value for an activity attribute |
| `RHEOIR009` | unknown activity attribute |
| `RHEOIR010` | malformed timeout value or unit |
| `RHEOIR011` | malformed transition (missing `->`, endpoint, or empty `when`) |
| `RHEOIR012` | malformed compensation |
| `RHEOIR013` | content or a stray `}` after the process block |
| `RHEOIR014` | malformed process version (only `v` + decimal digits) |
| `RHEOIR015` | unknown delegation kind or malformed delegate declaration |
| `RHEOIR016` | missing delegation delegatee |
| `RHEOIR017` | malformed delegation budget |
| `RHEOIR018` | unknown delegation attribute or missing value |
| `RHEOIR019` | malformed evidence declaration |
| `RHEOIR020` | malformed migration declaration |
| `RHEOIR021` | unknown migration policy |
| `RHEOIR022` | `delegate`/`evidence`/`migration` references an unknown activity |

After compilation, `rheo workflow validate <file>` performs definition-level
checks: non-empty stages, unique ids, transition endpoints exist, the transition
graph is acyclic, and conditions use the supported operator set.

### 7.1 Syntax boundaries in v1.0

The following limits are intentional and MUST NOT be inferred as supported
syntax:

- A quoted value MUST NOT contain whitespace; `capability "invoice validate"` is
  not a single value.
- Single-quoted values are preserved including the quotes.
- Backslash escapes are not interpreted.
- A UTF-8 BOM before `process` is rejected.
- A `duration` uses exactly one unit suffix; `1h30m` is not supported.
- `kind` is a single free-form token.
- Declarations MUST be on separate lines.
- Repeated `delegate` and `migration` declarations for one activity keep the last
  value; repeated `evidence` declarations append.
- Duplicate activity ids, transition endpoints, cycles and condition operators
  are diagnosed by `rheo workflow validate`, not by `Parse`.
- Conditions are copied verbatim; the parser does not evaluate them.

## 8. Execution Model

1. Compile with `rheo ir tojson <file> [--output <json>]`.
2. Validate with `rheo workflow validate <json>`.
3. Import with `rheo workflow define --file <json>`.
4. Open a run with `rheo run open <workflow> --as <actor> [--var k=v]`.
5. Execute stages with `rheo step <run-id> ...` or the worker API.

`rheo ir parse <file>` prints the IR summary (activities, transitions,
compensations) without compiling.

## 9. BPMN Projection

RHEO IR and BPMN are separate formats:

- BPMN import maps `task`/`userTask`/`serviceTask`/gateways/events/`subProcess`
  to stages and `sequenceFlow` to transitions.
- Embedded `subProcess` elements become a single stage; internal nodes and
  sequence flows are preserved in `WorkflowDefinition.bpmn`.
- `collaboration` participants (`processRef`), `messageFlow`, lanes and data
  objects are preserved in `WorkflowDefinition.bpmn` and are not executable
  transitions. Flow nodes record their owning `process_id`.
- `choreography`/`choreographyTask`/`subChoreography`/`callChoreography`,
  `participantRef`, `messageFlowRef` and `loopType` are preserved in
  `bpmn.choreographies`; choreography is not executable.
- `dataInputAssociation`/`dataOutputAssociation`, `sourceRef`/`targetRef`,
  `assignment` and `transformation` are preserved in `bpmn.data_associations`.
- `callActivity` (`calledElement`), `transaction`, `eventSubProcess`
  (`triggeredByEvent`), `adHocSubProcess`, send/receive/manual tasks,
  `isForCompensation`, and standard/multi-instance loop characteristics are
  preserved as stage metadata.
- BPMN DI preserves the plane `bpmnElement`/resolution, shape and edge
  attributes, label bounds, and `BPMNLabelStyle` font metadata.
- BPMN data stores, data-store references, properties, item definitions,
  data-object item/collection metadata, process-level `dataInput`/`dataOutput`,
  root event definitions, `eventDefinitionRef`, text annotations, associations
  and groups are preserved as metadata. Ad-hoc ordering/cancel attributes,
  transaction method and activity quantity attributes are also retained.
- BPMN resources and resource roles (`performer`/`humanPerformer`/
  `potentialOwner`/`resourceRole`), categories/category values, and
  `callChoreography` `calledChoreographyRef`/`participantAssociation` are
  preserved. Resource parameter bindings and assignment expressions,
  participant `interfaceRef`/`endPointRef`, data-object `dataState`, and
  process/activity `ioSpecification`/`inputSet`/`outputSet` round-trip as
  metadata. Root `message`/`signal`/`error`/`escalation`/
  `interface`/`operation`/`endPoint` declarations are also preserved.
- BPMN collaboration conversations (`conversation`, `subConversation`,
  `callConversation`, `conversationLink`), complex-gateway activation
  conditions, multi-instance behavior references, and ad-hoc completion
  conditions round-trip as metadata.
- BPMN export uses `WorkflowDefinition.bpmn` metadata to emit `subProcess`,
  boundary/intermediate events, gateway types, participants, message flows,
  `laneSet`/`lane`, `dataObject`/`dataObjectReference` and BPMN DI
  (`BPMNDiagram`/`BPMNShape`/`BPMNEdge`/waypoints). Without metadata it falls
  back to the flat `startEvent`/`task`/`endEvent` subset.

## 10. Versioning and Reserved Extensions

This document specifies RHEO IR v1.0. A future grammar version MUST use a new
document version and a new process-level marker or parser entry point; v1.0
parsers MUST reject unknown declaration keywords rather than silently ignoring
them.

The following extensions are reserved and are not part of v1.0:

- replan annotations beyond the stage migration policy;
- explicit effect-ledger actions beyond `effect_key`;
- subprocess/materialization declarations.

Until specified, authors MUST express these concerns through the compiled
`WorkflowDefinition`, runtime commands, or BPMN import metadata.

## 11. Conformance Checklist

The normative program corpus is
[`conformance/rheo-ir/v1.0/`](../conformance/rheo-ir/v1.0/manifest.json); see
[`conformance/README.md`](../conformance/README.md) for how to run it.

- [ ] `process` is the first non-comment line.
- [ ] The process block is closed and contains at least one activity.
- [ ] Every invalid program fails with the documented `RHEOIRxxx` code and line.
- [ ] Valid programs compile to the expected `WorkflowDefinition`.
- [ ] Activity ids are unique and referenced by existing transitions.
- [ ] Conditions use supported operators and valid operands.
- [ ] Timeout values use a supported unit.
- [ ] Compensation targets exist or are intentional capability names.
- [ ] The compiled JSON passes `rheo workflow validate`.

## 12. References

- Examples: [`examples/README.en.md`](../examples/README.en.md)
- CLI: [`docs/api.md`](api.md)
- Workflow schema: [`api/workflow.schema.json`](../api/workflow.schema.json)
- BPMN import/export: core `internal/bpmn` and core `docs/ROADMAP.md`
