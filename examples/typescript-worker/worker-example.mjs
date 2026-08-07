#!/usr/bin/env node
// Runnable TypeScript worker example as ESM (Node 18+).
//
// Node cannot import sdk/typescript/worker.ts without a transform step, so this
// file inlines a minimal copy of the Client/Worker classes from
// sdk/typescript/worker.ts and mocks globalThis.fetch against an in-memory
// Worker-API. task-1 succeeds, task-2 fails.
//
// Run:
//     node examples/typescript-worker/worker-example.mjs

class WorkerError extends Error {
  constructor(status, message) {
    super(message);
    this.status = status;
    this.name = "WorkerError";
  }
}

class Client {
  constructor(baseUrl) {
    this.baseUrl = baseUrl;
  }

  async req(method, path, body) {
    const res = await fetch(this.baseUrl + path, {
      method,
      headers: { "Content-Type": "application/json" },
      body: body ? JSON.stringify(body) : undefined,
    });
    const text = await res.text();
    if (!res.ok) throw new WorkerError(res.status, text);
    return text ? JSON.parse(text) : {};
  }

  poll(instanceId) {
    const path = instanceId
      ? `/api/v1/work-items?instance=${encodeURIComponent(instanceId)}`
      : "/api/v1/work-items";
    return this.req("GET", path);
  }

  claim(id, lease) {
    return this.req("POST", `/api/v1/work-items/${id}/claim`, { lease });
  }

  heartbeat(id, token, lease) {
    return this.req("POST", `/api/v1/work-items/${id}/heartbeat`, { token, lease });
  }

  complete(id, token) {
    return this.req("POST", `/api/v1/work-items/${id}/complete`, { token });
  }

  fail(id, token, error) {
    return this.req("POST", `/api/v1/work-items/${id}/fail`, { token, error });
  }
}

class Worker {
  constructor(client, fn, lease = "30s") {
    this.client = client;
    this.fn = fn;
    this.lease = lease;
  }

  async processOnce(instanceId) {
    let processed = 0;
    for (const item of await this.client.poll(instanceId)) {
      let token;
      try {
        token = (await this.client.claim(item.id, this.lease)).token;
      } catch {
        continue; // claimed by another worker
      }
      const res = await this.fn(item);
      try {
        if (res.status === "success") await this.client.complete(item.id, token);
        else await this.client.fail(item.id, token, res.error ?? "failure");
        processed++;
      } catch {
        /* stale token / expired lease */
      }
    }
    return processed;
  }
}

// ---- in-memory Worker-API mock (mirrors sdk/typescript/worker.test.ts) ----

const items = {
  "task-1": { id: "task-1", instance_id: "inst-task-1", activity_id: "act", state: "ready" },
  "task-2": { id: "task-2", instance_id: "inst-task-2", activity_id: "act", state: "ready" },
};
const tokens = {};
const failRecords = [];

function json(code, obj) {
  return new Response(JSON.stringify(obj), {
    status: code,
    headers: { "Content-Type": "application/json" },
  });
}

globalThis.fetch = async (input, init) => {
  const url = String(input);
  const method = init?.method ?? "GET";
  const body = init?.body ? JSON.parse(String(init.body)) : undefined;
  const itemId = url.split("/")[6];

  if (method === "GET" && url.endsWith("/api/v1/work-items")) {
    return json(200, Object.values(items));
  }
  if (method === "POST" && url.endsWith("/claim")) {
    const item = items[itemId];
    if (!item || item.state !== "ready") return json(409, { error: "conflict" });
    tokens[itemId] = "tok-" + itemId;
    return json(200, { token: tokens[itemId], lease: body?.lease ?? "30s" });
  }
  if (method === "POST" && url.endsWith("/complete")) {
    const item = items[itemId];
    if (!item || tokens[itemId] !== body?.token) return json(409, { error: "stale token" });
    item.state = "done";
    return json(200, { ok: true });
  }
  if (method === "POST" && url.endsWith("/fail")) {
    const item = items[itemId];
    if (!item || tokens[itemId] !== body?.token) return json(409, { error: "stale token" });
    item.state = "failed";
    item.error = body?.error ?? "failure";
    failRecords.push(body);
    return json(200, { ok: true });
  }
  if (method === "POST" && url.endsWith("/heartbeat")) {
    return json(200, { ok: true });
  }
  return json(404, { error: "not found" });
};

// ---- run the worker ----

const client = new Client("http://fake");
const fn = async (item) =>
  item.id === "task-1"
    ? { status: "success" }
    : { status: "failure", error: "simulated failure" };

const processed = await new Worker(client, fn, "30s").processOnce();
console.log(`processed=${processed}`);
for (const id of Object.keys(items).sort()) {
  console.log(`  ${id}: ${items[id].state}`);
}
