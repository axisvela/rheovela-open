import test from "node:test";
import assert from "node:assert/strict";

import { CoreAPIError, importBPMN, listWorkflows, loadWorkflow, publishWorkflow } from "./core.mjs";

function mockFetch(handler) {
  const calls = [];
  const impl = async (url, opts = {}) => {
    const call = { url: String(url), opts };
    calls.push(call);
    return handler(call);
  };
  return { impl, calls };
}

function jsonResponse(status, body) {
  return {
    ok: status >= 200 && status < 300,
    status,
    text: async () => (typeof body === "string" ? body : JSON.stringify(body)),
  };
}

test("loadWorkflow calls the core workflow endpoint", async () => {
  const wf = { workflow_id: "expense", title: "Expense", stages: [], transitions: [] };
  const { impl, calls } = mockFetch(() => jsonResponse(200, wf));
  const got = await loadWorkflow("http://core:8080/", "expense", { fetchImpl: impl });
  assert.deepEqual(got, wf);
  assert.equal(calls[0].url, "http://core:8080/api/v1/workflows/expense");
  assert.equal(calls[0].opts.method, "GET");
});

test("listWorkflows returns the latest definitions", async () => {
  const { impl, calls } = mockFetch(() => jsonResponse(200, [{ workflow_id: "a" }]));
  const got = await listWorkflows("http://core", { fetchImpl: impl });
  assert.deepEqual(got, [{ workflow_id: "a" }]);
  assert.equal(calls[0].url, "http://core/api/v1/workflows");
});

test("publishWorkflow sends JSON and tenant/bearer headers", async () => {
  const wf = { workflow_id: "expense", title: "Expense", source: "designer", stages: [], transitions: [] };
  const { impl, calls } = mockFetch(() => jsonResponse(201, { workflow_id: "expense", version: 3 }));
  const got = await publishWorkflow("http://core", wf, {
    fetchImpl: impl,
    tenant: "acme",
    token: "t0ken",
  });
  assert.deepEqual(got, { workflow_id: "expense", version: 3 });
  const call = calls[0];
  assert.equal(call.url, "http://core/api/v1/workflows");
  assert.equal(call.opts.method, "POST");
  assert.equal(call.opts.headers["X-Tenant-ID"], "acme");
  assert.equal(call.opts.headers.Authorization, "Bearer t0ken");
  assert.deepEqual(JSON.parse(call.opts.body), wf);
});

test("importBPMN posts XML and returns the preview", async () => {
  const preview = { workflow: { workflow_id: "imported" }, report: { unmapped: [] } };
  const { impl, calls } = mockFetch(() => jsonResponse(200, preview));
  const xml = "<bpmn:definitions/>";
  const got = await importBPMN("http://core", xml, { fetchImpl: impl, tenant: "acme" });
  assert.deepEqual(got, preview);
  const call = calls[0];
  assert.equal(call.url, "http://core/api/v1/workflows/import-bpmn");
  assert.equal(call.opts.method, "POST");
  assert.equal(call.opts.headers["Content-Type"], "application/xml");
  assert.equal(call.opts.headers["X-Tenant-ID"], "acme");
  assert.equal(call.opts.body, xml);
});

test("non-2xx responses throw CoreAPIError", async () => {
  const { impl } = mockFetch(() => jsonResponse(409, { error: "conflict" }));
  await assert.rejects(
    loadWorkflow("http://core", "expense", { fetchImpl: impl }),
    (err) => err instanceof CoreAPIError && err.status === 409 && err.body.includes("conflict"),
  );
});
