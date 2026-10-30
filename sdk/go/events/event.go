// Package events provides a durable WebSocket event client with reconnect,
// cursor replay, and an at-least-once pending queue.
package events

import (
	"encoding/json"
	"time"
)

// Event is one event delivered by the Rheovela WebSocket gateway.
type Event struct {
	Seq      int64           `json:"seq"`
	Type     string          `json:"type"`
	ActorID  string          `json:"actor_id"`
	Payload  json.RawMessage `json:"payload"`
	WallTime time.Time       `json:"wall_time"`
}
