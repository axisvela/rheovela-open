> [English](README.en.md) · 中文

# Rust Worker SDK —— `sdk/rust`

纯标准库实现的 core Worker HTTP API 客户端，由 `rheo serve` 暴露（路由见下，
契约见 `api/README.md`）。仅使用 `std::net::TcpStream` 与 `std::io`，无外部
crate，可完全离线构建（`cargo build --offline`）。

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

任何非 2xx 响应都会返回 `WorkerError { status, message }`。

## 类型

`WorkItem` 解析 `id`、`instance_id`、`activity_id`、`state`、`result`、
`checkpoint_ref`、`effect_cursor`、`attempts`、`max_attempts`、`last_error`，
以及嵌套的 `approval`（`state`、`pending_stage`、`final_args`、`reason`、
`requested_by`、`requested_at`、`decided_by`、`decided_at`、`comment`、
`checkpoint_ref`、`effect_cursor`）。`result` 与 `final_args` 是**原始 JSON
字符串**，请求中原样嵌入，不做二次编码。

`WorkResult` 表示 worker 回调的结果，`WorkStatus` 为 `Success`、`Failure` 或
`Approval`：

```rust
use rheovela_worker::{ApprovalRequest, WorkResult};

WorkResult::success(); // 无 result 的成功完成
WorkResult::success_with(
    Some(r#"{"ok":true}"#.to_string()),
    Some("ckpt-1".to_string()),
    Some("cur-1".to_string()),
);
WorkResult::failure("boom");
WorkResult::approval(ApprovalRequest {
    actor: "agent".to_string(),
    reason: "over limit".to_string(),
    final_args: Some(r#"{"amount":10}"#.to_string()),
    checkpoint_ref: String::new(),
    effect_cursor: String::new(),
});
```

`ApprovalRequest` 包含 `actor`、`reason`、`final_args`、`checkpoint_ref`、
`effect_cursor`。

## 客户端

```rust
use rheovela_worker::{ApprovalRequest, Client, Worker};

let w = Worker::new(Client::new("http://127.0.0.1:8080"));

let items = w.poll(None)?;              // 全局可领取项
let filtered = w.poll(Some("inst-1"))?;
let item = w.get("wi1")?;               // 含 approval/result

let token = w.claim("wi1", "30s")?;

// 携带原始 JSON 结果与持久化执行游标完成。
w.complete_with_result(
    "wi1", &token,
    Some(r#"{"ok":true}"#),
    Some("ckpt-1"),
    Some("cur-1"),
)?;

// 仅 token 的完成方式保持兼容。
w.complete("wi1", &token)?;

w.request_approval("wi1", &token, &ApprovalRequest {
    actor: "agent".to_string(),
    reason: "review".to_string(),
    final_args: None,
    checkpoint_ref: "ckpt-1".to_string(),
    effect_cursor: "cur-1".to_string(),
})?;
w.approve("wi1", "human", "ok", Some(r#"{"approved":true}"#))?;
w.reject("wi2", "human", "no")?;
w.cancel("wi3")?;
w.fail("wi4", &token, "boom")?;
```

`complete_with_result` 只发送非空的可选字段（`result`、`checkpoint_ref`、
`effect_cursor`），因此仅 token 的请求体仍可兼容旧服务端。

## Worker 循环

`process_once` 把「轮询 + 领取 + 执行 + 结算」串成一次处理。回调返回
`WorkResult`：

```rust
use rheovela_worker::{ApprovalRequest, Client, WorkResult, Worker};

let w = Worker::new(Client::new("http://127.0.0.1:8080"));

let processed = w.process_once(|item| {
    if item.activity_id == "dangerous" {
        return WorkResult::approval(ApprovalRequest {
            actor: "agent".to_string(),
            reason: "needs review".to_string(),
            final_args: None,
            checkpoint_ref: String::new(),
            effect_cursor: String::new(),
        });
    }
    WorkResult::success_with(
        Some(r#"{"ok":true}"#.to_string()),
        Some("ckpt-1".to_string()),
        Some("cur-1".to_string()),
    )
}, "30s")?;
```

- `success`：携带 `result`、`checkpoint_ref`、`effect_cursor` 完成；三者都不存在
  时退化为仅 token 的 `complete`。
- `failure`：以 `error` 失败（缺省 `"failure"`）。
- `approval`：请求审批；若 `approval` 为空则标记失败。`ApprovalRequest` 中缺失的
  checkpoint/effect 引用会从 `WorkResult` 补齐。
- 领取冲突（已被领取）自动跳过。
- 结算时 token 过期/失效会被静默忽略（`WorkerError`）。

## 构建与运行

```powershell
cd sdk/rust
cargo test --offline
cargo run --offline
```

预期输出：

```
processed=2
ok
```

`main.rs` 在进程内启动 fake Worker API，覆盖 success-with-result、
request-approval、approve/reject/cancel 与单条 `get`，并把实际请求端点与请求体
和预期序列比较；不一致时 panic（CI 直接运行即失败）。
