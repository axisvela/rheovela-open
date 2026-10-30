> [English](README.en.md) · 中文

# Java Worker SDK — `sdk/java`

仅依赖 JDK 标准库的 core Worker HTTP API 客户端，由 `rheo serve` 暴露（路由见下，
契约见 `api/README.md`）。使用 `java.net.http.HttpClient`，无外部依赖，JDK 21
（源码兼容 JDK 11+）。

## Worker API 路由

| 方法 | 路径 | 用途 |
|------|------|------|
| `GET` | `/api/v1/work-items[?instance=<id>]` | 列出工作项（全局轮询 = 可领取项） |
| `GET` | `/api/v1/work-items/{id}` | 获取单个工作项（含 result/approval） |
| `POST` | `/api/v1/work-items/{id}/claim` | 领取（返回 fencing token） |
| `POST` | `/api/v1/work-items/{id}/complete` | 完成，可携带 result / 游标 |
| `POST` | `/api/v1/work-items/{id}/request-approval` | 请求人工审批 |
| `POST` | `/api/v1/work-items/{id}/approve` | 批准待审批项 |
| `POST` | `/api/v1/work-items/{id}/reject` | 拒绝待审批项 |
| `POST` | `/api/v1/work-items/{id}/cancel` | 取消工作项 |
| `POST` | `/api/v1/work-items/{id}/fail` | 标记失败并记录错误 |

错误以 `{"error": "..."}` + 非 2xx 返回，并抛出
`Worker.WorkerException(status, body)`。

## 类型

`WorkItem` 解析 `id`、`instance_id`、`activity_id`、`state`、`result`、
`checkpoint_ref`、`effect_cursor`、`attempts`、`max_attempts`、`last_error`，
以及嵌套的 `approval`（`state`、`pending_stage`、`final_args`、`reason`、
`requested_by`、`requested_at`、`decided_by`、`decided_at`、`comment`、
`checkpoint_ref`、`effect_cursor`）。`result` 与 `final_args` 是**原始 JSON
字符串**，请求中原样嵌入，不做二次编码。

`Result` 表示 worker 回调的结果，状态为 `success`、`failure` 或 `approval`：

```java
Worker.Result.ok();                              // 兼容旧版，等价于无 result 的 success
Worker.Result.success("{\"ok\":true}", "ckpt-1", "cur-1");
Worker.Result.fail("boom");
Worker.Result.approval(new Worker.ApprovalRequest(
        "agent", "over limit", "{\"amount\":10}", "", ""));
```

`ApprovalRequest` 包含 `actor`、`reason`、`final_args`、`checkpoint_ref`、
`effect_cursor`。

## 客户端

```java
import java.util.List;
import worker.Worker;

Worker w = new Worker("http://127.0.0.1:8080");

List<Worker.WorkItem> items = w.poll(null);     // 全局可领取项
List<Worker.WorkItem> filtered = w.poll("inst-1");
Worker.WorkItem item = w.get("wi1");            // 含 approval/result

String token = w.claim("wi1", "30s");

// 携带原始 JSON 结果与持久化执行游标完成。
w.complete("wi1", token, "{\"ok\":true}", "ckpt-1", "cur-1");

// 仅 token 的完成方式保持兼容。
w.complete("wi1", token);

w.requestApproval("wi1", token, new Worker.ApprovalRequest(
        "agent", "review", null, "ckpt-1", "cur-1"));
w.approve("wi1", "human", "ok", "{\"approved\":true}");
w.reject("wi2", "human", "no");
w.cancel("wi3");
w.fail("wi4", token, "boom");
```

`complete` 只发送非空的可选字段（`result`、`checkpoint_ref`、`effect_cursor`），
因此仅 token 的请求体仍可兼容旧服务端。

## Worker 循环

`processOnce` 把「轮询 + 领取 + 执行 + 结算」串成一次处理。回调返回 `Result`：

```java
int processed = w.processOnce((item, token) -> {
    try {
        if ("dangerous".equals(item.activity_id())) {
            return Worker.Result.approval(new Worker.ApprovalRequest(
                    "agent", "needs review", null, null, null));
        }
        return Worker.Result.success("{\"ok\":true}", "ckpt-1", "cur-1");
    } catch (Exception e) {
        return Worker.Result.fail(e.getMessage());
    }
}, "30s");
```

- `success`：携带 `result`、`checkpoint_ref`、`effect_cursor` 完成。
- `failure`（或未知状态）：以 `error` 失败（缺省 `"failure"`）。
- `approval`：请求审批；若 `approval` 为空则标记失败。`ApprovalRequest` 中缺失的
  checkpoint/effect 引用会从 `Result` 补齐。
- 领取冲突（已被领取）自动跳过。
- 结算时 token 过期/失效会被静默忽略（`WorkerException`）。

## 构建与运行

```powershell
javac -d out sdk/java/worker/Worker.java sdk/java/example/Example.java
java -cp out example.Example
```

预期输出：

```
ok
```

`Example` 在进程内启动 fake Worker API，覆盖 success-with-result、
request-approval、approve/reject/cancel 与单条 `get`，并把实际请求端点与请求体
和预期序列比较；不一致时抛错（CI 直接运行）。

## 事件客户端

```sh
javac -d out sdk/java/worker/Worker.java sdk/java/worker/Events.java sdk/java/example/EventsExample.java
java -cp out example.EventsExample
```n