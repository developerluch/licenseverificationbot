package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"license-bot-go/db"
	"license-bot-go/wavv"
)

// wavvSync holds the reference to the sync service, set from main.
var wavvSync *wavv.SyncService

// SetWavvSync sets the sync service reference for API handlers.
func SetWavvSync(svc *wavv.SyncService) {
	wavvSync = svc
}

// handleWavvSyncSetToken stores a v1 JWT directly.
// POST /api/v1/wavv/sync/token  {"jwt": "...", "expiresAt": "2026-04-05T..."}
func (s *Server) handleWavvSyncSetToken(w http.ResponseWriter, r *http.Request) {
	if wavvSync == nil {
		http.Error(w, `{"error":"sync service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	var body struct {
		JWT       string `json:"jwt"`
		ExpiresAt string `json:"expiresAt,omitempty"`
		// Alternative: provide a GHL auth token to auto-authenticate
		GHLToken string `json:"ghlToken,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Option 1: GHL auth token → auto-authenticate
	if body.GHLToken != "" {
		if err := wavvSync.SetTokenFromGHLAuth(body.GHLToken); err != nil {
			log.Printf("[api] WAVV GHL auth failed: %v", err)
			jsonResp(w, map[string]string{"error": "GHL auth failed: " + err.Error()}, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"method":  "ghl_auth",
			"message": "Token set via GHL authentication, sync started",
		})
		return
	}

	// Option 2: Direct JWT
	if body.JWT == "" {
		http.Error(w, `{"error":"jwt or ghlToken required"}`, http.StatusBadRequest)
		return
	}

	expiresAt := time.Now().Add(30 * 24 * time.Hour) // default 30 days
	if body.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, body.ExpiresAt); err == nil {
			expiresAt = t
		}
	}

	if err := wavvSync.SetToken(body.JWT, expiresAt); err != nil {
		log.Printf("[api] WAVV set token failed: %v", err)
		jsonResp(w, map[string]string{"error": "set token: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        true,
		"method":    "direct_jwt",
		"expiresAt": expiresAt.Format(time.RFC3339),
		"message":   "Token saved, sync started",
	})
}

// handleWavvSyncForce triggers an immediate sync.
// POST /api/v1/wavv/sync/force
func (s *Server) handleWavvSyncForce(w http.ResponseWriter, r *http.Request) {
	if wavvSync == nil {
		http.Error(w, `{"error":"sync service not initialized"}`, http.StatusServiceUnavailable)
		return
	}

	result, err := wavvSync.ForceSync()
	if err != nil {
		jsonResp(w, map[string]string{"error": err.Error()}, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"summary": result,
	})
}

// handleWavvSyncStatus returns current sync status.
// GET /api/v1/wavv/sync/status
func (s *Server) handleWavvSyncStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Token status
	hasToken := false
	var tokenExpires time.Time
	if wavvSync != nil {
		hasToken, tokenExpires = wavvSync.TokenStatus()
	}

	// DB stats
	memberCount, statCount, lastSync, _ := s.db.GetWavvSyncStats(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hasToken":      hasToken,
		"tokenExpires":  tokenExpires.Format(time.RFC3339),
		"memberCount":   memberCount,
		"statCount":     statCount,
		"lastSync":      lastSync.Format(time.RFC3339),
		"syncAvailable": wavvSync != nil,
	})
}

// handleWavvSyncLeaderboard returns the synced leaderboard data.
// GET /api/v1/wavv/sync/leaderboard?from=2026-03-01&to=2026-03-07
func (s *Server) handleWavvSyncLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	from, to := parseDateRange(r)
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	entries, err := s.db.GetWavvTeamLeaderboard(ctx, fromStr, toStr)
	if err != nil {
		log.Printf("[api] WAVV sync leaderboard error: %v", err)
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if entries == nil {
		entries = []db.WavvLeaderboardEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

// handleWavvSyncMembers returns all synced WAVV members.
// GET /api/v1/wavv/sync/members
func (s *Server) handleWavvSyncMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	members, err := s.db.GetAllWavvMembers(ctx)
	if err != nil {
		log.Printf("[api] WAVV sync members error: %v", err)
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	if members == nil {
		members = []db.WavvMemberRecord{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// handleWavvSyncLinkMember links a WAVV member to a Discord user.
// POST /api/v1/wavv/sync/members/link  {"wavvMemberId": "...", "discordId": 123456}
func (s *Server) handleWavvSyncLinkMember(w http.ResponseWriter, r *http.Request) {
	var body struct {
		WavvMemberID string `json:"wavvMemberId"`
		DiscordID    int64  `json:"discordId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if body.WavvMemberID == "" || body.DiscordID == 0 {
		http.Error(w, `{"error":"wavvMemberId and discordId required"}`, http.StatusBadRequest)
		return
	}

	if err := s.db.LinkWavvMember(r.Context(), body.WavvMemberID, body.DiscordID); err != nil {
		log.Printf("[api] WAVV link member error: %v", err)
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"message": "Member linked",
	})
}

// jsonResp writes a JSON response with a given status code.
func jsonResp(w http.ResponseWriter, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
