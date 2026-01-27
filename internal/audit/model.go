package audit

import (
	"time"
)

type EventType string

const (
	EventUserLogin             EventType = "USER_LOGIN"
	EventTransactionCreated    EventType = "TRANSACTION_CREATED"
	EventTransactionUpdated    EventType = "TRANSACTION_UPDATED"
	EventRiskEvaluated         EventType = "RISK_EVALUATED"
	EventUserBehaviorCreated   EventType = "USER_BEHAVIOR_CREATED"
	EventUserBehaviorUpdated   EventType = "USER_BEHAVIOR_UPDATED"
	EventSecurityUpdated       EventType = "SECURITY_UPDATED"
)

type AuditLog struct {
	// ---- Event ----
	EventID    string    `json:"event_id"`     // UUID
	EventType  EventType `json:"event_type"`
	EventTime  time.Time `json:"event_time"`

	// ---- Actor ----
	ActorType  string    `json:"actor_type"`   // USER | SYSTEM
	ActorID    string    `json:"actor_id"`     // users.id
	ActorRole  string    `json:"actor_role"`   // user.role

	// ---- Entity ----
	EntityType string    `json:"entity_type"`  // users, transactions, user_behavior
	EntityID   string    `json:"entity_id"`

	// ---- Context ----
	IPAddress  string    `json:"ip_address"`
	DeviceID   string    `json:"device_id,omitempty"`

	// ---- Action ----
	Action     string    `json:"action"`       // LOGIN | CREATE | UPDATE | EVALUATE
	Status     string    `json:"status"`       // SUCCESS | FAILURE
	Reason     string    `json:"reason,omitempty"`

	// ---- Change Tracking ----
	OldValues  map[string]interface{} `json:"old_values,omitempty"`
	NewValues  map[string]interface{} `json:"new_values,omitempty"`

	// ---- Transaction Context ----
	TransactionID string `json:"transaction_id,omitempty"`

	// ---- Risk Context ----
	RiskScore  *int    `json:"risk_score,omitempty"`
	RiskLevel  *string `json:"risk_level,omitempty"`
	Decision   *string `json:"decision,omitempty"`

	// ---- Correlation ----
	RequestID  string `json:"request_id"`
}



