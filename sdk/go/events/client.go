package events

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Config configures a durable event client.
type Config struct {
	URL            string
	StreamID       string
	Types          []string
	CorrelationKey string
	Predicate      string
	StateDir       string
	AfterSeq       int64
	ReconnectMin   time.Duration
	ReconnectMax   time.Duration
	RetryDelay     time.Duration
	Handler        func(context.Context, Event) error
	OnError        func(error)
}

// Client consumes a filtered Rheovela event stream over WebSocket, persisting
// its cursor and unacknowledged events under StateDir.
type Client struct {
	cfg   Config
	queue *Queue
}

// New validates config and opens the durable queue.
func New(cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, errors.New("events: url is required")
	}
	if cfg.StreamID == "" {
		return nil, errors.New("events: stream id is required")
	}
	if cfg.StateDir == "" {
		return nil, errors.New("events: state dir is required")
	}
	if cfg.Handler == nil {
		return nil, errors.New("events: handler is required")
	}
	if cfg.ReconnectMin <= 0 {
		cfg.ReconnectMin = 200 * time.Millisecond
	}
	if cfg.ReconnectMax <= 0 {
		cfg.ReconnectMax = 30 * time.Second
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = time.Second
	}
	queue, err := OpenQueue(cfg.StateDir)
	if err != nil {
		return nil, err
	}
	if cfg.AfterSeq > 0 {
		if err := queue.SetFloor(cfg.AfterSeq); err != nil {
			return nil, err
		}
	}
	return &Client{cfg: cfg, queue: queue}, nil
}

// LastSeq returns the highest acknowledged seq.
func (c *Client) LastSeq() int64 { return c.queue.LastSeq() }

// Pending returns buffered unacknowledged events.
func (c *Client) Pending() []Event { return c.queue.Pending() }

// Run connects, drains pending events, and reconnects until ctx is done.
func (c *Client) Run(ctx context.Context) error {
	backoff := c.cfg.ReconnectMin
	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := c.drain(ctx); err != nil {
			c.report(err)
			if !sleep(ctx, c.cfg.RetryDelay) {
				return nil
			}
			continue
		}
		if err := c.session(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			c.report(err)
			if !sleep(ctx, backoff) {
				return nil
			}
			backoff *= 2
			if backoff > c.cfg.ReconnectMax {
				backoff = c.cfg.ReconnectMax
			}
			continue
		}
		backoff = c.cfg.ReconnectMin
	}
}

func (c *Client) drain(ctx context.Context) error {
	for _, event := range c.queue.Pending() {
		if err := c.cfg.Handler(ctx, event); err != nil {
			return err
		}
		if err := c.queue.Ack(event.Seq); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) session(ctx context.Context) error {
	endpoint, err := c.endpoint()
	if err != nil {
		return err
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, endpoint, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-stop:
		}
	}()

	for {
		var event Event
		if err := conn.ReadJSON(&event); err != nil {
			return err
		}
		if event.Seq == 0 {
			continue
		}
		if err := c.queue.Push(event); err != nil {
			return err
		}
		if err := c.cfg.Handler(ctx, event); err != nil {
			return err
		}
		if err := c.queue.Ack(event.Seq); err != nil {
			return err
		}
		_ = conn.WriteJSON(map[string]any{"type": "ack", "seq": event.Seq})
	}
}

func (c *Client) endpoint() (string, error) {
	parsed, err := url.Parse(c.cfg.URL)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "http", "ws":
		parsed.Scheme = "ws"
	case "https", "wss":
		parsed.Scheme = "wss"
	default:
		return "", errors.New("events: unsupported url scheme")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/api/v1/events/ws"
	query := parsed.Query()
	query.Set("stream", c.cfg.StreamID)
	if seq := c.queue.LastSeq(); seq > 0 {
		query.Set("after_seq", strconv.FormatInt(seq, 10))
	}
	if len(c.cfg.Types) > 0 {
		query.Set("types", strings.Join(c.cfg.Types, ","))
	}
	if c.cfg.CorrelationKey != "" {
		query.Set("correlation_key", c.cfg.CorrelationKey)
	}
	if c.cfg.Predicate != "" {
		query.Set("predicate", c.cfg.Predicate)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (c *Client) report(err error) {
	if err != nil && c.cfg.OnError != nil {
		c.cfg.OnError(err)
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}
