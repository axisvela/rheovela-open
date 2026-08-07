package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// loadSchema reads api/workflow.schema.json and returns its decoded root map.
func loadSchema(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join("..", "api", "workflow.schema.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("schema file not found: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}
	return root
}

// requiredStrings returns the string values of the top-level "required" array.
func requiredStrings(schema map[string]any) []string {
	req, ok := schema["required"].([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, v := range req {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func TestWorkflowSchemaWellFormed(t *testing.T) {
	root := loadSchema(t)

	if got := root["$schema"]; got != "http://json-schema.org/draft-07/schema#" {
		t.Errorf("$schema = %v, want draft-07 URI", got)
	}
	if root["$id"] == nil || root["$id"] == "" {
		t.Errorf("$id must be present")
	}
	if root["title"] == nil || root["title"] == "" {
		t.Errorf("title must be present")
	}
	if root["description"] == nil || root["description"] == "" {
		t.Errorf("description must be present")
	}
	if root["type"] != "object" {
		t.Errorf("type = %v, want object", root["type"])
	}

	required := requiredStrings(root)
	if len(required) == 0 {
		t.Fatalf("top-level required array is missing or empty")
	}
	want := map[string]bool{"workflow_id": false, "title": false, "source": false, "stages": false, "transitions": false}
	for _, k := range required {
		if _, ok := want[k]; ok {
			want[k] = true
		}
	}
	for k, present := range want {
		if !present {
			t.Errorf("required field %q is missing from top-level required array", k)
		}
	}
}

func TestWorkflowSchemaRequiredFields(t *testing.T) {
	root := loadSchema(t)

	stagesSchema, ok := root["properties"].(map[string]any)["stages"].(map[string]any)
	if !ok {
		t.Fatalf("properties.stages is not an object schema")
	}
	if stagesSchema["type"] != "array" {
		t.Errorf("properties.stages.type = %v, want array", stagesSchema["type"])
	}
	stageItems, ok := stagesSchema["items"].(map[string]any)
	if !ok {
		t.Fatalf("properties.stages.items is not an object schema")
	}
	if stageItems["additionalProperties"] != false {
		t.Errorf("stages.items.additionalProperties = %v, want false", stageItems["additionalProperties"])
	}
	stageRequired := requiredStrings(stageItems)
	if !contains(stageRequired, "id") || !contains(stageRequired, "title") {
		t.Errorf("stages.items.required = %v, want it to include id and title", stageRequired)
	}

	assigneeSpec, ok := stageItems["properties"].(map[string]any)["assignee_spec"].(map[string]any)
	if !ok {
		t.Fatalf("stages.items.properties.assignee_spec is not an object schema")
	}
	kindEnum, ok := assigneeSpec["properties"].(map[string]any)["kind"].(map[string]any)["enum"].([]any)
	if !ok || len(kindEnum) == 0 {
		t.Errorf("assignee_spec.kind must declare an enum")
	}
	for _, wantKind := range []string{"actor", "role", "capability", "open"} {
		if !containsAny(kindEnum, wantKind) {
			t.Errorf("assignee_spec.kind enum is missing %q", wantKind)
		}
	}
	assigneeRequired := requiredStrings(assigneeSpec)
	if !contains(assigneeRequired, "kind") {
		t.Errorf("assignee_spec.required = %v, want it to include kind", assigneeRequired)
	}

	transitionsSchema, ok := root["properties"].(map[string]any)["transitions"].(map[string]any)
	if !ok {
		t.Fatalf("properties.transitions is not an object schema")
	}
	if transitionsSchema["type"] != "array" {
		t.Errorf("properties.transitions.type = %v, want array", transitionsSchema["type"])
	}
	transitionItems, ok := transitionsSchema["items"].(map[string]any)
	if !ok {
		t.Fatalf("properties.transitions.items is not an object schema")
	}
	transitionRequired := requiredStrings(transitionItems)
	if !contains(transitionRequired, "from") || !contains(transitionRequired, "to") {
		t.Errorf("transitions.items.required = %v, want it to include from and to", transitionRequired)
	}
}

func TestWorkflowSchemaValidatesSample(t *testing.T) {
	root := loadSchema(t)

	sample := `{
		"workflow_id": "PurchaseApproval",
		"title": "PurchaseApproval",
		"source": "rheo-ir",
		"stages": [
			{"id": "validate", "title": "validate", "assignee_spec": {"kind": "capability", "required": ["invoice.validate"]}, "inputs": [], "outputs": []},
			{"id": "approve", "title": "approve", "assignee_spec": {"kind": "role", "role": "cost-center-owner"}, "inputs": [], "outputs": []},
			{"id": "post", "title": "post", "assignee_spec": {"kind": "capability", "required": ["erp.invoice.post"]}, "inputs": [], "outputs": []}
		],
		"transitions": [
			{"from": "validate", "to": "approve", "condition": "amount >= 1000"},
			{"from": "validate", "to": "post", "condition": "amount < 1000"}
		]
	}`

	var doc map[string]any
	if err := json.Unmarshal([]byte(sample), &doc); err != nil {
		t.Fatalf("sample workflow is not valid JSON: %v", err)
	}

	// Structural check: every top-level required key in the schema must be present.
	for _, key := range requiredStrings(root) {
		if _, ok := doc[key]; !ok {
			t.Errorf("sample workflow is missing required top-level key %q", key)
		}
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func containsAny(xs []any, want string) bool {
	for _, x := range xs {
		if s, ok := x.(string); ok && s == want {
			return true
		}
	}
	return false
}
