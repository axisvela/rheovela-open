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
