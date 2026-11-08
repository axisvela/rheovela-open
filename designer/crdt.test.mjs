import test from "node:test";
import assert from "node:assert/strict";
import { createCrdt, createSync } from "./crdt.mjs";

test("materializes seeded and local edits", () => {
  const doc = createCrdt("alice", {
    workflow_id: "expense",
    title: "Expense",
    source: "designer",
    stages: [{ id: "a", title: "A" }],
    transitions: [],
    compensations: [],
  });
  doc.setStage({ id: "b", title: "B" });
  doc.setWorkflowField("title", "Expense v2");
  const wf = doc.materialize();
  assert.equal(wf.workflow_id, "expense");
  assert.equal(wf.title, "Expense v2");
  assert.deepEqual(wf.stages.map((s) => s.id), ["a", "b"]);
});

test("concurrent writes use counter then actor tie-break", () => {
  const a = createCrdt("alice", { workflow_id: "w", stages: [], transitions: [] });
  const b = createCrdt("bob", { workflow_id: "w", stages: [], transitions: [] });
  const alice = a.set("title", "from alice");
  const bob = b.set("title", "from bob");
  a.merge([bob]);
  assert.equal(a.materialize().title, "from bob");

  const later = a.set("title", "alice later");
  b.merge([alice, later]);
  assert.equal(b.materialize().title, "alice later");
});

test("merge is idempotent", () => {
  const a = createCrdt("alice", { workflow_id: "w", stages: [], transitions: [] });
  a.setStage({ id: "a", title: "A" });
  const b = createCrdt("bob", {});
  assert.equal(b.merge(a.snapshot()), a.snapshot().length);
  assert.equal(b.merge(a.snapshot()), 0, "second merge applies nothing");
  assert.equal(b.materialize().stages[0].id, "a");
});

test("stage and transition removal materializes tombstones", () => {
  const doc = createCrdt("alice", {
    workflow_id: "w",
    stages: [{ id: "a" }, { id: "b" }],
    transitions: [{ from: "a", to: "b" }],
    compensations: [],
  });
  doc.removeStage("a");
  doc.removeTransition("a", "b");
  const wf = doc.materialize();
  assert.deepEqual(wf.stages.map((s) => s.id), ["b"]);
  assert.deepEqual(wf.transitions, []);
});

test("snapshot merges into a fresh replica", () => {
  const a = createCrdt("alice", { workflow_id: "w", stages: [], transitions: [] });
  a.setStage({ id: "a", title: "A" });
  a.setTransition({ from: "a", to: "b" });
  const b = createCrdt("bob", {});
  b.merge(a.snapshot());
  const wf = b.materialize();
  assert.equal(wf.stages[0].id, "a");
  assert.equal(wf.transitions[0].from, "a");
});

test("sync queue flushes and replays until ack", () => {
  const doc = createCrdt("alice", { workflow_id: "w" });
  const sent = [];
  const sync = createSync(doc, { send: (ops) => sent.push(...ops) });
  doc.set("title", "draft");
  assert.equal(sync.flush(), 1);
  assert.equal(sync.flush(), 1, "unacked ops replay on reconnect");
  sync.ack(doc.pending());
  assert.equal(sync.flush(), 0);
  assert.ok(sent.length >= 2);
  assert.equal(sync.materialize().title, "draft");
});
