import { describe, test, afterEach, mock } from "node:test";
import assert from "node:assert/strict";
import {
  Client,
  Worker,
  failure,
  requestApproval,
  success,
} from "./worker.ts";
import type { ApprovalRequest } from "./worker.ts";

interface RecordedCall {
  method: string;
  path: string;
  body?: unknown;
}

const FULL_ITEM = {
  id: "wi1",
  instance_id: "i1",
  activity_id: "a1",
  state: "ready",
  result: { ok: true },
  checkpoint_ref: "ckpt-1",
  effect_cursor: "cur-1",
  attempts: 2,
  max_attempts: 5,
  last_error: "boom",
  approval: {
    state: "pending",
    pending_stage: "review",
    final_args: { x: 1 },
    reason: "needs review",
    requested_by: "agent",
    requested_at: "2026-10-14T00:00:00Z",
    decided_by: "human",
    decided_at: "2026-10-14T01:00:00Z",
    comment: "looks good",
    checkpoint_ref: "ckpt-1",
    effect_cursor: "cur-1",
  },
};

function mockWorkerApi() {
  const records: RecordedCall[] = [];
  const completeRecords: RecordedCall["body"][] = [];
  const approvalRecords: RecordedCall["body"][] = [];
  const approveRecords: RecordedCall["body"][] = [];
  const rejectRecords: RecordedCall["body"][] = [];
  const cancelRecords: RecordedCall["body"][] = [];
  const failRecords: RecordedCall["body"][] = [];
  const conflictIds = new Set<string>();
  const workItems = new Map<string, unknown>();
  const token = "tok-1";
  let pollItems: unknown[] = [];

  const fakeFetch = mock.fn(
    (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      const url = new URL(String(input));
      const method = init?.method ?? "GET";
      const body = init?.body ? JSON.parse(String(init.body)) : undefined;
      const rec: RecordedCall = {
        method,
        path: url.pathname + url.search,
        body,
      };
      records.push(rec);

      const json = (code: number, obj: unknown) =>
        Promise.resolve(
          new Response(JSON.stringify(obj), {
            status: code,
            headers: { "Content-Type": "application/json" },
          }),
        );

      const parts = url.pathname.split("/");

      if (method === "GET" && url.pathname === "/api/v1/work-items") {
        return json(200, pollItems);
      }
      if (
        method === "GET" &&
        parts[3] === "work-items" &&
        parts[4] &&
        !parts[5]
      ) {
        const item = workItems.get(parts[4]);
        return item === undefined
          ? json(404, { error: "not found" })
          : json(200, item);
      }
      if (method === "POST" && parts[3] === "work-items") {
        const id = parts[4];
        const action = parts[5];
        if (action === "claim") {
          if (conflictIds.has(id)) return json(409, { error: "conflict" });
          return json(200, { token, lease: body?.lease ?? "30s" });
        }
        if (action === "complete") {
          completeRecords.push(body);
          return json(200, { ok: true });
        }
        if (action === "request-approval") {
          approvalRecords.push(body);
          return json(200, { ok: true });
        }
        if (action === "approve") {
          approveRecords.push(body);
          return json(200, { ok: true });
        }
        if (action === "reject") {
          rejectRecords.push(body);
          return json(200, { ok: true });
        }
        if (action === "cancel") {
          cancelRecords.push(body);
          return json(200, { ok: true });
        }
        if (action === "fail") {
          failRecords.push(body);
          return json(200, { ok: true });
        }
        if (action === "heartbeat") {
          return json(200, { ok: true });
        }
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
    completeRecords,
    approvalRecords,
    approveRecords,
    rejectRecords,
    cancelRecords,
    failRecords,
    setConflict: (id: string) => conflictIds.add(id),
    setPollItems: (items: unknown[]) => {
      pollItems = items;
    },
    setWorkItem: (id: string, item: unknown) => workItems.set(id, item),
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
      return success();
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
    assert.deepEqual(api.completeRecords, [{ token: "tok-1" }]);
  });

  test("claim conflict (409) skipped -> 0", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1" }]);
    api.setConflict("wi1");
    const n = await new Worker(new Client("http://fake"), async () =>
      success(),
    ).processOnce();

    assert.equal(n, 0);
    assert.deepEqual(
      api.records.map((r) => r.path),
      ["/api/v1/work-items", "/api/v1/work-items/wi1/claim"],
    );
  });

  test("fail records error", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1" }]);
    const n = await new Worker(new Client("http://fake"), async () =>
      failure("boom"),
    ).processOnce();

    assert.equal(n, 1);
    assert.deepEqual(api.failRecords, [{ token: "tok-1", error: "boom" }]);
  });

  test("worker success sends result and cursors", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1" }]);
    const n = await new Worker(new Client("http://fake"), async () =>
      success({ ok: true }, "ckpt-1", "cur-1"),
    ).processOnce();

    assert.equal(n, 1);
    assert.deepEqual(api.completeRecords, [
      {
        token: "tok-1",
        result: { ok: true },
        checkpoint_ref: "ckpt-1",
        effect_cursor: "cur-1",
      },
    ]);
  });

  test("worker approval path merges cursors", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1" }]);
    const req: ApprovalRequest = {
      actor: "agent",
      reason: "over limit",
      final_args: { amount: 10 },
    };
    const n = await new Worker(new Client("http://fake"), async () =>
      requestApproval(req),
    ).processOnce();

    assert.equal(n, 1);
    assert.deepEqual(
      api.records.map((r) => r.path),
      [
        "/api/v1/work-items",
        "/api/v1/work-items/wi1/claim",
        "/api/v1/work-items/wi1/request-approval",
      ],
    );
    assert.deepEqual(api.approvalRecords, [
      {
        token: "tok-1",
        actor: "agent",
        reason: "over limit",
        final_args: { amount: 10 },
      },
    ]);
  });

  test("worker approval path fills missing cursors from result", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1" }]);
    const req: ApprovalRequest = {
      actor: "agent",
      reason: "over limit",
      final_args: { amount: 10 },
    };
    const n = await new Worker(new Client("http://fake"), async () => ({
      status: "approval" as const,
      approval: req,
      checkpoint_ref: "ckpt-1",
      effect_cursor: "cur-1",
    })).processOnce();

    assert.equal(n, 1);
    assert.deepEqual(api.approvalRecords, [
      {
        token: "tok-1",
        actor: "agent",
        reason: "over limit",
        final_args: { amount: 10 },
        checkpoint_ref: "ckpt-1",
        effect_cursor: "cur-1",
      },
    ]);
    assert.equal(req.checkpoint_ref, undefined);
    assert.equal(req.effect_cursor, undefined);
  });

  test("worker approval without request fails", async () => {
    const api = mockWorkerApi();
    api.setPollItems([{ id: "wi1" }]);
    const n = await new Worker(new Client("http://fake"), async () => ({
      status: "approval" as const,
    })).processOnce();

    assert.equal(n, 1);
    assert.deepEqual(api.failRecords, [
      {
        token: "tok-1",
        error: "approval result is missing approval request",
      },
    ]);
  });
});

describe("Client", () => {
  test("complete with result and cursors body", async () => {
    const api = mockWorkerApi();
    const client = new Client("http://fake");

    await client.complete("wi1", "tok-1", {
      result: { ok: true },
      checkpoint_ref: "ckpt-1",
      effect_cursor: "cur-1",
    });

    assert.deepEqual(api.completeRecords, [
      {
        token: "tok-1",
        result: { ok: true },
        checkpoint_ref: "ckpt-1",
        effect_cursor: "cur-1",
      },
    ]);
  });

  test("complete token-only remains compatible", async () => {
    const api = mockWorkerApi();
    const client = new Client("http://fake");

    await client.complete("wi1", "tok-1");

    assert.deepEqual(api.completeRecords, [{ token: "tok-1" }]);
  });

  test("get work item parses fields", async () => {
    const api = mockWorkerApi();
    api.setWorkItem("wi1", FULL_ITEM);
    const client = new Client("http://fake");

    const item = await client.getWorkItem("wi1");

    assert.equal(item.id, "wi1");
    assert.equal(item.instance_id, "i1");
    assert.equal(item.activity_id, "a1");
    assert.equal(item.state, "ready");
    assert.deepEqual(item.result, { ok: true });
    assert.equal(item.checkpoint_ref, "ckpt-1");
    assert.equal(item.effect_cursor, "cur-1");
    assert.equal(item.attempts, 2);
    assert.equal(item.max_attempts, 5);
    assert.equal(item.last_error, "boom");
    assert.equal(item.approval?.state, "pending");
    assert.equal(item.approval?.pending_stage, "review");
    assert.deepEqual(item.approval?.final_args, { x: 1 });
    assert.equal(item.approval?.reason, "needs review");
    assert.equal(item.approval?.requested_by, "agent");
    assert.equal(item.approval?.requested_at, "2026-10-14T00:00:00Z");
    assert.equal(item.approval?.decided_by, "human");
    assert.equal(item.approval?.decided_at, "2026-10-14T01:00:00Z");
    assert.equal(item.approval?.comment, "looks good");
    assert.equal(item.approval?.checkpoint_ref, "ckpt-1");
    assert.equal(item.approval?.effect_cursor, "cur-1");
    assert.deepEqual(
      api.records.map((r) => r.path),
      ["/api/v1/work-items/wi1"],
    );
  });

  test("request approval body", async () => {
    const api = mockWorkerApi();
    const client = new Client("http://fake");

    await client.requestApproval("wi1", "tok-1", {
      actor: "agent",
      reason: "over limit",
      final_args: { amount: 10 },
      checkpoint_ref: "ckpt-1",
      effect_cursor: "cur-1",
    });

    assert.deepEqual(api.approvalRecords, [
      {
        token: "tok-1",
        actor: "agent",
        reason: "over limit",
        final_args: { amount: 10 },
        checkpoint_ref: "ckpt-1",
        effect_cursor: "cur-1",
      },
    ]);
  });

  test("approve, reject, cancel paths", async () => {
    const api = mockWorkerApi();
    const client = new Client("http://fake");

    await client.approve("wi1", "human", "ok", { approved: true });
    await client.reject("wi2", "human", "no");
    await client.cancel("wi3");

    assert.deepEqual(
      api.records.map((r) => r.path),
      [
        "/api/v1/work-items/wi1/approve",
        "/api/v1/work-items/wi2/reject",
        "/api/v1/work-items/wi3/cancel",
      ],
    );
    assert.deepEqual(api.approveRecords, [
      { actor: "human", comment: "ok", final_args: { approved: true } },
    ]);
    assert.deepEqual(api.rejectRecords, [{ actor: "human", reason: "no" }]);
    assert.deepEqual(api.cancelRecords, [{}]);
  });

  test("global poll returns claimable items", async () => {
    const api = mockWorkerApi();
    api.setPollItems([
      FULL_ITEM,
      {
        id: "wi2",
        instance_id: "i2",
        activity_id: "a2",
        state: "claimed",
      },
    ]);
    const client = new Client("http://fake");

    const items = await client.poll();

    assert.deepEqual(
      items.map((i) => i.id),
      ["wi1", "wi2"],
    );
    assert.deepEqual(
      items.map((i) => i.state),
      ["ready", "claimed"],
    );
    assert.deepEqual(items[0].result, { ok: true });
    assert.equal(items[0].approval?.state, "pending");
    assert.equal(items[1].approval, undefined);
    assert.deepEqual(
      api.records.map((r) => r.path),
      ["/api/v1/work-items"],
    );
  });

  test("poll filters by instance", async () => {
    const api = mockWorkerApi();
    const client = new Client("http://fake");

    assert.deepEqual(await client.poll("inst 1"), []);
    assert.deepEqual(
      api.records.map((r) => r.path),
      ["/api/v1/work-items?instance=inst%201"],
    );
  });

  test("heartbeat renews lease", async () => {
    const api = mockWorkerApi();
    const client = new Client("http://fake");

    await client.heartbeat("wi1", "tok-1", "60s");

    assert.deepEqual(api.records[0].body, { token: "tok-1", lease: "60s" });
  });
});
