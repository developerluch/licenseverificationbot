package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type approvalResponse struct {
	ID             int    `json:"id"`
	AgentDiscordID int64  `json:"agentDiscordId"`
	Agency         string `json:"agency"`
	OwnerDiscordID int64  `json:"ownerDiscordId"`
	Status         string `json:"status"`
	DenialReason   string `json:"denialReason,omitempty"`
	RequestedAt    string `json:"requestedAt"`
}

func (s *Server) handleGetApproval(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	req, err := s.db.GetApprovalRequest(ctx, id)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if req == nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	resp := approvalResponse{
		ID:             req.ID,
		AgentDiscordID: req.AgentDiscordID,
		Agency:         req.Agency,
		OwnerDiscordID: req.OwnerDiscordID,
		Status:         req.Status,
		DenialReason:   req.DenialReason,
		RequestedAt:    req.RequestedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
