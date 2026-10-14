package rheovela

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// capture is a mutex-guarded record of the last request the fake server saw.
type capture struct {
	mu   sync.Mutex
	auth string
	path string
	body string
}

func (c *capture) set(r *http.Request) {
	b := make([]byte, r.ContentLength)
	if r.ContentLength > 0 {
		_, _ = r.Body.Read(b)
	}
	c.mu.Lock()
	c.auth = r.Header.Get("Authorization")
	c.path = r.URL.Path
	c.body = string(b)
	c.mu.Unlock()
}

func (c *capture) get() (auth, path, body string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.auth, c.path, c.body
}

func newFake(t *testing.T, h http.HandlerFunc) (*Client, *capture) {
	t.Helper()
	cap := &capture{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cap.set(r)
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, WithTokenSource(StaticTokenSource("tok-1"))), cap
}

func TestClient_OpenRun(t *testing.T) {
	c, cap := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"instance_id":"i1","status":"active","project_id":"p1"}`))
	})
	inst, err := c.OpenRun(context.Background(), OpenRunRequest{Workflow: "wf-1", ProjectID: "p1", Actor: "axond"})
	if err != nil {
		t.Fatalf("OpenRun: %v", err)
	}
	if inst.InstanceID != "i1" || inst.Status != "active" {
		t.Errorf("instance = %+v", inst)
	}
	auth, path, body := cap.get()
	if auth != "Bearer tok-1" {
		t.Errorf("auth = %q, want Bearer tok-1", auth)
	}
	if path != "/api/v1/instances" {
		t.Errorf("path = %q", path)
	}
	if body == "" {
		t.Error("expected request body")
	}
}

func TestClient_Step(t *testing.T) {
	c, cap := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	if err := c.Step(context.Background(), "i1", StepRequest{Action: "enter", Stage: "s1"}); err != nil {
		t.Fatalf("Step: %v", err)
	}
	if _, path, _ := cap.get(); path != "/api/v1/instances/i1/steps" {
		t.Errorf("path = %q", path)
	}
}

func TestClient_ErrorStatus(t *testing.T) {
	c, _ := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"instance not found"}`))
	})
	if _, err := c.GetInstance(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClient_ClaimWorkItem(t *testing.T) {
	c, cap := newFake(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"ft-1","lease":"30s","lease_until":"2026-10-14T00:00:00Z"}`))
	})
	claim, err := c.ClaimWorkItem(context.Background(), "w1", "30s")
	if err != nil {
		t.Fatalf("ClaimWorkItem: %v", err)
	}
	if claim.Token != "ft-1" {
		t.Errorf("token = %q", claim.Token)
	}
	if _, path, _ := cap.get(); path != "/api/v1/work-items/w1/claim" {
		t.Errorf("path = %q", path)
	}
}
