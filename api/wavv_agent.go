package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"license-bot-go/db"
)

// handleWavvAgentStats returns production stats for a single agent.
func (s *Server) handleWavvAgentStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	discordID, err := strconv.ParseInt(r.PathValue("discordID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid discordID"}`, http.StatusBadRequest)
		return
	}

	from, to := parseDateRange(r)
	stats, err := s.db.GetWavvAgentStats(ctx, discordID, from, to)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleWavvAgentSessions returns session history for an agent.
func (s *Server) handleWavvAgentSessions(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	discordID, err := strconv.ParseInt(r.PathValue("discordID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid discordID"}`, http.StatusBadRequest)
		return
	}

	from, to := parseDateRange(r)
	sessions, err := s.db.GetWavvSessions(ctx, discordID, from, to)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		sessions = []db.WavvSession{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

// handleWavvAgentDaily returns daily summaries for an agent.
func (s *Server) handleWavvAgentDaily(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	discordID, err := strconv.ParseInt(r.PathValue("discordID"), 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid discordID"}`, http.StatusBadRequest)
		return
	}

	from, to := parseDateRange(r)
	daily, err := s.db.GetWavvDailySummary(ctx, discordID, from, to)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if daily == nil {
		daily = []db.WavvDailySummary{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(daily)
}
