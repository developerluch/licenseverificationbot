package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"license-bot-go/db"
)

// handleWavvGetGoal returns the goal for an agent.
func (s *Server) handleWavvGetGoal(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	discordID, err := strconv.ParseInt(r.PathValue("discordID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid discordID"}`, http.StatusBadRequest)
		return
	}

	goalType := r.URL.Query().Get("type")
	if goalType == "" {
		goalType = "weekly"
	}

	goal, err := s.db.GetWavvGoal(ctx, discordID, goalType)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if goal == nil {
		goal = &db.WavvGoal{DiscordID: discordID, GoalType: goalType}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(goal)
}

// handleWavvSetGoal upserts a production goal.
func (s *Server) handleWavvSetGoal(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	discordID, err := strconv.ParseInt(r.PathValue("discordID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid discordID"}`, http.StatusBadRequest)
		return
	}

	var body struct {
		GoalType     string `json:"goalType"`
		Dials        int    `json:"dials"`
		Connections  int    `json:"connections"`
		TalkMins     int    `json:"talkMins"`
		Appointments int    `json:"appointments"`
		Policies     int    `json:"policies"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if body.GoalType == "" {
		body.GoalType = "weekly"
	}

	goal := db.WavvGoal{
		DiscordID:    discordID,
		GuildID:      s.cfg.GuildIDInt(),
		GoalType:     body.GoalType,
		Dials:        body.Dials,
		Connections:  body.Connections,
		TalkMins:     body.TalkMins,
		Appointments: body.Appointments,
		Policies:     body.Policies,
	}

	if err := s.db.UpsertWavvGoal(ctx, goal); err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleWavvLogSession logs a new dialing session via the API.
func (s *Server) handleWavvLogSession(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	discordID, err := strconv.ParseInt(r.PathValue("discordID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid discordID"}`, http.StatusBadRequest)
		return
	}

	var body struct {
		SessionDate  string `json:"sessionDate"` // YYYY-MM-DD
		Dials        int    `json:"dials"`
		Connections  int    `json:"connections"`
		TalkTimeMins int    `json:"talkTimeMins"`
		Appointments int    `json:"appointments"`
		Callbacks    int    `json:"callbacks"`
		Policies     int    `json:"policies"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	sessionDate := time.Now()
	if body.SessionDate != "" {
		if parsed, err := time.Parse("2006-01-02", body.SessionDate); err == nil {
			sessionDate = parsed
		}
	}

	session := db.WavvSession{
		DiscordID:    discordID,
		GuildID:      s.cfg.GuildIDInt(),
		SessionDate:  sessionDate,
		Dials:        body.Dials,
		Connections:  body.Connections,
		TalkTimeMins: body.TalkTimeMins,
		Appointments: body.Appointments,
		Callbacks:    body.Callbacks,
		Policies:     body.Policies,
		Notes:        body.Notes,
	}

	if err := s.db.LogWavvSession(ctx, session); err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}
