package contracts_test

import (
	"encoding/json"
	"testing"
	"time"

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

func TestRunContextJSONRoundTrip(t *testing.T) {
	closed := time.Date(2026, 6, 9, 12, 0, 0, 0, time.UTC)
	ctx := contracts.RunContext{
		RunID:           "R1",
		ProjectID:       "P1",
		WorkflowID:      "WF1",
		CurrentStage:    "review",
		CompletedStages: []string{"research", "draft"},
		Status:          "active",
		OpenedBy:        "agent-1",
		StartedAt:       time.Date(2026, 6, 9, 10, 0, 0, 0, time.UTC),
		ClosedAt:        &closed,
	}

	b, err := json.Marshal(ctx)
	require.NoError(t, err)

	var decoded contracts.RunContext
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "R1", decoded.RunID)
	assert.Equal(t, []string{"research", "draft"}, decoded.CompletedStages)
	assert.Equal(t, "active", decoded.Status)
	assert.Equal(t, "review", decoded.CurrentStage)
}

func TestWorkflowDefinitionJSONRoundTrip(t *testing.T) {
	wf := contracts.WorkflowDefinition{
		WorkflowID: "paper-submission",
		Title:      "论文投稿流程",
		Source:     "pipeline",
		Stages: []contracts.Stage{
			{ID: "research", Title: "调研", Inputs: []string{"topic"}, Outputs: []string{"lit_review"}},
			{ID: "draft", Title: "起草", Inputs: []string{"lit_review"}, Outputs: []string{"manuscript"}},
			{ID: "review", Title: "评审", Inputs: []string{"manuscript"}, Outputs: []string{"review_notes"}},
			{ID: "submit", Title: "提交", Inputs: []string{"manuscript", "review_notes"}, Outputs: []string{"submission_id"}},
		},
		Transitions: []contracts.Transition{
			{From: "research", To: "draft"},
			{From: "draft", To: "review"},
			{From: "review", To: "submit"},
		},
	}

	b, err := json.Marshal(wf)
	require.NoError(t, err)

	var decoded contracts.WorkflowDefinition
	err = json.Unmarshal(b, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.Stages, 4)
	assert.Len(t, decoded.Transitions, 3)
}
