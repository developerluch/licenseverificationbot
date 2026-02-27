package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"license-bot-go/api/websocket"
	"license-bot-go/db"
)

// handleDiscordMemberLookup looks up a Discord member by ID.
// GET /api/v1/discord/members/{discordID}
func (s *Server) handleDiscordMemberLookup(w http.ResponseWriter, r *http.Request) {
	discordID := r.PathValue("discordID")
	if discordID == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	member, err := s.discord.GuildMember(s.cfg.GuildID, discordID)
	if err != nil || member == nil || member.User == nil {
		jsonOK(w, map[string]interface{}{
			"exists": false,
		})
		return
	}

	avatarURL := ""
	if member.User.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", member.User.ID, member.User.Avatar)
	}

	displayName := member.Nick
	if displayName == "" {
		displayName = member.User.GlobalName
	}
	if displayName == "" {
		displayName = member.User.Username
	}

	jsonOK(w, map[string]interface{}{
		"exists":      true,
		"discordId":   member.User.ID,
		"username":    member.User.Username,
		"displayName": displayName,
		"avatarUrl":   avatarURL,
		"roles":       member.Roles,
		"joinedAt":    member.JoinedAt.Format(time.RFC3339),
	})
}

// handleCreateAgent creates a new agent from the dashboard.
// POST /api/v1/agents
func (s *Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var req struct {
		DiscordID string `json:"discordId"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		State     string `json:"state"`
		Agency    string `json:"agency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.DiscordID == "" || req.FirstName == "" || req.LastName == "" {
		jsonError(w, "discordId, firstName, and lastName are required", http.StatusBadRequest)
		return
	}

	// Validate member exists in guild
	member, err := s.discord.GuildMember(s.cfg.GuildID, req.DiscordID)
	if err != nil || member == nil {
		jsonError(w, "Discord member not found in this server", http.StatusBadRequest)
		return
	}

	discordIDInt, err := strconv.ParseInt(req.DiscordID, 10, 64)
	if err != nil {
		jsonError(w, "invalid discordId", http.StatusBadRequest)
		return
	}
	guildIDInt, _ := strconv.ParseInt(s.cfg.GuildID, 10, 64)

	// Check if agent already exists
	existing, _ := s.db.GetAgent(ctx, discordIDInt)
	if existing != nil {
		jsonError(w, "agent already exists", http.StatusConflict)
		return
	}

	// Create agent
	stage := db.StageWelcome
	update := db.AgentUpdate{
		FirstName:    &req.FirstName,
		LastName:     &req.LastName,
		CurrentStage: &stage,
	}
	if req.State != "" {
		update.State = &req.State
	}
	if req.Agency != "" {
		update.Agency = &req.Agency
	}

	if err := s.db.UpsertAgent(ctx, discordIDInt, guildIDInt, update); err != nil {
		log.Printf("API: failed to create agent %s: %v", req.DiscordID, err)
		jsonError(w, "failed to create agent", http.StatusInternalServerError)
		return
	}

	// Create profile
	s.db.GetOrCreateProfile(ctx, req.DiscordID)

	// Broadcast event
	avatarURL := ""
	if member.User != nil && member.User.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", member.User.ID, member.User.Avatar)
	}
	username := ""
	if member.User != nil {
		username = member.User.Username
	}
	displayName := member.Nick
	if displayName == "" && member.User != nil {
		displayName = member.User.GlobalName
	}
	if displayName == "" {
		displayName = username
	}

	if s.hub != nil {
		s.hub.Publish(websocket.NewEvent(websocket.EventAgentCreated, websocket.AgentCreatedData{
			DiscordID:   req.DiscordID,
			Username:    username,
			DisplayName: displayName,
			AvatarURL:   avatarURL,
			Stage:       stage,
			CreatedBy:   "admin_portal",
		}))
	}

	// Fetch agent back and return
	agent, err := s.db.GetAgent(ctx, discordIDInt)
	if err != nil || agent == nil {
		jsonError(w, "agent created but could not fetch", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)
}

// handleBulkImport imports all non-bot Discord members as agents.
// POST /api/v1/agents/bulk-import
func (s *Server) handleBulkImport(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	guildIDInt, _ := strconv.ParseInt(s.cfg.GuildID, 10, 64)

	imported := 0
	skipped := 0
	bots := 0
	afterID := ""

	for {
		members, err := s.discord.GuildMembers(s.cfg.GuildID, afterID, 1000)
		if err != nil {
			log.Printf("API: bulk import GuildMembers error: %v", err)
			jsonError(w, "failed to fetch guild members: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for _, member := range members {
			if member.User == nil {
				continue
			}
			if member.User.Bot {
				bots++
				continue
			}

			discordIDInt, err := strconv.ParseInt(member.User.ID, 10, 64)
			if err != nil {
				continue
			}

			existing, _ := s.db.GetAgent(ctx, discordIDInt)
			if existing != nil {
				skipped++
				continue
			}

			// Create agent with basic info
			stage := db.StageWelcome
			firstName := member.User.GlobalName
			if firstName == "" {
				firstName = member.User.Username
			}
			lastName := ""

			s.db.UpsertAgent(ctx, discordIDInt, guildIDInt, db.AgentUpdate{
				FirstName:    &firstName,
				LastName:     &lastName,
				CurrentStage: &stage,
			})
			s.db.GetOrCreateProfile(ctx, member.User.ID)
			imported++
		}

		if len(members) < 1000 {
			break
		}
		afterID = members[len(members)-1].User.ID
	}

	total := imported + skipped + bots

	// Broadcast event
	if s.hub != nil {
		s.hub.Publish(websocket.NewEvent(websocket.EventBulkImport, websocket.BulkImportData{
			Imported: imported,
			Skipped:  skipped,
			Total:    total,
		}))
	}

	jsonOK(w, map[string]interface{}{
		"imported": imported,
		"skipped":  skipped,
		"bots":     bots,
		"total":    total,
	})
}
