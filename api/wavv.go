package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"license-bot-go/db"
)

// handleWavvOverview returns high-level WAVV production stats.
func (s *Server) handleWavvOverview(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	from, to := parseDateRange(r)
	overview, err := s.db.GetWavvOverview(ctx, from, to)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

// handleWavvTeams returns production rollups grouped by agency.
func (s *Server) handleWavvTeams(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	from, to := parseDateRange(r)
	teams, err := s.db.GetWavvTeamSummary(ctx, from, to)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if teams == nil {
		teams = []db.WavvTeamSummary{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(teams)
}

// handleWavvLeaderboard returns the agent leaderboard.
func (s *Server) handleWavvLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	from, to := parseDateRange(r)
	board, err := s.db.GetWavvAgentLeaderboard(ctx, from, to)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if board == nil {
		board = []db.WavvAgentRollup{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(board)
}
