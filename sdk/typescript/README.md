# TypeScript Worker SDK — `sdk/typescript`

Dependency-free HTTP client for the core Worker API exposed by `rheo serve`
(see `api/README.md` and `api/openapi.yaml`). Uses only the built-in `fetch`
globally available since Node 18. No runtime dependencies, no build step —
import the file directly.

## Worker API routes

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/api/v1/work-items?instance=<id>` | list work items |
| `POST` | `/api/v1/work-items/{id}/claim` | claim (returns fencing token) |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | renew lease |
| `POST` | `/api/v1/work-items/{id}/complete` | complete (token-guarded) |
| `POST` | `/api/v1/work-items/{id}/fail` | fail with error message |

Errors are returned as `{"error": "..."}` with a 4xx status and thrown as
`WorkerError` (carrying the HTTP `status`).

## Client

```ts
import { Client } from "./worker.ts"; // or "./worker.js" once transpiled

const client = new Client("http://127.0.0.1:8080");

const items = await client.poll();                    // WorkItem[]
const { token } = await client.claim(items[0].id, "30s");
await client.heartbeat(items[0].id, token);
await client.complete(items[0].id, token);            // or fail(id, token, "boom")
```

## Worker loop

`Worker` wraps polling, claiming, executing and settling into one pass:

```ts
import { Client, Worker } from "./worker.ts";

const client = new Client("http://127.0.0.1:8080");
const w = new Worker(client, async (item) => {
  // do the work; return { status: "success" } or { status: "failure", error }
  return { status: "success" };
}, "30s");

while (true) {
  if (await w.processOnce() === 0) {  // nothing ready, back off
    await new Promise((r) => setTimeout(r, 2000));
  }
}
```

- Items whose `claim` conflicts (already claimed, 409) are skipped.
- A stale/expired token on settle is silently ignored (`WorkerError`).
- Filter by instance with `await w.processOnce(instanceId)`.

## Tests

Uses Node's built-in test runner with a mock `fetch` (no framework):

```sh
node --experimental-transform-types --test sdk/typescript/worker.test.ts
```

Node's strip-only type stripping does not support TS parameter properties, so
run with `--experimental-transform-types` (Node 22.7+). On a Node version
without that flag, transpile with `tsc`/`esbuild` first or compile the tests
to JS.

## TypeScript usage note

The file uses the `.ts` extension with no build step. Node 18+ cannot run
`.ts` directly; either transpile (e.g. `tsc`, `esbuild`, `deno`) or rename to
`.js` and strip types. The SDK itself has zero dependencies in either case.
