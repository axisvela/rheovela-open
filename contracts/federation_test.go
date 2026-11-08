package contracts

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFederationHandoffRoundTrip(t *testing.T) {
	handoff := FederationHandoff{
		WorkflowID: "expense",
		ProjectID:  "globex/expense",
		InstanceID: "fed-1",
		Label:      "cross-org",
		Variables:  map[string]string{"amount": "42"},
		SLA:        &FederationSLA{PolicyID: "gold", Deadline: "2026-11-09T12:00:00Z"},
	}
	raw, err := json.Marshal(handoff)
	if err != nil {
		t.Fatal(err)
	}
	var decoded FederationHandoff
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.WorkflowID != "expense" || decoded.SLA == nil || decoded.SLA.PolicyID != "gold" {
		t.Fatalf("decoded = %+v", decoded)
	}
	if decoded.Variables["amount"] != "42" {
		t.Fatalf("variables = %+v", decoded.Variables)
	}
}

func TestFederationEnvelopeCarriesSenderTenant(t *testing.T) {
	envelope := FederationEnvelope{
		ID:             "env-1",
		FromPeer:       "acme-node",
		ToPeer:         "globex-node",
		TenantID:       "globex",
		SenderTenantID: "acme",
		EventType:      "InstanceHandoff",
		SentAt:         time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC),
		Nonce:          "n-1",
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["sender_tenant_id"] != "acme" || fields["tenant_id"] != "globex" {
		t.Fatalf("fields = %+v", fields)
	}
}
