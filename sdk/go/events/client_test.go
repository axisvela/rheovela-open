package events

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func waitSeq(t *testing.T, ch <-chan int64, want int64) {
	t.Helper()
	select {
	case got := <-ch:
		assert.Equal(t, want, got)
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for seq %d", want)
	}
}

func TestClientReceivesPersistsAndResumes(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteJSON(Event{Seq: 1, Type: "a"})
		_ = conn.WriteJSON(Event{Seq: 2, Type: "b"})
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	received := make(chan int64, 2)
	client, err := New(Config{
		URL: srv.URL, StreamID: "S", StateDir: t.TempDir(),
		RetryDelay: 10 * time.Millisecond, ReconnectMin: 10 * time.Millisecond,
		Handler: func(_ context.Context, event Event) error {
			received <- event.Seq
			return nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- client.Run(ctx) }()

	waitSeq(t, received, 1)
	waitSeq(t, received, 2)
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("client did not stop on cancel")
	}
	assert.Equal(t, int64(2), client.LastSeq())
	assert.Empty(t, client.Pending())
}

func TestClientReconnectsWithLastSeq(t *testing.T) {
	var mu sync.Mutex
	conns := 0
	afterSeqs := []string{}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		conns++
		conn := conns
		afterSeqs = append(afterSeqs, r.URL.Query().Get("after_seq"))
		mu.Unlock()

		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		if conn == 1 {
			_ = ws.WriteJSON(Event{Seq: 1, Type: "a"})
			return
		}
		_ = ws.WriteJSON(Event{Seq: 2, Type: "b"})
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	received := make(chan int64, 2)
	client, err := New(Config{
		URL: srv.URL, StreamID: "S", StateDir: t.TempDir(),
		RetryDelay: 10 * time.Millisecond, ReconnectMin: 10 * time.Millisecond,
		Handler: func(_ context.Context, event Event) error {
			received <- event.Seq
			return nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	waitSeq(t, received, 1)
	waitSeq(t, received, 2)
	mu.Lock()
	defer mu.Unlock()
	require.GreaterOrEqual(t, conns, 2)
	assert.Equal(t, "", afterSeqs[0])
	assert.Equal(t, "1", afterSeqs[1])
}

func TestClientRetriesHandlerFailures(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		_ = ws.WriteJSON(Event{Seq: 1, Type: "a"})
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	var mu sync.Mutex
	calls := 0
	client, err := New(Config{
		URL: srv.URL, StreamID: "S", StateDir: t.TempDir(),
		RetryDelay: 10 * time.Millisecond, ReconnectMin: 10 * time.Millisecond,
		Handler: func(context.Context, Event) error {
			mu.Lock()
			defer mu.Unlock()
			calls++
			if calls == 1 {
				return errors.New("transient")
			}
			return nil
		},
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := calls >= 2 && client.LastSeq() == 1
		mu.Unlock()
		if done {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	assert.GreaterOrEqual(t, calls, 2)
	mu.Unlock()
	assert.Equal(t, int64(1), client.LastSeq())
	assert.Empty(t, client.Pending())
}
