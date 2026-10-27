// Package contracts 定义 RHEOVELA 版本化契约（schema_version=1）。
package contracts

import (
	"encoding/json"
	"time"
)

const SchemaVersion = "1"

// KnownEventTypes 是 engine/validator 已知的事件类型目录。
var KnownEventTypes = []string{
	"RunOpened", "StepEntered", "StepCompleted", "StepFailed", "StepSkipped",
	"RunClosed", "StageAssigned", "TimerFired", "Migrated", "CompensationExecuted",
	"SubprocessStarted", "ProcessSuspended", "ProcessResumed",
	"WorkItemApprovalRequested", "WorkItemApproved", "WorkItemRejected",
}

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
	WorkflowID    string         `json:"workflow_id"`
	ProjectID     string         `json:"project_id,omitempty"`
	Title         string         `json:"title"`
	Kind          string         `json:"kind,omitempty"`
	Source        string         `json:"source"`
	Stages        []Stage        `json:"stages"`
	Transitions   []Transition   `json:"transitions"`
	Compensations []Compensation `json:"compensations,omitempty"`
	BPMN          *BPMN          `json:"bpmn,omitempty"`
}

// BPMN preserves non-executable BPMN structure so import/export can round-trip
// subprocesses, boundary events and collaborations.
type BPMN struct {
	Nodes                  []BPMNNode            `json:"nodes,omitempty"`
	Flows                  []BPMNFlow            `json:"flows,omitempty"`
	InternalFlows          []BPMNFlow            `json:"internal_flows,omitempty"`
	Processes              []BPMNProcess         `json:"processes,omitempty"`
	Participants           []BPMNParticipant     `json:"participants,omitempty"`
	MessageFlows           []BPMNMessageFlow     `json:"message_flows,omitempty"`
	Lanes                  []BPMNLane            `json:"lanes,omitempty"`
	DataObjects            []BPMNDataObject      `json:"data_objects,omitempty"`
	DataStores             []BPMNDataStore       `json:"data_stores,omitempty"`
	DataStoreRefs          []BPMNDataStoreRef    `json:"data_store_refs,omitempty"`
	Properties             []BPMNProperty        `json:"properties,omitempty"`
	ItemDefinitions        []BPMNItemDefinition  `json:"item_definitions,omitempty"`
	DataInputs             []BPMNDataInput       `json:"data_inputs,omitempty"`
	DataOutputs            []BPMNDataOutput      `json:"data_outputs,omitempty"`
	EventDefinitions       []BPMNEventDefinition `json:"event_definitions,omitempty"`
	Artifacts              []BPMNArtifact        `json:"artifacts,omitempty"`
	Resources              []BPMNResource        `json:"resources,omitempty"`
	Categories             []BPMNCategory        `json:"categories,omitempty"`
	Messages               []BPMNMessage         `json:"messages,omitempty"`
	Signals                []BPMNSignal          `json:"signals,omitempty"`
	Errors                 []BPMNError           `json:"errors,omitempty"`
	Escalations            []BPMNEscalation      `json:"escalations,omitempty"`
	Interfaces             []BPMNInterface       `json:"interfaces,omitempty"`
	EndPoints              []BPMNEndPoint        `json:"end_points,omitempty"`
	ProcessIOSpecification *BPMNIOSpecification  `json:"process_io_specification,omitempty"`
	Collaboration          *BPMNCollaboration    `json:"collaboration,omitempty"`
	Choreographies         []BPMNChoreography    `json:"choreographies,omitempty"`
	DataAssociations       []BPMNDataAssociation `json:"data_associations,omitempty"`
	Diagram                *BPMNDiagram          `json:"diagram,omitempty"`
}

// BPMNProcess preserves a process/pool declaration.
type BPMNProcess struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	IsExecutable bool   `json:"is_executable,omitempty"`
	Participant  string `json:"participant,omitempty"`
}

// BPMNChoreography preserves a choreography diagram.
type BPMNChoreography struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name,omitempty"`
	ParticipantRefs []string               `json:"participant_refs,omitempty"`
	Tasks           []BPMNChoreographyTask `json:"tasks,omitempty"`
	MessageFlows    []BPMNMessageFlow      `json:"message_flows,omitempty"`
}

// BPMNChoreographyTask preserves a choreographyTask, subChoreography or
// callChoreography.
type BPMNChoreographyTask struct {
	ID                       string                       `json:"id"`
	Name                     string                       `json:"name,omitempty"`
	Element                  string                       `json:"element"`
	InitiatingParticipantRef string                       `json:"initiating_participant_ref,omitempty"`
	ParticipantRefs          []string                     `json:"participant_refs,omitempty"`
	MessageFlowRefs          []string                     `json:"message_flow_refs,omitempty"`
	LoopType                 string                       `json:"loop_type,omitempty"`
	ParentChoreography       string                       `json:"parent_choreography,omitempty"`
	CalledChoreographyRef    string                       `json:"called_choreography_ref,omitempty"`
	ParticipantAssociations  []BPMNParticipantAssociation `json:"participant_associations,omitempty"`
	Tasks                    []BPMNChoreographyTask       `json:"tasks,omitempty"`
}

// BPMNDataAssociation preserves a dataInputAssociation or dataOutputAssociation.
type BPMNDataAssociation struct {
	ID              string           `json:"id"`
	Element         string           `json:"element"`
	Owner           string           `json:"owner"`
	Sources         []string         `json:"sources,omitempty"`
	Targets         []string         `json:"targets,omitempty"`
	Transformations []string         `json:"transformations,omitempty"`
	Assignments     []BPMNAssignment `json:"assignments,omitempty"`
}

// BPMNAssignment preserves an assignment inside a data association.
type BPMNAssignment struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

type BPMNDiagram struct {
	ID          string           `json:"id,omitempty"`
	PlaneID     string           `json:"plane_id,omitempty"`
	BPMNElement string           `json:"bpmn_element,omitempty"`
	Resolution  float64          `json:"resolution,omitempty"`
	Shapes      []BPMNShape      `json:"shapes,omitempty"`
	Edges       []BPMNEdge       `json:"edges,omitempty"`
	LabelStyles []BPMNLabelStyle `json:"label_styles,omitempty"`
}

type BPMNShape struct {
	ID                  string     `json:"id"`
	BPMNElement         string     `json:"bpmn_element,omitempty"`
	X                   float64    `json:"x"`
	Y                   float64    `json:"y"`
	Width               float64    `json:"width"`
	Height              float64    `json:"height"`
	IsExpanded          *bool      `json:"is_expanded,omitempty"`
	IsMarkerVisible     *bool      `json:"is_marker_visible,omitempty"`
	IsHorizontal        *bool      `json:"is_horizontal,omitempty"`
	IsMessageVisible    *bool      `json:"is_message_visible,omitempty"`
	ParticipantBandKind string     `json:"participant_band_kind,omitempty"`
	Label               *BPMNLabel `json:"label,omitempty"`
}

type BPMNEdge struct {
	ID                 string      `json:"id"`
	BPMNElement        string      `json:"bpmn_element,omitempty"`
	SourceElement      string      `json:"source_element,omitempty"`
	TargetElement      string      `json:"target_element,omitempty"`
	MessageVisibleKind string      `json:"message_visible_kind,omitempty"`
	Waypoints          []BPMNPoint `json:"waypoints,omitempty"`
	Label              *BPMNLabel  `json:"label,omitempty"`
}

// BPMNLabel preserves a DI label and its optional bounds/style reference.
type BPMNLabel struct {
	Bounds     *BPMNBounds `json:"bounds,omitempty"`
	LabelStyle string      `json:"label_style,omitempty"`
}

// BPMNBounds is a dc:Bounds rectangle.
type BPMNBounds struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// BPMNLabelStyle preserves a DI label style.
type BPMNLabelStyle struct {
	ID         string  `json:"id"`
	FontName   string  `json:"font_name,omitempty"`
	FontSize   float64 `json:"font_size,omitempty"`
	FontBold   *bool   `json:"font_bold,omitempty"`
	FontItalic *bool   `json:"font_italic,omitempty"`
}

type BPMNPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type BPMNLane struct {
	ID      string   `json:"id"`
	Name    string   `json:"name,omitempty"`
	NodeIDs []string `json:"node_ids,omitempty"`
}

// BPMNDataObject records a dataObject (Ref empty) or dataObjectReference
// (Ref points at the referenced dataObject).
type BPMNDataObject struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	Ref            string `json:"ref,omitempty"`
	ItemSubjectRef string `json:"item_subject_ref,omitempty"`
	IsCollection   *bool  `json:"is_collection,omitempty"`
	DataState      string `json:"data_state,omitempty"`
}

type BPMNDataStore struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	ItemSubjectRef string `json:"item_subject_ref,omitempty"`
	IsUnlimited    *bool  `json:"is_unlimited,omitempty"`
	Capacity       string `json:"capacity,omitempty"`
}

type BPMNDataStoreRef struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	DataStoreRef string `json:"data_store_ref,omitempty"`
}

type BPMNProperty struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	ItemSubjectRef string `json:"item_subject_ref,omitempty"`
}

type BPMNItemDefinition struct {
	ID           string `json:"id"`
	StructureRef string `json:"structure_ref,omitempty"`
	IsCollection *bool  `json:"is_collection,omitempty"`
}

type BPMNDataInput struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	ItemSubjectRef string `json:"item_subject_ref,omitempty"`
	IsCollection   *bool  `json:"is_collection,omitempty"`
}

type BPMNDataOutput struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	ItemSubjectRef string `json:"item_subject_ref,omitempty"`
	IsCollection   *bool  `json:"is_collection,omitempty"`
}

type BPMNResource struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// BPMNResourceRole preserves performer/humanPerformer/potentialOwner/resourceRole.
type BPMNResourceRole struct {
	ID                   string                         `json:"id"`
	Element              string                         `json:"element"`
	Name                 string                         `json:"name,omitempty"`
	ResourceRef          string                         `json:"resource_ref,omitempty"`
	ParameterBindings    []BPMNResourceParameterBinding `json:"parameter_bindings,omitempty"`
	AssignmentExpression string                         `json:"assignment_expression,omitempty"`
}

type BPMNResourceParameterBinding struct {
	ID           string `json:"id"`
	ParameterRef string `json:"parameter_ref,omitempty"`
	Expression   string `json:"expression,omitempty"`
}

type BPMNIOSpecification struct {
	ID             string          `json:"id"`
	DataInputRefs  []string        `json:"data_input_refs,omitempty"`
	DataOutputRefs []string        `json:"data_output_refs,omitempty"`
	InputSets      []BPMNInputSet  `json:"input_sets,omitempty"`
	OutputSets     []BPMNOutputSet `json:"output_sets,omitempty"`
}

type BPMNInputSet struct {
	ID            string   `json:"id"`
	Name          string   `json:"name,omitempty"`
	DataInputRefs []string `json:"data_input_refs,omitempty"`
}

type BPMNOutputSet struct {
	ID             string   `json:"id"`
	Name           string   `json:"name,omitempty"`
	DataOutputRefs []string `json:"data_output_refs,omitempty"`
}

type BPMNCategory struct {
	ID     string              `json:"id"`
	Name   string              `json:"name,omitempty"`
	Values []BPMNCategoryValue `json:"values,omitempty"`
}

type BPMNCategoryValue struct {
	ID    string `json:"id"`
	Value string `json:"value,omitempty"`
}

type BPMNMessage struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	ItemRef string `json:"item_ref,omitempty"`
}

type BPMNSignal struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	StructureRef string `json:"structure_ref,omitempty"`
}

type BPMNError struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	StructureRef string `json:"structure_ref,omitempty"`
}

type BPMNEscalation struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	EscalationCode string `json:"escalation_code,omitempty"`
	StructureRef   string `json:"structure_ref,omitempty"`
}

type BPMNOperation struct {
	ID                string   `json:"id"`
	Name              string   `json:"name,omitempty"`
	ImplementationRef string   `json:"implementation_ref,omitempty"`
	InMessageRef      string   `json:"in_message_ref,omitempty"`
	OutMessageRef     string   `json:"out_message_ref,omitempty"`
	ErrorRefs         []string `json:"error_refs,omitempty"`
}

type BPMNInterface struct {
	ID                string          `json:"id"`
	Name              string          `json:"name,omitempty"`
	ImplementationRef string          `json:"implementation_ref,omitempty"`
	Operations        []BPMNOperation `json:"operations,omitempty"`
}

type BPMNEndPoint struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type BPMNCollaboration struct {
	ID            string             `json:"id"`
	Name          string             `json:"name,omitempty"`
	Conversations []BPMNConversation `json:"conversations,omitempty"`
}

type BPMNConversation struct {
	ID                     string                 `json:"id"`
	Name                   string                 `json:"name,omitempty"`
	Element                string                 `json:"element"`
	CalledCollaborationRef string                 `json:"called_collaboration_ref,omitempty"`
	ParticipantRefs        []string               `json:"participant_refs,omitempty"`
	MessageFlowRefs        []string               `json:"message_flow_refs,omitempty"`
	Links                  []BPMNConversationLink `json:"links,omitempty"`
	Conversations          []BPMNConversation     `json:"conversations,omitempty"`
}

type BPMNConversationLink struct {
	ID     string `json:"id"`
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
}

type BPMNParticipantAssociation struct {
	InnerParticipantRef string `json:"inner_participant_ref,omitempty"`
	OuterParticipantRef string `json:"outer_participant_ref,omitempty"`
}

// BPMNArtifact preserves textAnnotation, association or group.
type BPMNArtifact struct {
	ID                   string `json:"id"`
	Element              string `json:"element"`
	Text                 string `json:"text,omitempty"`
	Source               string `json:"source,omitempty"`
	Target               string `json:"target,omitempty"`
	AssociationDirection string `json:"association_direction,omitempty"`
	CategoryValueRef     string `json:"category_value_ref,omitempty"`
}

// BPMNNode maps a workflow stage (or a subprocess-internal node) back to its
// BPMN element type and event semantics.
type BPMNNode struct {
	ID               string `json:"id"`
	Element          string `json:"element"`
	Name             string `json:"name,omitempty"`
	EventKind        string `json:"event_kind,omitempty"`
	AttachedTo       string `json:"attached_to,omitempty"`
	ParentSubprocess string `json:"parent_subprocess,omitempty"`
	ProcessID        string `json:"process_id,omitempty"`

	// Gateway semantics.
	GatewayDirection string `json:"gateway_direction,omitempty"`
	EventGatewayType string `json:"event_gateway_type,omitempty"`
	Instantiate      *bool  `json:"instantiate,omitempty"`
	DefaultFlow      string `json:"default_flow,omitempty"`

	// Event semantics.
	CancelActivity      *bool                 `json:"cancel_activity,omitempty"`
	ParallelMultiple    *bool                 `json:"parallel_multiple,omitempty"`
	Direction           string                `json:"direction,omitempty"`
	EventDefinitions    []BPMNEventDefinition `json:"event_definitions,omitempty"`
	EventDefinitionRefs []string              `json:"event_definition_refs,omitempty"`

	// Activity/subprocess semantics.
	CalledElement            string               `json:"called_element,omitempty"`
	TriggeredByEvent         *bool                `json:"triggered_by_event,omitempty"`
	IsForCompensation        *bool                `json:"is_for_compensation,omitempty"`
	Ordering                 string               `json:"ordering,omitempty"`
	CancelRemainingInstances *bool                `json:"cancel_remaining_instances,omitempty"`
	Method                   string               `json:"method,omitempty"`
	CompletionQuantity       string               `json:"completion_quantity,omitempty"`
	StartQuantity            string               `json:"start_quantity,omitempty"`
	Loop                     *BPMNLoop            `json:"loop,omitempty"`
	ResourceRoles            []BPMNResourceRole   `json:"resource_roles,omitempty"`
	IOSpecification          *BPMNIOSpecification `json:"io_specification,omitempty"`
	ActivationCondition      string               `json:"activation_condition,omitempty"`
	AdHocCompletionCondition string               `json:"ad_hoc_completion_condition,omitempty"`
}

// BPMNLoop preserves standardLoopCharacteristics or
// multiInstanceLoopCharacteristics.
type BPMNLoop struct {
	Kind                 string `json:"kind"`
	IsSequential         *bool  `json:"is_sequential,omitempty"`
	LoopCardinality      string `json:"loop_cardinality,omitempty"`
	CompletionCondition  string `json:"completion_condition,omitempty"`
	Collection           string `json:"collection,omitempty"`
	ElementVariable      string `json:"element_variable,omitempty"`
	TestBefore           *bool  `json:"test_before,omitempty"`
	LoopMaximum          string `json:"loop_maximum,omitempty"`
	LoopCondition        string `json:"loop_condition,omitempty"`
	Behavior             string `json:"behavior,omitempty"`
	OneBehaviorEventRef  string `json:"one_behavior_event_ref,omitempty"`
	NoneBehaviorEventRef string `json:"none_behavior_event_ref,omitempty"`
}

// BPMNEventDefinition preserves one eventDefinition element and its attributes.
type BPMNEventDefinition struct {
	ID                string `json:"id,omitempty"`
	Kind              string `json:"kind"`
	Ref               string `json:"ref,omitempty"`
	Name              string `json:"name,omitempty"`
	TimeDate          string `json:"time_date,omitempty"`
	TimeDuration      string `json:"time_duration,omitempty"`
	TimeCycle         string `json:"time_cycle,omitempty"`
	ActivityRef       string `json:"activity_ref,omitempty"`
	WaitForCompletion *bool  `json:"wait_for_completion,omitempty"`
	Condition         string `json:"condition,omitempty"`
}

// BPMNFlow is a sequenceFlow inside a subprocess.
type BPMNFlow struct {
	ID               string `json:"id"`
	From             string `json:"from"`
	To               string `json:"to"`
	Name             string `json:"name,omitempty"`
	Condition        string `json:"condition,omitempty"`
	ParentSubprocess string `json:"parent_subprocess,omitempty"`
}

type BPMNParticipant struct {
	ID            string                       `json:"id"`
	Name          string                       `json:"name,omitempty"`
	ProcessRef    string                       `json:"process_ref,omitempty"`
	Multiplicity  *BPMNParticipantMultiplicity `json:"multiplicity,omitempty"`
	InterfaceRefs []string                     `json:"interface_refs,omitempty"`
	EndPointRefs  []string                     `json:"end_point_refs,omitempty"`
}

// BPMNParticipantMultiplicity preserves participantMultiplicity.
type BPMNParticipantMultiplicity struct {
	Minimum int    `json:"minimum,omitempty"`
	Maximum string `json:"maximum,omitempty"`
}

type BPMNMessageFlow struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	Target     string `json:"target"`
	Name       string `json:"name,omitempty"`
	MessageRef string `json:"message_ref,omitempty"`
}

type Stage struct {
	ID              string         `json:"id"`
	Title           string         `json:"title"`
	Assignee        string         `json:"assignee,omitempty"`
	AssigneeSpec    *AssigneeSpec  `json:"assignee_spec,omitempty"`
	OwnerSkill      string         `json:"owner_skill,omitempty"`
	Inputs          []string       `json:"inputs"`
	Outputs         []string       `json:"outputs"`
	Gate            *GateCondition `json:"gate,omitempty"`
	Delegation      *Delegation    `json:"delegation,omitempty"`
	Evidence        []string       `json:"evidence,omitempty"`
	MigrationPolicy string         `json:"migration_policy,omitempty"`
	EffectKey       string         `json:"effect_key,omitempty"`
	Timeout         string         `json:"timeout,omitempty"`
}

// Delegation describes an authority delegation attached to a stage.
type Delegation struct {
	Kind      string   `json:"kind"`
	Delegatee string   `json:"delegatee"`
	Scopes    []string `json:"scopes,omitempty"`
	Budget    int64    `json:"budget,omitempty"`
	Evidence  []string `json:"evidence,omitempty"`
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

// Compensation declares how to compensate a completed stage. CompensateWith
// names another stage; Capability names a compensating capability.
type Compensation struct {
	ForActivity    string `json:"for_activity"`
	CompensateWith string `json:"compensate_with,omitempty"`
	Capability     string `json:"capability,omitempty"`
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
