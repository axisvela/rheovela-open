package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeStore struct {
	mu       sync.Mutex
	items    map[string]WorkItem
	tokens   map[string]string // item id -> token
	leased   map[string]bool
	done     []string
	failed   map[string]string
	claims   int
	claimErr error
	hbCalls  int  // total Heartbeat calls
	hbFailAt int  // 1-based: fail the Nth Heartbeat call; 0 = never fail
	// requeueOnFail models the durable runtime's retry behavior: a failed
	// work item returns to the ready queue and can be claimed again.
	requeueOnFail bool
}

func newFakeStore(items ...WorkItem) *fakeStore {
	s := &fakeStore{
		items:  map[string]WorkItem{},
		tokens: map[string]string{},
		leased: map[string]bool{},
		failed: map[string]string{},
	}
	for _, it := range items {
		s.items[it.ID] = it
	}
	return s
}

func (s *fakeStore) PollReady() ([]WorkItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]WorkItem, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, it)
	}
	return out, nil
}

func (s *fakeStore) Claim(id string, lease time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.claims++
	if s.claimErr != nil {
		return "", s.claimErr
	}
	if s.leased[id] {
		return "", errors.New("already claimed")
	}
	s.leased[id] = true
	token := "tok-" + id
	s.tokens[id] = token
	return token, nil
}

func (s *fakeStore) Heartbeat(id, token string, lease time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hbCalls++
	if s.hbFailAt > 0 && s.hbCalls == s.hbFailAt {
		return errors.New("lease expired")
	}
	if !s.leased[id] {
		return errors.New("not leased")
	}
	return nil
}

func (s *fakeStore) Complete(id, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.leased[id] {
		s.done = append(s.done, id)
	}
	return nil
}

func (s *fakeStore) Fail(id, token, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed[id] = errMsg
	if s.requeueOnFail {
		delete(s.leased, id)
	}
	return nil
}

func TestProcessOnceCompletesWork(t *testing.T) {
	store := newFakeStore(WorkItem{ID: "w1", InstanceID: "i1", ActivityID: "a1", State: "ready"})
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		return WorkResult{Status: "success", Outcome: "ok"}
	}, time.Minute)

	n, err := w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"w1"}, store.done)
	assert.Empty(t, store.failed)
}

func TestProcessOnceFailureRecordsError(t *testing.T) {
	store := newFakeStore(WorkItem{ID: "w2", InstanceID: "i2", ActivityID: "a2", State: "ready"})
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		return WorkResult{Status: "failure", Error: "boom"}
	}, time.Minute)

	n, err := w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Empty(t, store.done)
	assert.Equal(t, map[string]string{"w2": "boom"}, store.failed)
}

func TestProcessOnceSkipsAlreadyClaimed(t *testing.T) {
	store := newFakeStore(WorkItem{ID: "w3", InstanceID: "i3", ActivityID: "a3", State: "ready"})
	store.claimErr = errors.New("another worker won")
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		return WorkResult{Status: "success", Outcome: "should not run"}
	}, time.Minute)

	n, err := w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
	assert.Empty(t, store.done)
	assert.Empty(t, store.failed)
	assert.Equal(t, 1, store.claims)
}

func TestProcessOnceStopsOnCancellation(t *testing.T) {
	store := newFakeStore(
		WorkItem{ID: "w4", InstanceID: "i4", ActivityID: "a4", State: "ready"},
		WorkItem{ID: "w5", InstanceID: "i5", ActivityID: "a5", State: "ready"},
	)
	var calls int
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		calls++
		return WorkResult{Status: "success", Outcome: "ok"}
	}, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	first, err := w.ProcessOnce(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, first)
	assert.Equal(t, 2, calls)

	store.items["w4"] = WorkItem{ID: "w4", InstanceID: "i4", ActivityID: "a4", State: "ready"}
	cancel()
	n, err := w.ProcessOnce(ctx)
	assert.Equal(t, 0, n)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestProcessOnceHeartbeatsDuringWork(t *testing.T) {
	store := newFakeStore(WorkItem{ID: "hb1", InstanceID: "i1", ActivityID: "a1", State: "ready"})
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		select {
		case <-ctx.Done():
			return WorkResult{Status: "failure", Error: "canceled"}
		case <-time.After(150 * time.Millisecond):
		}
		return WorkResult{Status: "success", Outcome: "slow ok"}
	}, time.Second, WithHeartbeatInterval(50*time.Millisecond))

	n, err := w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"hb1"}, store.done)
	store.mu.Lock()
	defer store.mu.Unlock()
	assert.GreaterOrEqual(t, store.hbCalls, 1, "heartbeat should fire during long-running fn")
}

func TestHeartbeatFailureCancelsWork(t *testing.T) {
	store := newFakeStore(WorkItem{ID: "hb2", InstanceID: "i2", ActivityID: "a2", State: "ready"})
	store.hbFailAt = 1
	var canceled bool
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		select {
		case <-ctx.Done():
			canceled = true
			return WorkResult{Status: "success", Outcome: "ctx canceled"}
		case <-time.After(5 * time.Second):
			return WorkResult{Status: "success", Outcome: "timeout"}
		}
	}, time.Second, WithHeartbeatInterval(50*time.Millisecond))

	n, err := w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.True(t, canceled, "fn context should be canceled on heartbeat failure")
	assert.Empty(t, store.done)
	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Equal(t, map[string]string{"hb2": "lease lost"}, store.failed)
}

func TestNoHeartbeatForFastTask(t *testing.T) {
	store := newFakeStore(WorkItem{ID: "fast", InstanceID: "i1", ActivityID: "a1", State: "ready"})
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		return WorkResult{Status: "success", Outcome: "ok"}
	}, time.Second)

	n, err := w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"fast"}, store.done)
	assert.Empty(t, store.failed)
}

// TestProcessOnceRetriesAfterFailureThenCompletes closes the R6 work-item
// loop: a transient failure returns the item to the queue (durable-runtime
// retry), and the next ProcessOnce reclaims and completes it — one effect,
// one completion.
func TestProcessOnceRetriesAfterFailureThenCompletes(t *testing.T) {
	store := newFakeStore(WorkItem{ID: "retry1", InstanceID: "i1", ActivityID: "a1", State: "ready"})
	store.requeueOnFail = true
	var calls int
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		calls++
		if calls == 1 {
			return WorkResult{Status: "failure", Error: "transient"}
		}
		return WorkResult{Status: "success", Outcome: "recovered"}
	}, time.Minute)

	n, err := w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Empty(t, store.done)
	assert.Equal(t, map[string]string{"retry1": "transient"}, store.failed)

	n, err = w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"retry1"}, store.done)
	assert.Equal(t, 2, calls, "work fn should run once per attempt")
}

// TestCancellationDuringWorkRequeuesWithoutDuplicateCompletion cancels the work
// in flight: the attempt fails (does not complete), the item is requeued, and a
// later run with a fresh context completes it exactly once — no duplicate
// completion/effect.
func TestCancellationDuringWorkRequeuesWithoutDuplicateCompletion(t *testing.T) {
	store := newFakeStore(WorkItem{ID: "cancel1", InstanceID: "i1", ActivityID: "a1", State: "ready"})
	store.requeueOnFail = true
	started := make(chan struct{})
	var attempts int
	w := New(store, func(ctx context.Context, item WorkItem) WorkResult {
		attempts++
		if attempts == 1 {
			close(started)
			<-ctx.Done()
			return WorkResult{Status: "failure", Error: "canceled"}
		}
		return WorkResult{Status: "success", Outcome: "done"}
	}, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { <-started; cancel() }()

	_, err := w.ProcessOnce(ctx)
	require.NoError(t, err)
	assert.Empty(t, store.done, "a cancelled attempt must not complete")
	assert.Equal(t, map[string]string{"cancel1": "canceled"}, store.failed)

	n, err := w.ProcessOnce(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []string{"cancel1"}, store.done, "exactly one completion after reclaim")
	assert.Equal(t, 2, attempts)
}
