package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"license-bot-go/api/websocket"
	"license-bot-go/db"
)

// --- Request bodies ---

type kickRequest struct {
	Reason string `json:"reason"`
}

type nudgeRequest struct {
	Message string `json:"message"`
}

type roleRequest struct {
	RoleID string `json:"role_id"`
	Action string `json:"action"` // "assign" or "remove"
}

type messageRequest struct {
	Message string `json:"message"`
}

type stageRequest struct {
	Stage int `json:"stage"`
}

type approvalActionRequest struct {
	Action string `json:"action"` // "approve" or "deny"
	Reason string `json:"reason,omitempty"`
}

// --- Handlers ---

// handleKickAgent kicks a member from the Discord server.
func (s *Server) handleKickAgent(w http.ResponseWriter, r *http.Request) {
	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var req kickRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if s.discord == nil {
		jsonError(w, "Discord session not available", http.StatusServiceUnavailable)
		return
	}

	// Kick from Discord
	reason := req.Reason
	if reason == "" {
		reason = "Kicked via admin portal"
	}
	if err := s.discord.GuildMemberDeleteWithReason(s.cfg.GuildID, discordID, reason); err != nil {
		log.Printf("API: failed to kick %s: %v", discordID, err)
		jsonError(w, "failed to kick member: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Mark in database
	discordIDInt, _ := strconv.ParseInt(discordID, 10, 64)
	if discordIDInt != 0 {
		s.db.LogActivity(context.Background(), discordIDInt, "kicked_api", "Kicked via admin portal: "+reason)
	}

	// Broadcast event
	if s.hub != nil {
		s.hub.Publish(websocket.NewEvent(websocket.EventAgentKicked, websocket.AgentKickedData{
			DiscordID: discordID,
			Reason:    reason,
			KickedBy:  "admin_portal",
		}))
	}

	jsonOK(w, map[string]string{"status": "kicked", "discord_id": discordID})
}

// handleNudgeAgent sends a DM to the agent.
func (s *Server) handleNudgeAgent(w http.ResponseWriter, r *http.Request) {
	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var req nudgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		jsonError(w, "message is required", http.StatusBadRequest)
		return
	}

	if s.discord == nil {
		jsonError(w, "Discord session not available", http.StatusServiceUnavailable)
		return
	}

	// Open DM channel and send message
	ch, err := s.discord.UserChannelCreate(discordID)
	if err != nil {
		jsonError(w, "failed to create DM channel: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = s.discord.ChannelMessageSend(ch.ID, req.Message)
	if err != nil {
		jsonError(w, "failed to send DM: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Log
	discordIDInt, _ := strconv.ParseInt(discordID, 10, 64)
	if discordIDInt != 0 {
		s.db.LogActivity(context.Background(), discordIDInt, "nudged_api", "Nudged via admin portal")
	}

	if s.hub != nil {
		s.hub.Publish(websocket.NewEvent(websocket.EventAgentNudged, websocket.AgentNudgedData{
			DiscordID: discordID,
			Message:   req.Message,
			NudgedBy:  "admin_portal",
		}))
	}

	jsonOK(w, map[string]string{"status": "nudged", "discord_id": discordID})
}

// handleRoleAgent assigns or removes a Discord role.
func (s *Server) handleRoleAgent(w http.ResponseWriter, r *http.Request) {
	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var req roleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoleID == "" || (req.Action != "assign" && req.Action != "remove") {
		jsonError(w, "role_id and action (assign/remove) required", http.StatusBadRequest)
		return
	}

	if s.discord == nil {
		jsonError(w, "Discord session not available", http.StatusServiceUnavailable)
		return
	}

	var err error
	if req.Action == "assign" {
		err = s.discord.GuildMemberRoleAdd(s.cfg.GuildID, discordID, req.RoleID)
	} else {
		err = s.discord.GuildMemberRoleRemove(s.cfg.GuildID, discordID, req.RoleID)
	}
	if err != nil {
		jsonError(w, "failed to "+req.Action+" role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.hub != nil {
		s.hub.Publish(websocket.NewEvent(websocket.EventRoleAssigned, websocket.RoleAssignedData{
			DiscordID:  discordID,
			RoleID:     req.RoleID,
			Action:     req.Action,
			AssignedBy: "admin_portal",
		}))
	}

	jsonOK(w, map[string]string{"status": req.Action + "ed", "discord_id": discordID, "role_id": req.RoleID})
}

// handleMessageAgent sends a DM to the agent.
func (s *Server) handleMessageAgent(w http.ResponseWriter, r *http.Request) {
	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var req messageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		jsonError(w, "message is required", http.StatusBadRequest)
		return
	}

	if s.discord == nil {
		jsonError(w, "Discord session not available", http.StatusServiceUnavailable)
		return
	}

	ch, err := s.discord.UserChannelCreate(discordID)
	if err != nil {
		jsonError(w, "failed to create DM channel: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = s.discord.ChannelMessageSend(ch.ID, req.Message)
	if err != nil {
		jsonError(w, "failed to send message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if s.hub != nil {
		s.hub.Publish(websocket.NewEvent(websocket.EventMessageSent, websocket.MessageSentData{
			DiscordID: discordID,
			Message:   req.Message,
			SentBy:    "admin_portal",
		}))
	}

	jsonOK(w, map[string]string{"status": "sent", "discord_id": discordID})
}

// handleStageAgent manually sets an agent's pipeline stage.
func (s *Server) handleStageAgent(w http.ResponseWriter, r *http.Request) {
	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	var req stageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Stage < 1 || req.Stage > 8 {
		jsonError(w, "stage must be 1-8", http.StatusBadRequest)
		return
	}

	discordIDInt, err := strconv.ParseInt(discordID, 10, 64)
	if err != nil {
		jsonError(w, "invalid discordID", http.StatusBadRequest)
		return
	}

	guildIDInt, _ := strconv.ParseInt(s.cfg.GuildID, 10, 64)

	s.db.UpsertAgent(context.Background(), discordIDInt, guildIDInt, db.AgentUpdate{
		CurrentStage: &req.Stage,
	})
	s.db.LogActivity(context.Background(), discordIDInt, "stage_changed_api", "Stage set to "+strconv.Itoa(req.Stage)+" via admin portal")

	if s.hub != nil {
		s.hub.Publish(websocket.NewEvent(websocket.EventStageChanged, websocket.StageChangedData{
			DiscordID: discordID,
			NewStage:  req.Stage,
			ChangedBy: "admin_portal",
		}))
	}

	jsonOK(w, map[string]string{"status": "updated", "discord_id": discordID, "stage": strconv.Itoa(req.Stage)})
}

// handleApprovalAction approves or denies an approval request.
func (s *Server) handleApprovalAction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, "invalid approval id", http.StatusBadRequest)
		return
	}

	var req approvalActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Action != "approve" && req.Action != "deny") {
		jsonError(w, "action must be 'approve' or 'deny'", http.StatusBadRequest)
		return
	}

	var dbErr error
	status := "approved"
	if req.Action == "approve" {
		dbErr = s.db.ApproveAgent(context.Background(), id)
	} else {
		status = "denied"
		dbErr = s.db.DenyAgent(context.Background(), id, req.Reason)
	}
	if dbErr != nil {
		jsonError(w, "failed to update approval: "+dbErr.Error(), http.StatusInternalServerError)
		return
	}

	if s.hub != nil {
		s.hub.Publish(websocket.NewEvent(websocket.EventApprovalUpdated, websocket.ApprovalUpdatedData{
			ApprovalID: id,
			Action:     status,
			ActionBy:   "admin_portal",
		}))
	}

	jsonOK(w, map[string]string{"status": status, "approval_id": idStr})
}

// --- Helpers ---

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
