> [English](README.en.md) · 中文

# Python Worker SDK — `sdk/python`

仅依赖标准库（`urllib` / `json`）的 core Worker HTTP API 客户端；`events.py` 额外使用 `websockets`（`pip install -r requirements-events.txt`）提供持久化 WebSocket 事件订阅。由 `rheo serve`
暴露（路由见下，契约见 `api/README.md`）。无第三方依赖，支持 Python 3.9+。

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

错误以 `{"error": "..."}` + 4xx 返回，并抛出 `worker.WorkerError`。

## 类型

`WorkItem` 解析 `id`、`instance_id`、`activity_id`、`state`、`result`、
`checkpoint_ref`、`effect_cursor`、`attempts`、`max_attempts`、`last_error`，
以及嵌套的 `approval`（`state`、`pending_stage`、`final_args`、`reason`、
`requested_by`、`requested_at`、`decided_by`、`decided_at`、`comment`、
`checkpoint_ref`、`effect_cursor`）。

```python
import worker

item = worker.WorkItem(
    id="wi1", instance_id="i1", activity_id="a1", state="ready",
)
```

`WorkResult` 表示 worker 回调的结果，使用以下辅助构造：

```python
worker.WorkResult.success(result=None, checkpoint_ref="", effect_cursor="")
worker.WorkResult.failure("boom")
worker.WorkResult.request_approval(
    worker.ApprovalRequest(actor="agent", reason="over limit")
)
```

`ApprovalRequest` 包含 `actor`、`reason`、`final_args`、`checkpoint_ref`、
`effect_cursor`。

## 客户端

```python
import worker

client = worker.Client("http://127.0.0.1:8080")

items = client.poll()                          # 全局可领取项
items = client.poll(instance_id="inst-1")      # 按实例过滤
item = client.get_work_item("wi1")             # 含 approval/result

claim = client.claim("wi1", lease="30s")
token = claim["token"]
client.heartbeat("wi1", token)

# 携带结构化结果与持久化执行游标完成。
client.complete("wi1", token, result={"ok": True},
                checkpoint_ref="ckpt-1", effect_cursor="cur-1")

# 仅 token 的完成方式保持兼容。
client.complete("wi1", token)

client.request_approval("wi1", token,
                        worker.ApprovalRequest(actor="agent", reason="review"))
client.approve("wi1", "human", comment="ok", final_args={"approved": True})
client.reject("wi2", "human", "no")
client.cancel("wi3")
client.fail("wi4", token, "boom")
```

`complete` 只发送非空的可选字段（`result`、`checkpoint_ref`、`effect_cursor`），
因此仅 token 的请求体仍可兼容旧服务端。

## Worker 循环

`Worker` 把「轮询 + 领取 + 执行 + 结算」串成一次处理。回调**必须返回
`WorkResult`**（相对旧版元组返回值是破坏性变更）：

```python
import worker

def fn(item):
    try:
        if item.activity_id == "dangerous":
            return worker.WorkResult.request_approval(
                worker.ApprovalRequest(actor="agent", reason="needs review")
            )
        return worker.WorkResult.success(
            result={"ok": True}, checkpoint_ref="ckpt-1",
        )
    except Exception as exc:
        return worker.WorkResult.failure(str(exc))

client = worker.Client("http://127.0.0.1:8080")
w = worker.Worker(client, fn, lease="30s")

while True:
    if w.process_once() == 0:   # 无活可做，退避
        import time
        time.sleep(2)
```

- `success`：携带 `result`、`checkpoint_ref`、`effect_cursor` 完成。
- `failure`（或未知状态）：以 `error` 失败（缺省 `"failure"`）。
- `approval`：请求审批；若 `approval` 为空则标记失败。`ApprovalRequest` 中缺失的
  checkpoint/effect 引用会从 `WorkResult` 补齐。
- 领取冲突（已被领取，409）自动跳过。
- 结算时 token 过期/失效会被静默忽略（`WorkerError`）。
- 用 `w.process_once(instance_id)` 按实例过滤。

## 测试

```sh
python -m py_compile sdk/python/worker.py
python -m unittest discover -s sdk/python -v
```
