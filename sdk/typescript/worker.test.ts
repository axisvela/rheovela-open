import { describe, test, afterEach, mock } from "node:test";
import assert from "node:assert/strict";
import { Client, Worker } from "./worker.ts";

interface RecordedCall {
  method: string;
  path: string;
  body?: unknown;
}

function mockWorkerApi() {
  const records: RecordedCall[] = [];
  const failRecords: RecordedCall["body"][] = [];
  const conflictIds = new Set<string>();
  const token = "tok-1";
  let pollItems: unknown[] = [];

  const fakeFetch = mock.fn(
    (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      const url = String(input);
      const method = init?.method ?? "GET";
      const body = init?.body ? JSON.parse(String(init.body)) : undefined;
      const rec: RecordedCall = { method, path: url.replace("http://fake", ""), body };
      records.push(rec);

      const json = (code: number, obj: unknown) =>
        Promise.resolve(
          new Response(JSON.stringify(obj), {
            status: code,
            headers: { "Content-Type": "application/json" },
          }),
        );

      if (method === "GET" && url.endsWith("/api/v1/work-items")) {
        return json(200, pollItems);
      }
      if (method === "POST" && url.endsWith("/claim")) {
        const id = url.split("/")[6];
        if (conflictIds.has(id)) return json(409, { error: "conflict" });
        return json(200, { token, lease: body?.lease ?? "30s" });
      }
      if (method === "POST" && url.endsWith("/complete")) {
        return json(200, { ok: true });
      }
      if (method === "POST" && url.endsWith("/fail")) {
        failRecords.push(body);
        return json(200, { ok: true });
      }
      if (method === "POST" && url.endsWith("/heartbeat")) {
        return json(200, { ok: true });
      }
      return json(404, { error: "not found" });
    },
  );

  const prev = globalThis.fetch;
  globalThis.fetch = fakeFetch as unknown as typeof fetch;
  afterEach(() => {
    globalThis.fetch = prev;
  });

  return {
    records,
    failRecords,
    setConflict: (id: string) => conflictIds.add(id),
    setPollItems: (items: unknown[]) => {
      pollItems = items;
    },
  };
}

describe("Worker", () => {
  test("claim then complete -> processOnce returns 1", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1", instance_id: "i1", activity_id: "a1", state: "ready" }]);
    const client = new Client("http://fake");
    const calls: unknown[] = [];
    const n = await new Worker(client, async (item) => {
      calls.push(item);
      return { status: "success" } as const;
    }).processOnce();

    assert.equal(n, 1);
    assert.deepEqual(calls, [{ id: "wi1", instance_id: "i1", activity_id: "a1", state: "ready" }]);
    assert.deepEqual(
      api.records.map((r) => r.path),
      [
        "/api/v1/work-items",
        "/api/v1/work-items/wi1/claim",
        "/api/v1/work-items/wi1/complete",
      ],
    );
    assert.deepEqual(api.records[2].body, { token: "tok-1" });
  });

  test("claim conflict (409) skipped -> 0", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1" }]);
    api.setConflict("wi1");
    const n = await new Worker(new Client("http://fake"), async () => ({
      status: "success",
    } as const)).processOnce();

    assert.equal(n, 0);
    assert.deepEqual(
      api.records.map((r) => r.path),
      ["/api/v1/work-items", "/api/v1/work-items/wi1/claim"],
    );
  });

  test("fail records error", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1" }]);
    const n = await new Worker(new Client("http://fake"), async () => ({
      status: "failure",
      error: "boom",
    } as const)).processOnce();

    assert.equal(n, 1);
    assert.deepEqual(api.failRecords, [{ token: "tok-1", error: "boom" }]);
  });
});
