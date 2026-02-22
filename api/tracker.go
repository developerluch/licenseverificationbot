package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type trackerOverviewResponse struct {
	TotalAgents    int     `json:"totalAgents"`
	LicensedAgents int     `json:"licensedAgents"`
	Percentage     float64 `json:"percentage"`
}

type agencyStatsResponse struct {
	Agency         string  `json:"agency"`
	TotalAgents    int     `json:"totalAgents"`
	LicensedAgents int     `json:"licensedAgents"`
	Percentage     float64 `json:"percentage"`
}

type recruiterStatsResponse struct {
	RecruiterName      string `json:"recruiterName"`
	RecruiterDiscordID int64  `json:"recruiterDiscordId"`
	TotalRecruits      int    `json:"totalRecruits"`
	LicensedRecruits   int    `json:"licensedRecruits"`
}

func (s *Server) handleTrackerOverview(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	stats, err := s.db.GetOverallTrackerStats(ctx)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	resp := trackerOverviewResponse{
		TotalAgents:    stats.TotalAgents,
		LicensedAgents: stats.LicensedAgents,
		Percentage:     stats.Percentage,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleTrackerAgencies(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	agencies, err := s.db.GetAgencyTrackerStats(ctx)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	var result []agencyStatsResponse
	for _, a := range agencies {
		result = append(result, agencyStatsResponse{
			Agency:         a.Agency,
			TotalAgents:    a.TotalAgents,
			LicensedAgents: a.LicensedAgents,
			Percentage:     a.Percentage,
		})
	}

	if result == nil {
		result = []agencyStatsResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleTrackerRecruiters(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	agency := r.URL.Query().Get("agency")
	if agency == "" {
		http.Error(w, `{"error":"agency query parameter is required"}`, http.StatusBadRequest)
		return
	}

	recruiters, err := s.db.GetRecruiterTrackerStats(ctx, agency)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	var result []recruiterStatsResponse
	for _, rec := range recruiters {
		result = append(result, recruiterStatsResponse{
			RecruiterName:      rec.RecruiterName,
			RecruiterDiscordID: rec.RecruiterDiscordID,
			TotalRecruits:      rec.TotalRecruits,
			LicensedRecruits:   rec.LicensedRecruits,
		})
	}

	if result == nil {
		result = []recruiterStatsResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
