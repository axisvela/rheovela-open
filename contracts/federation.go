package contracts

import (
	"encoding/json"
	"time"
)

// FederationEnvelope is the wire contract for one signed cross-org message.
// SenderTenantID is the originating tenant; TenantID is the receiving tenant.
// Older same-tenant peers omit SenderTenantID and use TenantID for both.
type FederationEnvelope struct {
	ID             string          `json:"id"`
	FromPeer       string          `json:"from_peer"`
	ToPeer         string          `json:"to_peer"`
	TenantID       string          `json:"tenant_id"`
	SenderTenantID string          `json:"sender_tenant_id,omitempty"`
	InstanceID     string          `json:"instance_id,omitempty"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload,omitempty"`
	SentAt         time.Time       `json:"sent_at"`
	Nonce          string          `json:"nonce"`
	Alg            string          `json:"alg,omitempty"`
	KeyID          string          `json:"key_id,omitempty"`
	Signature      string          `json:"signature,omitempty"`
}

// FederationSLA carries the shared SLA context for a handoff. Enforcement stays
// with the receiving tenant's runtime.
type FederationSLA struct {
	PolicyID string `json:"policy_id,omitempty"`
	Deadline string `json:"deadline,omitempty"`
}

// FederationHandoff is the InstanceHandoff payload.
type FederationHandoff struct {
	WorkflowID string            `json:"workflow_id"`
	ProjectID  string            `json:"project_id"`
	InstanceID string            `json:"instance_id,omitempty"`
	Label      string            `json:"label,omitempty"`
	Variables  map[string]string `json:"variables,omitempty"`
	SLA        *FederationSLA    `json:"sla,omitempty"`
}
