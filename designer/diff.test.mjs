import test from "node:test";
import assert from "node:assert/strict";

import { diffWorkflows } from "./model.mjs";

function workflow(overrides = {}) {
  return {
    workflow_id: "expense",
    title: "Expense",
    source: "designer",
    stages: [
      { id: "start", title: "Start", inputs: [], outputs: [] },
      { id: "approve", title: "Approve", inputs: [], outputs: [] },
    ],
    transitions: [{ from: "start", to: "approve" }],
    compensations: [],
    ...overrides,
  };
}

test("identical workflows produce an empty diff", () => {
  const diff = diffWorkflows(workflow(), workflow());
  assert.equal(diff.empty, true);
  assert.deepEqual(diff.stages.added, []);
  assert.deepEqual(diff.stages.removed, []);
  assert.deepEqual(diff.stages.changed, []);
});

test("stage additions, removals and field changes are reported", () => {
  const before = workflow();
  const after = workflow({
    title: "Expense v2",
    stages: [
      { id: "start", title: "Start", inputs: [], outputs: [], effect_key: "key" },
      { id: "pay", title: "Pay", inputs: [], outputs: [] },
    ],
  });
  const diff = diffWorkflows(before, after);
  assert.deepEqual(diff.stages.added, ["pay"]);
  assert.deepEqual(diff.stages.removed, ["approve"]);
  assert.deepEqual(diff.stages.changed, [{ id: "start", fields: ["effect_key"] }]);
  assert.deepEqual(diff.workflow, [{ field: "title", before: "Expense", after: "Expense v2" }]);
  assert.equal(diff.empty, false);
});

test("transition and compensation diffs use readable keys", () => {
  const before = workflow({
    transitions: [
      { from: "start", to: "approve" },
      { from: "approve", to: "end", condition: "amount > 0" },
    ],
    compensations: [{ for_activity: "approve", capability: "approve.reverse" }],
  });
  const after = workflow({
    transitions: [{ from: "start", to: "approve", condition: "amount >= 1000" }],
    compensations: [{ for_activity: "approve", compensate_with: "start" }],
  });
  const diff = diffWorkflows(before, after);
  assert.deepEqual(diff.transitions.added, ["start -> approve when amount >= 1000"]);
  assert.deepEqual(diff.transitions.removed, [
    "start -> approve",
    "approve -> end when amount > 0",
  ]);
  assert.deepEqual(diff.compensations.added, ["approve with start"]);
  assert.deepEqual(diff.compensations.removed, ["approve with capability approve.reverse"]);
});

test("BPMN metadata additions are counted", () => {
  const before = workflow();
  const after = workflow({
    bpmn: {
      nodes: [{ id: "start", element: "startEvent" }],
      diagram: {
        shapes: [{ id: "s1", bpmn_element: "start", x: 0, y: 0, width: 36, height: 36 }],
        edges: [],
      },
    },
  });
  const diff = diffWorkflows(before, after);
  assert.equal(diff.bpmn.before, false);
  assert.equal(diff.bpmn.after, true);
  assert.deepEqual(diff.bpmn.nodes.added, ["start"]);
  assert.deepEqual(diff.bpmn.shapes.added, ["s1"]);
  assert.equal(diff.empty, false);
});
