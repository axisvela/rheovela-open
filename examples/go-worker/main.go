// Command go-worker demonstrates the worker SDK end-to-end against an
// in-memory fake WorkStore.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/axisrobo/rheovela-open/sdk/worker"
)

// fakeStore implements worker.WorkStore in memory. In production the durable
// store is provided by the rheovela core (AGPL) via runtime.WorkerBridge.
type fakeStore struct {
	mu    sync.Mutex
	items map[string]worker.WorkItem
	token string
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: map[string]worker.WorkItem{}}
}

func (s *fakeStore) seed(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[id] = worker.WorkItem{ID: id, InstanceID: "inst-" + id, ActivityID: "act", State: "ready"}
}

func (s *fakeStore) PollReady() ([]worker.WorkItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]worker.WorkItem, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, it)
	}
	return out, nil
}

func (s *fakeStore) Claim(id string, lease time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if !ok || it.State != "ready" {
		return "", errors.New("not claimable")
	}
	s.token = "tok-" + id
	return s.token, nil
}

func (s *fakeStore) Heartbeat(id, token string, lease time.Duration) error { return nil }

func (s *fakeStore) Complete(id, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if ok && token == s.token {
		it.State = "done"
		s.items[id] = it
	}
	return nil
}

func (s *fakeStore) Fail(id, token, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if ok && token == s.token {
		it.State = "failed"
		s.items[id] = it
	}
	return nil
}

// runExample seeds two ready work items, processes them once, and reports how
// many were processed plus each item's final state.
func runExample() (processed int, states map[string]string, err error) {
	store := newFakeStore()
	store.seed("task-1")
	store.seed("task-2")

	fn := func(ctx context.Context, item worker.WorkItem) worker.WorkResult {
		if item.ID == "task-1" {
			return worker.WorkResult{Status: "success", Outcome: "ok"}
		}
		return worker.WorkResult{Status: "failure", Error: "simulated failure"}
	}

	w := worker.New(store, fn, time.Second)
	processed, err = w.ProcessOnce(context.Background())
	if err != nil {
		return processed, nil, err
	}

	states = map[string]string{}
	for id, it := range store.items {
		states[id] = it.State
	}
	return processed, states, nil
}

func main() {
	processed, states, err := runExample()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("processed=%d\n", processed)
	for id, st := range states {
		fmt.Printf("  %s: %s\n", id, st)
	}
}
