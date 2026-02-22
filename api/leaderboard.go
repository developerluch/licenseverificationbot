package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type leaderboardEntryResponse struct {
	DiscordID  int64 `json:"discordId"`
	TotalCount int   `json:"totalCount"`
	Rank       int   `json:"rank"`
}

func (s *Server) handleWeeklyLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	activityType := r.URL.Query().Get("type")
	validTypes := map[string]bool{"all": true, "calls": true, "appointments": true, "presentations": true, "policies": true, "recruits": true}
	if !validTypes[activityType] {
		activityType = "all"
	}

	entries, err := s.db.GetWeeklyLeaderboard(ctx, activityType, 10)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	var result []leaderboardEntryResponse
	for _, e := range entries {
		result = append(result, leaderboardEntryResponse{
			DiscordID:  e.DiscordID,
			TotalCount: e.TotalCount,
			Rank:       e.Rank,
		})
	}
	if result == nil {
		result = []leaderboardEntryResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleMonthlyLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	activityType := r.URL.Query().Get("type")
	validTypes := map[string]bool{"all": true, "calls": true, "appointments": true, "presentations": true, "policies": true, "recruits": true}
	if !validTypes[activityType] {
		activityType = "all"
	}

	entries, err := s.db.GetMonthlyLeaderboard(ctx, activityType, 10)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	var result []leaderboardEntryResponse
	for _, e := range entries {
		result = append(result, leaderboardEntryResponse{
			DiscordID:  e.DiscordID,
			TotalCount: e.TotalCount,
			Rank:       e.Rank,
		})
	}
	if result == nil {
		result = []leaderboardEntryResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
