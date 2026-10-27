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
- Edit per-stage title, assignee spec, evidence, effect key, timeout, migration
  policy, and delegation (kind/delegatee/scopes/budget/evidence).
- Add/remove transitions with optional `when` conditions.
- Render a simple layered DAG preview.
- Render preserved BPMN DI shapes and edges (`bpmn.diagram`) when a workflow was
  imported from BPMN.
- Undo/redo for structural edits, stage edits, text fields, JSON apply, and
  imports (`Ctrl/Cmd+Z`, `Ctrl/Cmd+Shift+Z`, `Ctrl+Y`).
- Import preview: show a stage/transition/compensation/BPMN diff before applying
  a JSON or BPMN import.
- Validate required fields, duplicate stage ids, missing transition endpoints,
  cycles, compensation references, and delegation kinds.
- Import/export `WorkflowDefinition` JSON.
- Preserve delegation/evidence metadata (`Stage.delegation`, `Stage.evidence`).
- Load a sample expense-approval workflow.
- Optional Core API integration:
  - `Load from core` uses `GET /api/v1/workflows/{id}`.
  - `Publish to core` uses `POST /api/v1/workflows`.
  - `List workflows` uses `GET /api/v1/workflows`.
  - `Import BPMN` previews XML via `POST /api/v1/workflows/import-bpmn` and
    loads the imported definition into the editor.
  - Optional `X-Tenant-ID` and `Authorization: Bearer` headers.

### Core API / CORS

The designer is static and fetches the configured core URL from the browser. If
the page and core are not same-origin, the browser requires CORS on the core
server (or a same-origin reverse proxy). Publish is blocked while the local
validation panel reports errors.

## Scope

This is a community editor, not an enterprise console. It renders preserved BPMN
DI layout but is not a full BPMN modeler: it does not edit DI shapes, publish to
a running core server, manage tenants, or approve migrations. The exported JSON
can be validated with `rheo workflow validate` and imported with
`rheo workflow define`.

## Tests

```sh
node --test designer/model.test.mjs designer/core.test.mjs \
  designer/history.test.mjs designer/diagram.test.mjs designer/diff.test.mjs
```

The model tests cover structural validation, DAG cycles, stage/transition
removal cascades, JSON round-trips, and delegation/evidence preservation. The
core client tests cover load/list/publish URLs, headers, request bodies, and
non-2xx error handling. History, DI layout, and import diff tests cover undo/redo
truncation, viewport normalization, and added/removed/changed metadata.
