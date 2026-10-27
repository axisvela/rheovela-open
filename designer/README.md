# RHEOVELA Community Process Designer

A zero-dependency, static designer for `WorkflowDefinition` JSON. It is aligned
with the public contracts (`contracts/types.go`, `api/workflow.schema.json`) and
the OpenAPI surface, but it does not call the core API directly.

## Run

Open `designer/index.html` in a browser, or serve the repository root:

```sh
python -m http.server 8080
# then open http://127.0.0.1:8080/designer/
```

No build step or external JavaScript dependency is required.

## Features

- Edit workflow id/title and add/remove stages.
- Add/remove transitions with optional `when` conditions.
- Render a simple layered DAG preview.
- Validate required fields, duplicate stage ids, missing transition endpoints,
  cycles, compensation references, and delegation kinds.
- Import/export `WorkflowDefinition` JSON.
- Preserve delegation/evidence metadata (`Stage.delegation`, `Stage.evidence`).
- Load a sample expense-approval workflow.

## Scope

This is a community editor, not an enterprise console. It does not publish to a
running core server, manage tenants, approve migrations, or render BPMN DI
layout. The exported JSON can be validated with `rheo workflow validate` and
imported with `rheo workflow define`.

## Tests

```sh
node --test designer/model.test.mjs
```

The model tests cover structural validation, DAG cycles, stage/transition
removal cascades, JSON round-trips, and delegation/evidence preservation.
