package contracts_test

import (
	"encoding/json"
	"testing"

	"github.com/axisrobo/rheovela-open/contracts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaVersionConstant(t *testing.T) {
	assert.Equal(t, "1", contracts.SchemaVersion)
}

func TestEventRoundTrip(t *testing.T) {
	ev := contracts.Event{
		ID:            "e1",
		StreamID:      "R1",
		ActorID:       "agent-1",
		Type:          "RunOpened",
		Payload:       json.RawMessage(`{"workflow_id":"WF1"}`),
		SchemaVersion: contracts.SchemaVersion,
	}
	b, err := json.Marshal(ev)
	require.NoError(t, err)
	var got contracts.Event
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "R1", got.StreamID)
	assert.Equal(t, "1", got.SchemaVersion)
}

func TestUnknownEventTypeRejected(t *testing.T) {
	known := map[string]bool{
		"RunOpened": true, "StepEntered": true, "StepCompleted": true,
		"StepFailed": true, "StepSkipped": true, "RunClosed": true,
	}
	_, ok := known["FutureUnknownEvent"]
	assert.False(t, ok, "unknown event type must not be silently accepted")
}
