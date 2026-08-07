# Go Worker SDK — `sdk/worker`

面向 worker 的 Go SDK（Apache-2.0）：**领取 → 心跳 → 完成/失败** 的持久化任务
消费循环。默认处理 claim / lease / fencing token / 幂等完成与结构化失败；worker
无需关心 SQLite、事务或事件流细节。

## WorkStore — durable runtime 端口

`WorkStore` 是 worker 与 durable runtime 之间的**端口**。worker 只依赖这个接口，
不依赖具体存储实现：

```go
type WorkStore interface {
    PollReady() ([]WorkItem, error)                                   // 取回所有 ready 工作项
    Claim(id string, lease time.Duration) (token string, err error)   // 领取，返回 fencing token
    Heartbeat(id, token string, lease time.Duration) error            // 续租
    Complete(id, token string) error                                  // 幂等完成
    Fail(id, token, errMsg string) error                              // 结构化失败
}
```

`Claim` 返回的 **fencing token** 是并发护栏：只有持 token 的 worker 能
`Complete`/`Fail` 该项，避免「租约已过期、被其他 worker 重新领取」后旧 worker 仍写入。

`WorkItem` / `WorkResult`：

```go
type WorkItem struct {
    ID         string
    InstanceID string
    ActivityID string
    State      string
}

type WorkResult struct {
    Status  string // "success" | "failure"
    Error   string
    Outcome string
}
```

## 如何实现 WorkStore

### 方式 A：嵌入 core 的 `runtime.WorkerBridge`（推荐，本地/edge）

core（AGPL）在公共 `runtime` 包中提供了 `WorkerBridge`，它通过 core 的 broker
（`work_items` 表 + claim/lease/fencing 语义）实现 `WorkStore`：

```go
import (
    "github.com/axisrobo/rheovela/runtime"   // core 公共 API
    worker "github.com/axisrobo/rheovela-open/sdk/worker"
)
```

构造 `WorkerBridge` 需要 core 的 `Store`：

```go
db, err := runtime.OpenDB(dbPath)      // 打开 SQLite
st := runtime.NewStore(db)
bridge := runtime.NewWorkerBridge(st)  // implements worker.WorkStore
```

依赖方向：`worker` SDK（Apache）本身零依赖 core；`WorkerBridge` 位于 core 侧，把
core 的 broker 语义桥接到 SDK 的 `WorkStore` 端口。

### 方式 B：自行实现 `WorkStore`

在自己的后端上实现上述五个方法即可（例如未来的 Postgres adapter、远程网关、
企业版 broker）。`RunStoreContract`（core `internal/store/contract_test.go`）是
存储适配器必须通过的语义一致性闸门。

## Worker / WorkFn / ProcessOnce

`Worker` 把「轮询 + 领取 + 执行 + 结算」串成一个循环：

```go
w := worker.New(bridge, fn, lease, opts...)
n, err := w.ProcessOnce(ctx)   // 轮询一次并处理所有 ready 项，返回处理数量
```

- **`fn`**：`WorkFn = func(ctx context.Context, item WorkItem) WorkResult`。
- **`lease`**：每项的租约时长（`time.Duration`）。
- **`ProcessOnce`**：`PollReady` → 逐项 `Claim`（并发失败/不可领取则跳过）→ 执行
  `fn` → `success` 则 `Complete`，否则 `Fail(res.Error)`；`ctx` 取消时立即停止并返回
  `context.Canceled`。
- **`WithLogger(logf)`**：可选日志（`logf(format, args...)`），打印 claim/complete/fail。

## 最小示例

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/axisrobo/rheovela/runtime"
    worker "github.com/axisrobo/rheovela-open/sdk/worker"
)

func main() {
    db, err := runtime.OpenDB("~/.proc/proc.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()
    st := runtime.NewStore(db)
    bridge := runtime.NewWorkerBridge(st)

    // fn：接到工作项就执行，返回 success/failure
    fn := func(ctx context.Context, item worker.WorkItem) worker.WorkResult {
        fmt.Printf("working on %s (activity=%s)\n", item.ID, item.ActivityID)
        return worker.WorkResult{Status: "success", Outcome: "ok"}
    }

    w := worker.New(bridge, fn, 30*time.Second,
        worker.WithLogger(func(format string, args ...any) { fmt.Printf(format+"\n", args...) }))

    for {
        n, err := w.ProcessOnce(context.Background())
        if err != nil {
            fmt.Printf("process: %v\n", err)
        }
        if n == 0 {
            time.Sleep(2 * time.Second) // 无活可做，稍后重试
        }
    }
}
```

## 与 core 的配合

- core `rheo serve` 暴露 HTTP Ops API（`api/README.md`）；`work_items` 由
  scheduler / broker 产出，worker SDK 消费。
- 事件流（`StepEntered` / `StepCompleted` 等）与 effect ledger 由 core 维护；
  worker 通过 `WorkStore` 结算即可，无需直写事件表。
- 现有测试见 `worker_test.go`（完成/失败/并发跳过/ctx 取消）。
