> English · [中文](README.md)

# TypeScript Worker SDK — `sdk/typescript`

Dependency-free HTTP client for the core Worker API exposed by `rheo serve`
(see `api/README.md` and `api/openapi.yaml`). Uses only the built-in `fetch`
globally available since Node 18. No runtime dependencies, no build step —
import the file directly.

## Worker API routes

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/work-items[?instance=<id>]` | list work items (global poll = claimable items) |
| `GET` | `/api/v1/work-items/{id}` | get one work item with result/approval |
| `POST` | `/api/v1/work-items/{id}/claim` | claim (returns fencing token) |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | renew lease |
| `POST` | `/api/v1/work-items/{id}/complete` | complete with optional result/cursors |
| `POST` | `/api/v1/work-items/{id}/request-approval` | request human approval |
| `POST` | `/api/v1/work-items/{id}/approve` | approve a pending item |
| `POST` | `/api/v1/work-items/{id}/reject` | reject a pending item |
| `POST` | `/api/v1/work-items/{id}/cancel` | cancel an item |
| `POST` | `/api/v1/work-items/{id}/fail` | fail with error message |

Errors are returned as `{"error": "..."}` with a 4xx status and thrown as
`WorkerError` (carrying the HTTP `status`).

## Types

`WorkItem` carries `id`, `instance_id`, `activity_id`, `state`, `result`,
`checkpoint_ref`, `effect_cursor`, `attempts`, `max_attempts`, `last_error`, and
the nested `approval` (`state`, `pending_stage`, `final_args`, `reason`,
`requested_by`, `requested_at`, `decided_by`, `decided_at`, `comment`,
`checkpoint_ref`, `effect_cursor`).

`WorkResult` is the worker callback outcome, with status `success`, `failure`,
or `approval`:

```ts
import { success, failure, requestApproval } from "./worker.ts";

success({ ok: true }, "ckpt-1", "cur-1");
failure("boom");
requestApproval({ actor: "agent", reason: "over limit", final_args: { amount: 10 } });
```

`ApprovalRequest` has `actor`, `reason`, `final_args`, `checkpoint_ref`, and
`effect_cursor`.

## Client

```ts
import { Client } from "./worker.ts";

const client = new Client("http://127.0.0.1:8080");

const items = await client.poll();                  // claimable items globally
const filtered = await client.poll("inst-1");       // filter by instance
const item = await client.getWorkItem("wi1");       // includes approval/result

const { token } = await client.claim("wi1", "30s");
await client.heartbeat("wi1", token);

// Complete with a typed result and durable-execution references.
await client.complete("wi1", token, {
  result: { ok: true },
  checkpoint_ref: "ckpt-1",
  effect_cursor: "cur-1",
});

// Token-only completion remains compatible.
await client.complete("wi1", token);

await client.requestApproval("wi1", token, { actor: "agent", reason: "review" });
await client.approve("wi1", "human", "ok", { approved: true });
await client.reject("wi2", "human", "no");
await client.cancel("wi3");
await client.fail("wi4", token, "boom");
```

`complete` only sends non-empty optional fields (`result`, `checkpoint_ref`,
`effect_cursor`), so a token-only body keeps working with older servers.

## Worker loop

`Worker` wraps polling, claiming, executing and settling into one pass. The
callback **must return a `WorkResult`** (breaking change from the old
`{ status }` return value):

```ts
import { Client, Worker, success, failure, requestApproval } from "./worker.ts";

const client = new Client("http://127.0.0.1:8080");
const w = new Worker(client, async (item) => {
  try {
    if (item.activity_id === "dangerous") {
      return requestApproval({ actor: "agent", reason: "needs review" });
    }
    return success({ ok: true }, "ckpt-1");
  } catch (err) {
    return failure(String(err));
  }
}, "30s");

while (true) {
  if (await w.processOnce() === 0) {  // nothing ready, back off
    await new Promise((r) => setTimeout(r, 2000));
  }
}
```

- `success`: complete with `result`, `checkpoint_ref`, `effect_cursor`.
- `failure` (or unknown status): fail with `error` (default `"failure"`).
- `approval`: request approval; a missing `approval` request fails the item.
  Missing checkpoint/effect references on the `ApprovalRequest` are filled from
  the `WorkResult`.
- Items whose `claim` conflicts (already claimed, 409) are skipped.
- A stale/expired token on settle is silently ignored (`WorkerError`).
- Filter by instance with `await w.processOnce(instanceId)`.

## Tests

Uses Node's built-in test runner with a mock `fetch` (no framework):

```sh
node --test --experimental-strip-types sdk/typescript/worker.test.ts
```

Node 22.6+ / 24 strip-only type stripping runs `.ts` directly. The source avoids
parameter properties, enums, and other non-strippable syntax, so
`--experimental-transform-types` is not required.

## TypeScript usage note

The file uses the `.ts` extension with no build step. Node 18+ cannot run
`.ts` directly; either transpile (e.g. `tsc`, `esbuild`, `deno`) or rename to
`.js` and strip types. The SDK itself has zero dependencies in either case.
