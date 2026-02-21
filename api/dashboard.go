package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type funnelStage struct {
	Stage int    `json:"stage"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type summaryResponse struct {
	TotalActive int `json:"totalActive"`
	TotalKicked int `json:"totalKicked"`
	AtRisk      int `json:"atRisk"`
	Funnel      []funnelStage `json:"funnel"`
}

var stageLabels = map[int]string{
	1: "Joined",
	2: "Form Started",
	3: "Sorted",
	4: "Student",
	5: "Licensed",
	6: "Contracting",
	7: "Setup",
	8: "Active",
}

func (s *Server) handleFunnel(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	counts, err := s.db.GetAgentCounts(ctx)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	var funnel []funnelStage
	for stage := 1; stage <= 8; stage++ {
		funnel = append(funnel, funnelStage{
			Stage: stage,
			Label: stageLabels[stage],
			Count: counts[stage],
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(funnel)
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	counts, err := s.db.GetAgentCounts(ctx)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	kicked, err := s.db.GetKickedCount(ctx)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	// At-risk: inactive for 2+ weeks in stages 1-4
	atRiskThreshold := time.Now().AddDate(0, 0, -14)
	atRiskAgents, err := s.db.GetInactiveAgents(ctx, atRiskThreshold)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	total := 0
	var funnel []funnelStage
	for stage := 1; stage <= 8; stage++ {
		count := counts[stage]
		total += count
		funnel = append(funnel, funnelStage{
			Stage: stage,
			Label: stageLabels[stage],
			Count: count,
		})
	}

	resp := summaryResponse{
		TotalActive: total,
		TotalKicked: kicked,
		AtRisk:      len(atRiskAgents),
		Funnel:      funnel,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAtRisk(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	threshold := time.Now().AddDate(0, 0, -14)
	agents, err := s.db.GetInactiveAgents(ctx, threshold)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	var result []agentResponse
	for _, a := range agents {
		result = append(result, agentResponse{
			DiscordID:       a.DiscordID,
			GuildID:         a.GuildID,
			FirstName:       a.FirstName,
			LastName:        a.LastName,
			State:           a.State,
			Agency:          a.Agency,
			UplineManager:   a.UplineManager,
			ExperienceLevel: a.ExperienceLevel,
			LicenseStatus:   a.LicenseStatus,
			LicenseVerified: a.LicenseVerified,
			CurrentStage:    a.CurrentStage,
			CreatedAt:       a.CreatedAt,
			LastActive:      a.LastActive,
		})
	}

	if result == nil {
		result = []agentResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
