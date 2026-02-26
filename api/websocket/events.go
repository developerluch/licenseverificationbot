package websocket

import (
	"encoding/json"
	"time"
)

// Event types broadcast to connected clients.
const (
	EventAgentJoined      = "agent_joined"
	EventStageChanged     = "stage_changed"
	EventLicenseVerified  = "license_verified"
	EventAgentKicked      = "agent_kicked"
	EventAgentNudged      = "agent_nudged"
	EventApprovalUpdated  = "approval_updated"
	EventRoleAssigned     = "role_assigned"
	EventMessageSent      = "message_sent"
	EventFormCompleted    = "form_completed"
)

// Event is the envelope sent to all WebSocket clients.
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// NewEvent creates a new event with the current timestamp.
func NewEvent(eventType string, data interface{}) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// JSON serialises the event.
func (e Event) JSON() []byte {
	b, _ := json.Marshal(e)
	return b
}

// --- Specific event payloads ---

type AgentJoinedData struct {
	DiscordID string `json:"discord_id"`
	Username  string `json:"username"`
	Stage     int    `json:"stage"`
}

type StageChangedData struct {
	DiscordID string `json:"discord_id"`
	OldStage  int    `json:"old_stage"`
	NewStage  int    `json:"new_stage"`
	ChangedBy string `json:"changed_by"` // "system" or admin discord ID
}

type LicenseVerifiedData struct {
	DiscordID string `json:"discord_id"`
	State     string `json:"state"`
	NPN       string `json:"npn"`
	Status    string `json:"status"` // "verified", "failed", "manual"
}

type AgentKickedData struct {
	DiscordID string `json:"discord_id"`
	Reason    string `json:"reason"`
	KickedBy  string `json:"kicked_by"`
}

type AgentNudgedData struct {
	DiscordID string `json:"discord_id"`
	Message   string `json:"message"`
	NudgedBy  string `json:"nudged_by"`
}

type ApprovalUpdatedData struct {
	ApprovalID int    `json:"approval_id"`
	DiscordID  string `json:"discord_id"`
	Action     string `json:"action"` // "approved" or "denied"
	ActionBy   string `json:"action_by"`
}

type RoleAssignedData struct {
	DiscordID  string `json:"discord_id"`
	RoleName   string `json:"role_name"`
	RoleID     string `json:"role_id"`
	Action     string `json:"action"` // "assign" or "remove"
	AssignedBy string `json:"assigned_by"`
}

type MessageSentData struct {
	DiscordID string `json:"discord_id"`
	Message   string `json:"message"`
	SentBy    string `json:"sent_by"`
}

type FormCompletedData struct {
	DiscordID string `json:"discord_id"`
	FullName  string `json:"full_name"`
	Agency    string `json:"agency"`
}
