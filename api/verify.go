package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"license-bot-go/api/websocket"
	"license-bot-go/db"
)

// handleVerifyAgent triggers license verification from the dashboard.
// POST /api/v1/agents/{discordID}/verify
func (s *Server) handleVerifyAgent(w http.ResponseWriter, r *http.Request) {
	discordIDStr := r.PathValue("discordID")
	if discordIDStr == "" {
		jsonError(w, "missing discordID", http.StatusBadRequest)
		return
	}

	discordIDInt, err := strconv.ParseInt(discordIDStr, 10, 64)
	if err != nil {
		jsonError(w, "invalid discordID", http.StatusBadRequest)
		return
	}
	guildIDInt, _ := strconv.ParseInt(s.cfg.GuildID, 10, 64)

	// Parse optional body
	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		State     string `json:"state"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// If body fields missing, load from DB
	if req.FirstName == "" || req.LastName == "" || req.State == "" {
		agent, _ := s.db.GetAgent(r.Context(), discordIDInt)
		if agent != nil {
			if req.FirstName == "" {
				req.FirstName = agent.FirstName
			}
			if req.LastName == "" {
				req.LastName = agent.LastName
			}
			if req.State == "" {
				req.State = agent.State
			}
		}
	}

	req.State = strings.ToUpper(strings.TrimSpace(req.State))

	if req.FirstName == "" || req.LastName == "" || req.State == "" {
		jsonError(w, "firstName, lastName, and state are required (provide in body or ensure agent has them stored)", http.StatusBadRequest)
		return
	}

	if s.registry == nil {
		jsonError(w, "scraper registry not available", http.StatusServiceUnavailable)
		return
	}

	// Get scraper and perform lookup
	scraper := s.registry.GetScraper(req.State)

	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	results, err := scraper.LookupByName(ctx, req.FirstName, req.LastName)
	if err != nil {
		log.Printf("API verify: scraper error for %s: %v", req.State, err)
		jsonError(w, fmt.Sprintf("license lookup error for %s: %v", req.State, err), http.StatusInternalServerError)
		return
	}

	// 2-pass match: prefer life-licensed active, then any active
	var matchIdx = -1
	for idx := range results {
		if results[idx].Found && results[idx].Active && results[idx].IsLifeLicensed() {
			matchIdx = idx
			break
		}
	}
	if matchIdx < 0 {
		for idx := range results {
			if results[idx].Found && results[idx].Active {
				matchIdx = idx
				break
			}
		}
	}

	if matchIdx >= 0 {
		match := &results[matchIdx]

		// Update agent
		verified := true
		stage := db.StageVerified
		s.db.UpsertAgent(ctx, discordIDInt, guildIDInt, db.AgentUpdate{
			FirstName:       &req.FirstName,
			LastName:        &req.LastName,
			State:           &req.State,
			LicenseVerified: &verified,
			LicenseNPN:      &match.NPN,
			CurrentStage:    &stage,
		})

		// Save license check
		s.db.SaveLicenseCheck(ctx, db.LicenseCheck{
			DiscordID:      discordIDInt,
			GuildID:        guildIDInt,
			FirstName:      req.FirstName,
			LastName:       req.LastName,
			State:          req.State,
			NPN:            match.NPN,
			LicenseNumber:  match.LicenseNumber,
			LicenseType:    match.LicenseType,
			LicenseStatus:  match.Status,
			ExpirationDate: match.ExpirationDate,
			LOAs:           match.LOAs,
			Found:          true,
		})

		// Assign roles
		if s.cfg.LicensedAgentRoleID != "" {
			s.discord.GuildMemberRoleAdd(s.cfg.GuildID, discordIDStr, s.cfg.LicensedAgentRoleID)
		}
		if s.cfg.StudentRoleID != "" {
			s.discord.GuildMemberRoleRemove(s.cfg.GuildID, discordIDStr, s.cfg.StudentRoleID)
		}

		// Mark deadline verified
		s.db.MarkDeadlineVerified(ctx, discordIDInt)

		// Broadcast events
		if s.hub != nil {
			s.hub.Publish(websocket.NewEvent(websocket.EventLicenseVerified, websocket.LicenseVerifiedData{
				DiscordID: discordIDStr,
				State:     req.State,
				FullName:  req.FirstName + " " + req.LastName,
				NPN:       match.NPN,
				Status:    "verified",
			}))
			s.hub.Publish(websocket.NewEvent(websocket.EventStageChanged, websocket.StageChangedData{
				DiscordID: discordIDStr,
				NewStage:  db.StageVerified,
				ChangedBy: "admin_portal",
			}))
		}

		jsonOK(w, map[string]interface{}{
			"verified":       true,
			"npn":            match.NPN,
			"licenseNumber":  match.LicenseNumber,
			"licenseType":    match.LicenseType,
			"status":         match.Status,
			"expirationDate": match.ExpirationDate,
			"loas":           match.LOAs,
		})
		return
	}

	// No match found
	errMsg := fmt.Sprintf("No active license found for %s %s in %s", req.FirstName, req.LastName, req.State)
	if len(results) > 0 && results[0].Error != "" {
		errMsg = results[0].Error
	}
	jsonOK(w, map[string]interface{}{
		"verified": false,
		"error":    errMsg,
	})
}
