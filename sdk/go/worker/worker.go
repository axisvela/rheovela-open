// Package worker 提供面向 worker 的 SDK：领取、心跳、完成/失败。
package worker

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"
)

type Approval struct {
	State         string          `json:"state"`
	PendingStage  string          `json:"pending_stage,omitempty"`
	FinalArgs     json.RawMessage `json:"final_args,omitempty"`
	Reason        string          `json:"reason,omitempty"`
	RequestedBy   string          `json:"requested_by,omitempty"`
	RequestedAt   time.Time       `json:"requested_at,omitempty"`
	DecidedBy     string          `json:"decided_by,omitempty"`
	DecidedAt     *time.Time      `json:"decided_at,omitempty"`
	Comment       string          `json:"comment,omitempty"`
	CheckpointRef string          `json:"checkpoint_ref,omitempty"`
	EffectCursor  string          `json:"effect_cursor,omitempty"`
}

type ApprovalRequest struct {
	Actor         string
	Reason        string
	FinalArgs     json.RawMessage
	CheckpointRef string
	EffectCursor  string
}

type WorkItem struct {
	ID                  string
	InstanceID          string
	ActivityID          string
	State               string
	Assignee            string
	Kind                string
	Payload             json.RawMessage
	RequiredEvidence    []string
	Evidence            json.RawMessage
	Budget              int64
	Spent               int64
	ToolScope           []string
	CredentialRef       string
	ApprovalRoles       []string
	CompletionMode      string
	RequiredApprovals   int
	RequiredCompletions int
	Deadline            *time.Time
	Escalation          string
	Result              json.RawMessage
	CheckpointRef       string
	EffectCursor        string
	Approval            *Approval
}

type WorkResult struct {
	Status        string // "success" | "failure" | "approval"
	Error         string
	Outcome       string
	Actor         string
	Result        json.RawMessage
	Evidence      json.RawMessage
	Cost          int64
	CheckpointRef string
	EffectCursor  string
	Approval      *ApprovalRequest
}

// ClaimOptions carries optional claim-time guards.
type ClaimOptions struct {
	Actor      string
	Tools      []string
	Credential string
}

type WorkFn func(ctx context.Context, item WorkItem) WorkResult

// ResultStore is an optional WorkStore extension: when implemented, successful
// results are committed together with their result payload and cursors.
type ResultStore interface {
	CompleteWithResult(id, token string, res WorkResult) error
}

// ClaimGuardStore is an optional WorkStore extension: when implemented, the
// worker forwards actor/tools/credential guards during claims.
type ClaimGuardStore interface {
	ClaimWithOptions(id string, leaseDuration time.Duration, opts ClaimOptions) (token string, err error)
}

// ApprovalStore is an optional WorkStore extension: when implemented, a worker
// can request human approval instead of completing or failing.
type ApprovalStore interface {
	RequestApproval(id, token string, req ApprovalRequest) error
}

// WorkStore 是 durable runtime（rheovela core）必须实现的端口。
type WorkStore interface {
	PollReady() ([]WorkItem, error)
	Claim(id string, leaseDuration time.Duration) (token string, err error)
	Heartbeat(id, token string, leaseDuration time.Duration) error
	Complete(id, token string) error
	Fail(id, token, errMsg string) error
}

type Worker struct {
	store             WorkStore
	lease             time.Duration
	fn                WorkFn
	logf              func(format string, args ...any)
	heartbeatInterval time.Duration
	claimOptions      ClaimOptions
}

type Option func(*Worker)

func WithLogger(logf func(format string, args ...any)) Option {
	return func(w *Worker) { w.logf = logf }
}

func WithHeartbeatInterval(d time.Duration) Option {
	return func(w *Worker) { w.heartbeatInterval = d }
}

// WithClaimOptions forwards actor/tools/credential guards to ClaimGuardStore
// implementations; plain WorkStore implementations ignore them.
func WithClaimOptions(opts ClaimOptions) Option {
	return func(w *Worker) { w.claimOptions = opts }
}

func New(store WorkStore, fn WorkFn, lease time.Duration, opts ...Option) *Worker {
	w := &Worker{
		store:             store,
		fn:                fn,
		lease:             lease,
		logf:              func(string, ...any) {},
		heartbeatInterval: lease / 2,
	}
	if w.heartbeatInterval < time.Second {
		w.heartbeatInterval = time.Second
	}
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
		var token string
		if guarded, ok := w.store.(ClaimGuardStore); ok {
			token, err = guarded.ClaimWithOptions(it.ID, w.lease, w.claimOptions)
		} else {
			token, err = w.store.Claim(it.ID, w.lease)
		}
		if err != nil {
			w.logf("claim %s: %v", it.ID, err) // 并发失败/不可领取 → 跳过
			continue
		}
		w.logf("claim %s token=%s", it.ID, token)
		res := w.runWithHeartbeat(ctx, it, token)
		switch res.Status {
		case "success":
			var err error
			if rs, ok := w.store.(ResultStore); ok {
				err = rs.CompleteWithResult(it.ID, token, res)
			} else {
				err = w.store.Complete(it.ID, token)
			}
			if err != nil {
				w.logf("complete %s: %v", it.ID, err)
				continue
			}
		case "approval":
			as, ok := w.store.(ApprovalStore)
			if !ok {
				w.logf("request approval %s: store does not implement ApprovalStore", it.ID)
				if err := w.store.Fail(it.ID, token, "worker store does not support approval"); err != nil {
					w.logf("fail %s: %v", it.ID, err)
					continue
				}
				processed++
				continue
			}
			if res.Approval == nil {
				if err := w.store.Fail(it.ID, token, "approval result is missing approval request"); err != nil {
					w.logf("fail %s: %v", it.ID, err)
					continue
				}
				processed++
				continue
			}
			req := *res.Approval
			if req.CheckpointRef == "" {
				req.CheckpointRef = res.CheckpointRef
			}
			if req.EffectCursor == "" {
				req.EffectCursor = res.EffectCursor
			}
			if err := as.RequestApproval(it.ID, token, req); err != nil {
				w.logf("request approval %s: %v", it.ID, err)
				if ferr := w.store.Fail(it.ID, token, err.Error()); ferr != nil {
					w.logf("fail %s: %v", it.ID, ferr)
					continue
				}
				processed++
				continue
			}
		default:
			if err := w.store.Fail(it.ID, token, res.Error); err != nil {
				w.logf("fail %s: %v", it.ID, err)
				continue
			}
		}
		processed++
	}
	return processed, nil
}

// runWithHeartbeat 运行 fn，并在后台按 heartbeatInterval 定期发送心跳以续租。
// 若心跳失败（租约过期/stale token），则取消 fn 的上下文并返回 "lease lost" 失败结果。
// 心跳 goroutine 在 fn 返回后通过 stop 通道停止，不会泄漏。
func (w *Worker) runWithHeartbeat(ctx context.Context, item WorkItem, token string) WorkResult {
	hctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := make(chan struct{})
	done := make(chan struct{})
	var leaseLost atomic.Bool
	go func() {
		defer close(done)
		t := time.NewTicker(w.heartbeatInterval)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if err := w.store.Heartbeat(item.ID, token, w.lease); err != nil {
					w.logf("heartbeat %s: %v", item.ID, err)
					leaseLost.Store(true)
					cancel()
					return
				}
			}
		}
	}()

	res := w.fn(hctx, item)
	close(stop)
	if res.Actor == "" {
		res.Actor = w.claimOptions.Actor
	}
	<-done
	if leaseLost.Load() {
		return WorkResult{Status: "failure", Error: "lease lost"}
	}
	return res
}
