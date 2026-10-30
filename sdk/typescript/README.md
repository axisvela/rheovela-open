> [English](README.en.md) · 中文

# TypeScript Worker SDK — `sdk/typescript`

依赖为零的 core Worker HTTP API 客户端；`events.ts` 使用 Node 22+ 内置 WebSocket 提供持久化事件订阅与离线队列。由 `rheo serve` 暴露（路由见下，契约见
`api/README.md` 与 `api/openapi.yaml`）。仅使用 Node 18+ 内置的 `fetch`，无运行时
依赖、无构建步骤，直接引入源文件即可。

## Worker API 路由

| 方法 | 路径 | 用途 |
|------|------|------|
| `GET` | `/api/v1/work-items[?instance=<id>]` | 列出工作项（全局轮询 = 可领取项） |
| `GET` | `/api/v1/work-items/{id}` | 获取单个工作项（含 result/approval） |
| `POST` | `/api/v1/work-items/{id}/claim` | 领取（返回 fencing token） |
| `POST` | `/api/v1/work-items/{id}/heartbeat` | 续租 |
| `POST` | `/api/v1/work-items/{id}/complete` | 完成，可携带 result / 游标 |
| `POST` | `/api/v1/work-items/{id}/request-approval` | 请求人工审批 |
| `POST` | `/api/v1/work-items/{id}/approve` | 批准待审批项 |
| `POST` | `/api/v1/work-items/{id}/reject` | 拒绝待审批项 |
| `POST` | `/api/v1/work-items/{id}/cancel` | 取消工作项 |
| `POST` | `/api/v1/work-items/{id}/fail` | 标记失败并记录错误 |

错误以 `{"error": "..."}` + 4xx 返回，并抛出 `WorkerError`（携带 HTTP `status`）。

## 类型

`WorkItem` 包含 `id`、`instance_id`、`activity_id`、`state`、`result`、
`checkpoint_ref`、`effect_cursor`、`attempts`、`max_attempts`、`last_error`，
以及嵌套的 `approval`（`state`、`pending_stage`、`final_args`、`reason`、
`requested_by`、`requested_at`、`decided_by`、`decided_at`、`comment`、
`checkpoint_ref`、`effect_cursor`）。

`WorkResult` 表示 worker 回调的结果，状态为 `success`、`failure` 或 `approval`：

```ts
import { success, failure, requestApproval } from "./worker.ts";

success({ ok: true }, "ckpt-1", "cur-1");
failure("boom");
requestApproval({ actor: "agent", reason: "over limit", final_args: { amount: 10 } });
```

`ApprovalRequest` 包含 `actor`、`reason`、`final_args`、`checkpoint_ref`、
`effect_cursor`。

## 客户端

```ts
import { Client } from "./worker.ts";

const client = new Client("http://127.0.0.1:8080");

const items = await client.poll();                  // 全局可领取项
const filtered = await client.poll("inst-1");       // 按实例过滤
const item = await client.getWorkItem("wi1");       // 含 approval/result

const { token } = await client.claim("wi1", "30s");
await client.heartbeat("wi1", token);

// 携带结构化结果与持久化执行游标完成。
await client.complete("wi1", token, {
  result: { ok: true },
  checkpoint_ref: "ckpt-1",
  effect_cursor: "cur-1",
});

// 仅 token 的完成方式保持兼容。
await client.complete("wi1", token);

await client.requestApproval("wi1", token, { actor: "agent", reason: "review" });
await client.approve("wi1", "human", "ok", { approved: true });
await client.reject("wi2", "human", "no");
await client.cancel("wi3");
await client.fail("wi4", token, "boom");
```

`complete` 只发送非空的可选字段（`result`、`checkpoint_ref`、`effect_cursor`），
因此仅 token 的请求体仍可兼容旧服务端。

## Worker 循环

`Worker` 把「轮询 + 领取 + 执行 + 结算」串成一次处理。回调**必须返回
`WorkResult`**（相对旧版 `{ status }` 返回值是破坏性变更）：

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
  if (await w.processOnce() === 0) {  // 无活可做，退避
    await new Promise((r) => setTimeout(r, 2000));
  }
}
```

- `success`：携带 `result`、`checkpoint_ref`、`effect_cursor` 完成。
- `failure`（或未知状态）：以 `error` 失败（缺省 `"failure"`）。
- `approval`：请求审批；若 `approval` 为空则标记失败。`ApprovalRequest` 中缺失的
  checkpoint/effect 引用会从 `WorkResult` 补齐。
- 领取冲突（已被领取，409）自动跳过。
- 结算时 token 过期/失效会被静默忽略（`WorkerError`）。
- 用 `w.processOnce(instanceId)` 按实例过滤。

## 测试

使用 Node 内置测试运行器和 mock `fetch`（无测试框架）：

```sh
node --test --experimental-strip-types sdk/typescript/worker.test.ts
```

Node 22.6+ / 24 的 strip-only 类型擦除可直接运行 `.ts`。源码刻意避免
parameter properties、enum 等不可擦除语法，无需 `--experimental-transform-types`。

## TypeScript 使用说明

源文件使用 `.ts` 扩展名且无构建步骤。Node 18+ 不能直接运行 `.ts`；可转译
（`tsc`、`esbuild`、`deno`）或重命名为 `.js` 后擦除类型。SDK 本身零依赖。

## 事件客户端测试

```sh
node --test --experimental-strip-types events.test.ts
```n