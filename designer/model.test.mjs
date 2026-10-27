import test from "node:test";
import assert from "node:assert/strict";

import {
  addStage,
  addTransition,
  emptyWorkflow,
  normalizeWorkflow,
  removeStage,
  removeTransition,
  serializeWorkflow,
  updateStage,
  validateWorkflow,
} from "./model.mjs";

test("empty workflow validates with the expected structural errors", () => {
  const errors = validateWorkflow(emptyWorkflow());
  assert.ok(errors.some((e) => e.includes("stages")));
});

test("valid workflow has no errors", () => {
  const wf = {
    workflow_id: "expense",
    title: "Expense",
    source: "designer",
    stages: [
      { id: "start", title: "Start", inputs: [], outputs: [] },
      { id: "approve", title: "Approve", inputs: [], outputs: [] },
    ],
    transitions: [{ from: "start", to: "approve" }],
  };
  assert.deepEqual(validateWorkflow(wf), []);
});

test("duplicate stage ids are rejected", () => {
  const wf = addStage(emptyWorkflow(), "a", "A");
  wf.stages.push({ id: "a", title: "Duplicate", inputs: [], outputs: [] });
  assert.ok(validateWorkflow(wf).some((e) => e.includes("duplicate")));
});

test("missing transition endpoint is rejected", () => {
  const wf = addStage(emptyWorkflow(), "a", "A");
  wf.transitions.push({ from: "a", to: "ghost" });
  assert.ok(validateWorkflow(wf).some((e) => e.includes("ghost")));
});

test("cycle is rejected", () => {
  let wf = addStage(emptyWorkflow(), "a", "A");
  wf = addStage(wf, "b", "B");
  wf = addTransition(wf, "a", "b");
  wf = addTransition(wf, "b", "a");
  assert.ok(validateWorkflow(wf).some((e) => e.includes("cycle")));
});

test("invalid delegation kind is rejected", () => {
  const wf = addStage(emptyWorkflow(), "a", "A");
  wf.stages[0].delegation = { kind: "robot", delegatee: "x" };
  assert.ok(validateWorkflow(wf).some((e) => e.includes("delegation")));
});

test("removing a stage cascades transitions and compensations", () => {
  let wf = addStage(emptyWorkflow(), "a", "A");
  wf = addStage(wf, "b", "B");
  wf = addTransition(wf, "a", "b");
  wf.compensations = [{ for_activity: "a", capability: "undo.a" }];
  wf.bpmn = { lanes: [{ id: "lane", node_ids: ["a", "b"] }] };
  wf = removeStage(wf, "a");
  assert.deepEqual(wf.stages.map((s) => s.id), ["b"]);
  assert.deepEqual(wf.transitions, []);
  assert.deepEqual(wf.compensations, []);
  assert.deepEqual(wf.bpmn.lanes[0].node_ids, ["b"]);
});

test("removeTransition removes only the selected edge", () => {
  let wf = addStage(emptyWorkflow(), "a", "A");
  wf = addStage(wf, "b", "B");
  wf = addTransition(wf, "a", "b");
  wf = addTransition(wf, "b", "a");
  wf = removeTransition(wf, 0);
  assert.deepEqual(wf.transitions, [{ from: "b", to: "a" }]);
});

test("updateStage edits title, assignee, evidence, and delegation", () => {
  let wf = addStage(emptyWorkflow(), "a", "A");
  wf = updateStage(wf, "a", {
    title: "Approve",
    assignee: { kind: "role", role: "lead" },
    evidence: ["receipt"],
    delegation: { kind: "role", delegatee: "lead", scopes: ["approve"], budget: 10, evidence: ["delegation"] },
  });
  const stage = wf.stages[0];
  assert.equal(stage.title, "Approve");
  assert.deepEqual(stage.assignee_spec, { kind: "role", role: "lead" });
  assert.deepEqual(stage.evidence, ["receipt"]);
  assert.equal(stage.delegation.delegatee, "lead");

  wf = updateStage(wf, "a", { delegation: null, evidence: [] });
  assert.equal(wf.stages[0].delegation, undefined);
  assert.deepEqual(wf.stages[0].evidence, []);
});

test("updateStage rejects an unknown stage", () => {
  assert.throws(() => updateStage(emptyWorkflow(), "ghost", { title: "x" }), /unknown stage/);
});

test("serialize and normalize round-trip", () => {
  const wf = addStage(emptyWorkflow(), "a", "A");
  const text = serializeWorkflow(wf);
  const parsed = normalizeWorkflow(text);
  assert.equal(parsed.workflow_id, wf.workflow_id);
  assert.deepEqual(parsed.stages, wf.stages);
});

test("normalize preserves delegation and evidence", () => {
  const wf = normalizeWorkflow({
    workflow_id: "w",
    title: "W",
    source: "designer",
    stages: [
      {
        id: "a",
        title: "A",
        inputs: [],
        outputs: [],
        evidence: ["receipt"],
        delegation: { kind: "role", delegatee: "lead", scopes: ["approve"], budget: 10 },
      },
    ],
    transitions: [],
  });
  assert.deepEqual(wf.stages[0].evidence, ["receipt"]);
  assert.equal(wf.stages[0].delegation.kind, "role");
});
