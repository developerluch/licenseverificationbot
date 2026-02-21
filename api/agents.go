package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type agentResponse struct {
	DiscordID       int64      `json:"discordId"`
	GuildID         int64      `json:"guildId"`
	FirstName       string     `json:"firstName"`
	LastName        string     `json:"lastName"`
	Email           string     `json:"email,omitempty"`
	State           string     `json:"state"`
	Agency          string     `json:"agency"`
	UplineManager   string     `json:"uplineManager"`
	ExperienceLevel string     `json:"experienceLevel"`
	LicenseStatus   string     `json:"licenseStatus"`
	LicenseVerified bool       `json:"licenseVerified"`
	LicenseNPN      string     `json:"licenseNpn,omitempty"`
	CurrentStage    int        `json:"currentStage"`
	CreatedAt       time.Time  `json:"createdAt"`
	ActivatedAt     *time.Time `json:"activatedAt,omitempty"`
	LastActive      *time.Time `json:"lastActive,omitempty"`
}

type agentDetailResponse struct {
	agentResponse
	PhoneNumber          string     `json:"phoneNumber,omitempty"`
	ProductionWritten    string     `json:"productionWritten,omitempty"`
	LeadSource           string     `json:"leadSource,omitempty"`
	VisionGoals          string     `json:"visionGoals,omitempty"`
	RoleBackground       string     `json:"roleBackground,omitempty"`
	FunHobbies           string     `json:"funHobbies,omitempty"`
	ContractingBooked    bool       `json:"contractingBooked"`
	ContractingCompleted bool       `json:"contractingCompleted"`
	SetupCompleted       bool       `json:"setupCompleted"`
	FormCompletedAt      *time.Time `json:"formCompletedAt,omitempty"`
	SortedAt             *time.Time `json:"sortedAt,omitempty"`
	KickedAt             *time.Time `json:"kickedAt,omitempty"`
	KickedReason         string     `json:"kickedReason,omitempty"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query params
	stageStr := r.URL.Query().Get("stage")
	search := r.URL.Query().Get("search")

	var result []agentResponse

	if search != "" {
		agents, err := s.db.SearchAgents(ctx, search)
		if err != nil {
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}
		for _, a := range agents {
			result = append(result, agentResponse{
				DiscordID:       a.DiscordID,
				GuildID:         a.GuildID,
				FirstName:       a.FirstName,
				LastName:        a.LastName,
				Email:           a.Email,
				State:           a.State,
				Agency:          a.Agency,
				UplineManager:   a.UplineManager,
				ExperienceLevel: a.ExperienceLevel,
				LicenseStatus:   a.LicenseStatus,
				LicenseVerified: a.LicenseVerified,
				LicenseNPN:      a.LicenseNPN,
				CurrentStage:    a.CurrentStage,
				CreatedAt:       a.CreatedAt,
				ActivatedAt:     a.ActivatedAt,
				LastActive:      a.LastActive,
			})
		}
	} else if stageStr != "" {
		stage, err := strconv.Atoi(stageStr)
		if err != nil {
			http.Error(w, `{"error":"invalid stage"}`, http.StatusBadRequest)
			return
		}
		agents, err := s.db.GetAgentsByStage(ctx, stage)
		if err != nil {
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}
		for _, a := range agents {
			result = append(result, agentResponse{
				DiscordID:       a.DiscordID,
				GuildID:         a.GuildID,
				FirstName:       a.FirstName,
				LastName:        a.LastName,
				Email:           a.Email,
				State:           a.State,
				Agency:          a.Agency,
				UplineManager:   a.UplineManager,
				ExperienceLevel: a.ExperienceLevel,
				LicenseStatus:   a.LicenseStatus,
				LicenseVerified: a.LicenseVerified,
				LicenseNPN:      a.LicenseNPN,
				CurrentStage:    a.CurrentStage,
				CreatedAt:       a.CreatedAt,
				ActivatedAt:     a.ActivatedAt,
				LastActive:      a.LastActive,
			})
		}
	} else {
		agents, err := s.db.GetAllAgents(ctx, false)
		if err != nil {
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}
		for _, a := range agents {
			result = append(result, agentResponse{
				DiscordID:       a.DiscordID,
				GuildID:         a.GuildID,
				FirstName:       a.FirstName,
				LastName:        a.LastName,
				Email:           a.Email,
				State:           a.State,
				Agency:          a.Agency,
				UplineManager:   a.UplineManager,
				ExperienceLevel: a.ExperienceLevel,
				LicenseStatus:   a.LicenseStatus,
				LicenseVerified: a.LicenseVerified,
				LicenseNPN:      a.LicenseNPN,
				CurrentStage:    a.CurrentStage,
				CreatedAt:       a.CreatedAt,
				ActivatedAt:     a.ActivatedAt,
				LastActive:      a.LastActive,
			})
		}
	}

	if result == nil {
		result = []agentResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleGetAgent(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	discordIDStr := r.PathValue("discordID")
	discordID, err := strconv.ParseInt(discordIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid discord ID"}`, http.StatusBadRequest)
		return
	}

	agent, err := s.db.GetAgent(ctx, discordID)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if agent == nil {
		http.Error(w, `{"error":"agent not found"}`, http.StatusNotFound)
		return
	}

	resp := agentDetailResponse{
		agentResponse: agentResponse{
			DiscordID:       agent.DiscordID,
			GuildID:         agent.GuildID,
			FirstName:       agent.FirstName,
			LastName:        agent.LastName,
			Email:           agent.Email,
			State:           agent.State,
			Agency:          agent.Agency,
			UplineManager:   agent.UplineManager,
			ExperienceLevel: agent.ExperienceLevel,
			LicenseStatus:   agent.LicenseStatus,
			LicenseVerified: agent.LicenseVerified,
			LicenseNPN:      agent.LicenseNPN,
			CurrentStage:    agent.CurrentStage,
			CreatedAt:       agent.CreatedAt,
			ActivatedAt:     agent.ActivatedAt,
			LastActive:      agent.LastActive,
		},
		PhoneNumber:          agent.PhoneNumber,
		ProductionWritten:    agent.ProductionWritten,
		LeadSource:           agent.LeadSource,
		VisionGoals:          agent.VisionGoals,
		RoleBackground:       agent.RoleBackground,
		FunHobbies:           agent.FunHobbies,
		ContractingBooked:    agent.ContractingBooked,
		ContractingCompleted: agent.ContractingCompleted,
		SetupCompleted:       agent.SetupCompleted,
		FormCompletedAt:      agent.FormCompletedAt,
		SortedAt:             agent.SortedAt,
		KickedAt:             agent.KickedAt,
		KickedReason:         agent.KickedReason,
		UpdatedAt:            agent.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
