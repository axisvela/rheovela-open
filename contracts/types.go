// Package contracts 定义 RHEOVELA 版本化契约（schema_version=1）。
package contracts

import (
	"encoding/json"
	"time"
)

const SchemaVersion = "1"

type ActorType string

const (
	ActorHuman ActorType = "human"
	ActorAgent ActorType = "agent"
)

type Actor struct {
	ID           string    `json:"id"`
	Type         ActorType `json:"type"`
	Capabilities []string  `json:"capabilities"`
}

type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Event struct {
	ID            string          `json:"id"`
	StreamID      string          `json:"stream_id"`
	ActorID       string          `json:"actor_id"`
	Type          string          `json:"type"`
	Payload       json.RawMessage `json:"payload"`
	LamportClock  int64           `json:"lamport_clock"`
	WallTime      time.Time       `json:"wall_time"`
	SchemaVersion string          `json:"schema_version,omitempty"`
	Signature     string          `json:"signature,omitempty"`
}

type WorkflowDefinition struct {
	WorkflowID  string       `json:"workflow_id"`
	ProjectID   string       `json:"project_id,omitempty"`
	Title       string       `json:"title"`
	Kind        string       `json:"kind,omitempty"`
	Source      string       `json:"source"`
	Stages      []Stage      `json:"stages"`
	Transitions []Transition `json:"transitions"`
}

type Stage struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Assignee     string         `json:"assignee,omitempty"`
	AssigneeSpec *AssigneeSpec  `json:"assignee_spec,omitempty"`
	OwnerSkill   string         `json:"owner_skill,omitempty"`
	Inputs       []string       `json:"inputs"`
	Outputs      []string       `json:"outputs"`
	Gate         *GateCondition `json:"gate,omitempty"`
}

type AssigneeSpec struct {
	Kind     string   `json:"kind"`
	ActorID  string   `json:"actor_id,omitempty"`
	Role     string   `json:"role,omitempty"`
	Required []string `json:"required,omitempty"`
}

type StageExecution struct {
	RunID       string   `json:"run_id"`
	StageID     string   `json:"stage_id"`
	AssignedTo  string   `json:"assigned_to,omitempty"`
	AssignedBy  string   `json:"assigned_by,omitempty"`
	ExecutedBy  []string `json:"executed_by"`
	CompletedBy string   `json:"completed_by,omitempty"`
	ViaSkill    string   `json:"via_skill,omitempty"`
	Status      string   `json:"status"`
}

type GateCondition struct {
	Requires []string `json:"requires"`
}

type Transition struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Condition string `json:"condition,omitempty"`
}

type RunContext struct {
	RunID           string            `json:"run_id"`
	ProjectID       string            `json:"project_id"`
	WorkflowID      string            `json:"workflow_id"`
	TaskID          string            `json:"task_id,omitempty"`
	Label           string            `json:"label,omitempty"`
	CurrentStage    string            `json:"current_stage"`
	Assignee        string            `json:"assignee,omitempty"`
	CompletedStages []string          `json:"completed_stages"`
	Variables       map[string]string `json:"variables,omitempty"`
	Status          string            `json:"status"`
	OpenedBy        string            `json:"opened_by"`
	StartedAt       time.Time         `json:"started_at"`
	ClosedAt        *time.Time        `json:"closed_at,omitempty"`
}
