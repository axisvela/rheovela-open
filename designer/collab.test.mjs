import { test } from "node:test";
import assert from "node:assert/strict";

import { createEECollabClient } from "./collab.mjs";

function recordingFetch(response = { ok: true, status: 200, json: async () => ({ ok: true }) }) {
  const calls = [];
  return {
    calls,
    fetchImpl: async (url, init) => {
      calls.push({ url, init });
      return response;
    },
  };
}

test("sendOps posts the op batch to the document endpoint", async () => {
  const { calls, fetchImpl } = recordingFetch();
  const client = createEECollabClient({
    baseUrl: "https://ee.example/",
    documentId: "doc-1",
    headers: { authorization: "Bearer t" },
    fetchImpl,
  });

  const ops = [{ id: "a:1", key: "title", value: "x", deleted: false, counter: 1, actor: "a" }];
  await client.sendOps(ops);

  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, "https://ee.example/api/v1/ee/collab/documents/doc-1/ops");
  assert.equal(calls[0].init.method, "POST");
  assert.equal(calls[0].init.headers.authorization, "Bearer t");
  assert.equal(calls[0].init.headers["content-type"], "application/json");
  assert.deepEqual(JSON.parse(calls[0].init.body), { ops });
});

test("heartbeat sends client id and cursor", async () => {
  const { calls, fetchImpl } = recordingFetch();
  const client = createEECollabClient({ baseUrl: "https://ee.example", documentId: "doc-1", fetchImpl });

  await client.heartbeat("c1", { field: "title" });

  assert.equal(calls[0].url, "https://ee.example/api/v1/ee/collab/presence");
  assert.deepEqual(JSON.parse(calls[0].init.body), { client_id: "c1", cursor: { field: "title" } });
});

test("getDocument and presence use GET", async () => {
  const { calls, fetchImpl } = recordingFetch();
  const client = createEECollabClient({ baseUrl: "https://ee.example", documentId: "doc-1", fetchImpl });

  await client.getDocument();
  await client.presence();

  assert.equal(calls[0].url, "https://ee.example/api/v1/ee/collab/documents/doc-1");
  assert.equal(calls[0].init.method, "GET");
  assert.equal(calls[1].url, "https://ee.example/api/v1/ee/collab/presence");
});

test("non-2xx responses throw with the status", async () => {
  const { fetchImpl } = recordingFetch({ ok: false, status: 403, json: async () => ({}) });
  const client = createEECollabClient({ baseUrl: "https://ee.example", documentId: "doc-1", fetchImpl });

  await assert.rejects(() => client.getDocument(), /HTTP 403/);
});

test("snapshot converts an EE document into CRDT ops", async () => {
  const { fetchImpl } = recordingFetch({
    ok: true,
    status: 200,
    json: async () => ({
      fields: {
        title: { value: "x", clock: 2, replica: "a", deleted: false },
        "stage:s1": { value: { id: "s1" }, clock: 3, replica: "a", deleted: true },
      },
    }),
  });
  const client = createEECollabClient({ baseUrl: "https://ee.example", documentId: "doc-1", fetchImpl });

  const ops = await client.snapshot();
  assert.equal(ops.length, 2);
  const stage = ops.find((op) => op.key === "stage:s1");
  assert.equal(stage.counter, 3);
  assert.equal(stage.actor, "a");
  assert.equal(stage.deleted, true);
  assert.deepEqual(stage.value, { id: "s1" });
});

test("baseUrl and documentId are required", () => {
  assert.throws(() => createEECollabClient({ documentId: "doc-1" }), /baseUrl/);
  assert.throws(() => createEECollabClient({ baseUrl: "https://ee.example" }), /documentId/);
});
