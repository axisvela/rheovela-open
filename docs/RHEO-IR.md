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
2. rejects malformed programs with an error that identifies the offending line;
3. compiles a parsed process to `WorkflowDefinition` as defined in section 6;
4. preserves compensation declarations for the compensation planner.

## 3. Lexical Rules

- The source is UTF-8 text.
- Lines are separated by `\n` (a trailing `\r` is ignored as whitespace).
- Blank lines are ignored.
- `//` starts a line comment. A `//` inside a single- or double-quoted value is
  not a comment. There are no block comments.
- Whitespace separates tokens. Indentation is insignificant.
- `ident` is any non-whitespace token that does not contain `{`, `}`, `:`, `->`
  or `//`. Implementations SHOULD use `[A-Za-z_][A-Za-z0-9_.-]*`.
- A `value` is either a double-quoted string, a single-quoted string, or a bare
  token. The v1.0 parser removes surrounding double quotes only; single quotes
  are preserved literally.
- A `duration` is a decimal number followed by one of `d`, `h`, `m`, `s`, `ms`,
  `us`, `ns`. Fractional values are allowed (for example `1.5h`).

## 4. Grammar

```abnf
program       = process-decl *( WS declaration ) WS
process-decl  = "process" WS ident [ WS version ] WS "{" [ WS "}" ]
version       = "v" 1*DIGIT
declaration   = activity / transition / compensate / delegate / evidence
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
condition     = 1*( any non-newline character )
```

The process block MAY be closed on the same line (`process P { }`), but it MUST
contain at least one `activity` declaration. Declarations after the closing `}`
are invalid.

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
- `effect_key "<template>"` and `timeout <duration>` are retained in the IR AST
  for effect-ledger and timeout integrations. They are not fields of
  `WorkflowDefinition.Stage` in `schema_version=1`.
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
| `effect_key` / `timeout` | retained in IR AST; not in `WorkflowDefinition` v1 |
| `compensate ...` | `compensations[] = {for_activity, compensate_with/capability}` |
| `delegate ...` | `stages[].delegation = {kind, delegatee, scopes, budget, evidence}` |
| `evidence ... require` | `stages[].evidence[]` |

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

The parser returns `ErrParse`-wrapped errors. The following are parse errors:

- missing `process` declaration;
- missing `{` after the process declaration;
- unterminated process block;
- process with no activities;
- unknown declaration keyword;
- malformed activity (`activity <id> : <kind>` required);
- missing attribute value;
- unknown activity attribute;
- malformed timeout or timeout unit;
- malformed transition (missing `->`, missing endpoint, or missing `when`);
- malformed compensation (missing `with`, or missing capability value);
- content after the closing `}`.

After compilation, `rheo workflow validate <file>` performs definition-level
checks: non-empty stages, unique ids, transition endpoints exist, the transition
graph is acyclic, and conditions use the supported operator set.

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
- `collaboration` participants and `messageFlow` elements are preserved in
  `WorkflowDefinition.bpmn` and are not executable transitions.
- BPMN export uses `WorkflowDefinition.bpmn` metadata to emit `subProcess`,
  boundary/intermediate events, gateway types, participants and message flows.
  Without metadata it falls back to the flat `startEvent`/`task`/`endEvent`
  subset.

## 10. Versioning and Reserved Extensions

This document specifies RHEO IR v1.0. A future grammar version MUST use a new
document version and a new process-level marker or parser entry point; v1.0
parsers MUST reject unknown declaration keywords rather than silently ignoring
them.

The following extensions are reserved and are not part of v1.0:

- migration/replan annotations;
- explicit effect-ledger actions beyond `effect_key`;
- subprocess/materialization declarations.

Until specified, authors MUST express these concerns through the compiled
`WorkflowDefinition`, runtime commands, or BPMN import metadata.

## 11. Conformance Checklist

- [ ] `process` is the first non-comment line.
- [ ] The process block is closed and contains at least one activity.
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
