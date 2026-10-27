import test from "node:test";
import assert from "node:assert/strict";

import { createHistory } from "./history.mjs";

test("history starts with no undo or redo", () => {
  const history = createHistory({ id: 1 });
  assert.equal(history.canUndo(), false);
  assert.equal(history.canRedo(), false);
  assert.deepEqual(history.current(), { id: 1 });
});

test("undo and redo walk snapshots", () => {
  const history = createHistory({ id: 1 });
  history.push({ id: 2 });
  history.push({ id: 3 });
  assert.equal(history.canUndo(), true);
  assert.deepEqual(history.undo(), { id: 2 });
  assert.deepEqual(history.undo(), { id: 1 });
  assert.equal(history.canUndo(), false);
  assert.equal(history.canRedo(), true);
  assert.deepEqual(history.redo(), { id: 2 });
  assert.deepEqual(history.redo(), { id: 3 });
  assert.equal(history.canRedo(), false);
});

test("a new push truncates redo entries", () => {
  const history = createHistory({ id: 1 });
  history.push({ id: 2 });
  history.undo();
  history.push({ id: 9 });
  assert.equal(history.canRedo(), false);
  assert.deepEqual(history.current(), { id: 9 });
  assert.deepEqual(history.undo(), { id: 1 });
});

test("history is limited and keeps the newest entries", () => {
  const history = createHistory({ id: 0 }, 3);
  history.push({ id: 1 });
  history.push({ id: 2 });
  history.push({ id: 3 });
  assert.equal(history.size(), 3);
  assert.deepEqual(history.current(), { id: 3 });
  assert.deepEqual(history.undo(), { id: 2 });
  assert.deepEqual(history.undo(), { id: 1 });
  assert.equal(history.canUndo(), false);
});

test("reset replaces the whole history", () => {
  const history = createHistory({ id: 1 });
  history.push({ id: 2 });
  history.reset({ id: 9 });
  assert.equal(history.canUndo(), false);
  assert.equal(history.canRedo(), false);
  assert.deepEqual(history.current(), { id: 9 });
});
