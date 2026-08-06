// Package worker 提供面向 worker 的 SDK：领取、心跳、完成/失败。
package worker

import (
	"context"
	"time"
)

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

type WorkFn func(ctx context.Context, item WorkItem) WorkResult

// WorkStore 是 durable runtime（rheovela core）必须实现的端口。
type WorkStore interface {
	PollReady() ([]WorkItem, error)
	Claim(id string, leaseDuration time.Duration) (token string, err error)
	Heartbeat(id, token string, leaseDuration time.Duration) error
	Complete(id, token string) error
	Fail(id, token, errMsg string) error
}

type Worker struct {
	store WorkStore
	lease time.Duration
	fn    WorkFn
	logf  func(format string, args ...any)
}

type Option func(*Worker)

func WithLogger(logf func(format string, args ...any)) Option {
	return func(w *Worker) { w.logf = logf }
}

func New(store WorkStore, fn WorkFn, lease time.Duration, opts ...Option) *Worker {
	w := &Worker{store: store, fn: fn, lease: lease, logf: func(string, ...any) {}}
	for _, o := range opts {
		o(w)
	}
	return w
}

// ProcessOnce 轮询一次并处理所有已领取就绪的 work item；返回处理数量。
func (w *Worker) ProcessOnce(ctx context.Context) (int, error) {
	items, err := w.store.PollReady()
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, it := range items {
		if ctx.Err() != nil {
			return processed, ctx.Err()
		}
		token, err := w.store.Claim(it.ID, w.lease)
		if err != nil {
			w.logf("claim %s: %v", it.ID, err) // 并发失败/不可领取 → 跳过
			continue
		}
		w.logf("claim %s token=%s", it.ID, token)
		res := w.fn(ctx, it)
		if res.Status == "success" {
			if err := w.store.Complete(it.ID, token); err != nil {
				w.logf("complete %s: %v", it.ID, err)
				continue
			}
		} else {
			if err := w.store.Fail(it.ID, token, res.Error); err != nil {
				w.logf("fail %s: %v", it.ID, err)
				continue
			}
		}
		processed++
	}
	return processed, nil
}
