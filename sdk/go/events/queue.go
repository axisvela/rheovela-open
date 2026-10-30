package events

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const stateFile = "events-state.json"

// Queue is a small durable pending-event buffer. It is safe for concurrent
// use by one delivery loop.
type Queue struct {
	mu    sync.Mutex
	path  string
	state persistedState
}

type persistedState struct {
	LastSeq int64   `json:"last_seq"`
	Pending []Event `json:"pending"`
}

// OpenQueue loads (or creates) the durable state under dir.
func OpenQueue(dir string) (*Queue, error) {
	if dir == "" {
		return nil, errors.New("events: state dir is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	q := &Queue{path: filepath.Join(dir, stateFile)}
	raw, err := os.ReadFile(q.path)
	if errors.Is(err, os.ErrNotExist) {
		return q, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return q, nil
	}
	if err := json.Unmarshal(raw, &q.state); err != nil {
		return nil, err
	}
	q.normalize()
	return q, nil
}

// LastSeq returns the highest acknowledged seq.
func (q *Queue) LastSeq() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.state.LastSeq
}

// Pending returns a copy of buffered events in ascending seq order.
func (q *Queue) Pending() []Event {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]Event(nil), q.state.Pending...)
}

// SetFloor raises the acknowledged seq when there is no buffered work.
func (q *Queue) SetFloor(seq int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.state.Pending) > 0 || seq <= q.state.LastSeq {
		return nil
	}
	q.state.LastSeq = seq
	return q.persistLocked()
}

// Push buffers an event that has not been acknowledged yet.
func (q *Queue) Push(event Event) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if event.Seq == 0 || event.Seq <= q.state.LastSeq {
		return nil
	}
	for _, pending := range q.state.Pending {
		if pending.Seq == event.Seq {
			return nil
		}
	}
	q.state.Pending = append(q.state.Pending, event)
	q.normalize()
	return q.persistLocked()
}

// Ack marks seq (and every earlier buffered event) as handled.
func (q *Queue) Ack(seq int64) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if seq > q.state.LastSeq {
		q.state.LastSeq = seq
	}
	kept := q.state.Pending[:0]
	for _, event := range q.state.Pending {
		if event.Seq > seq {
			kept = append(kept, event)
		}
	}
	q.state.Pending = kept
	return q.persistLocked()
}

func (q *Queue) normalize() {
	sort.Slice(q.state.Pending, func(i, j int) bool {
		return q.state.Pending[i].Seq < q.state.Pending[j].Seq
	})
	out := q.state.Pending[:0]
	var last int64
	for _, event := range q.state.Pending {
		if event.Seq == 0 || event.Seq <= q.state.LastSeq || event.Seq == last {
			continue
		}
		last = event.Seq
		out = append(out, event)
	}
	q.state.Pending = out
}

func (q *Queue) persistLocked() error {
	raw, err := json.Marshal(q.state)
	if err != nil {
		return err
	}
	tmp := q.path + ".tmp"
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, q.path)
}
