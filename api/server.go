package api

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	gorillaws "github.com/gorilla/websocket"

	"license-bot-go/api/websocket"
	"license-bot-go/config"
	"license-bot-go/db"
	"license-bot-go/scrapers"
)

// WebSocket upgrader
var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // CORS handled at middleware level
	},
}

// Server is the REST API server with WebSocket support.
type Server struct {
	cfg      *config.Config
	db       *db.DB
	discord  *discordgo.Session
	hub      *websocket.Hub
	registry *scrapers.Registry
	srv      *http.Server
}

// NewServer creates a new API server with WebSocket hub and Discord session.
func NewServer(cfg *config.Config, database *db.DB, discord *discordgo.Session, hub *websocket.Hub, registry *scrapers.Registry) *Server {
	s := &Server{cfg: cfg, db: database, discord: discord, hub: hub, registry: registry}

	mux := http.NewServeMux()

	// Health check (unauthenticated)
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	// WebSocket endpoint (token via query param)
	mux.HandleFunc("GET /api/v1/ws", s.handleWebSocket)

	// Authenticated routes
	auth := s.authMiddleware

	// Agent endpoints (read)
	mux.HandleFunc("GET /api/v1/agents", auth(s.handleListAgents))
	mux.HandleFunc("GET /api/v1/agents/{discordID}", auth(s.handleGetAgent))

	// Dashboard endpoints (read)
	mux.HandleFunc("GET /api/v1/dashboard/funnel", auth(s.handleFunnel))
	mux.HandleFunc("GET /api/v1/dashboard/summary", auth(s.handleSummary))
	mux.HandleFunc("GET /api/v1/dashboard/at-risk", auth(s.handleAtRisk))

	// Tracker endpoints (read)
	mux.HandleFunc("GET /api/v1/tracker/overview", auth(s.handleTrackerOverview))
	mux.HandleFunc("GET /api/v1/tracker/agencies", auth(s.handleTrackerAgencies))
	mux.HandleFunc("GET /api/v1/tracker/recruiters", auth(s.handleTrackerRecruiters))

	// Leaderboard endpoints (read)
	mux.HandleFunc("GET /api/v1/leaderboard/weekly", auth(s.handleWeeklyLeaderboard))
	mux.HandleFunc("GET /api/v1/leaderboard/monthly", auth(s.handleMonthlyLeaderboard))

	// Approval endpoints (read + write)
	mux.HandleFunc("GET /api/v1/approvals/{id}", auth(s.handleGetApproval))
	mux.HandleFunc("PUT /api/v1/approvals/{id}", auth(s.handleApprovalAction))

	// Admin action endpoints (write)
	mux.HandleFunc("POST /api/v1/agents/{discordID}/kick", auth(s.handleKickAgent))
	mux.HandleFunc("POST /api/v1/agents/{discordID}/nudge", auth(s.handleNudgeAgent))
	mux.HandleFunc("POST /api/v1/agents/{discordID}/role", auth(s.handleRoleAgent))
	mux.HandleFunc("POST /api/v1/agents/{discordID}/message", auth(s.handleMessageAgent))
	mux.HandleFunc("POST /api/v1/agents/{discordID}/stage", auth(s.handleStageAgent))

	// Portal: Profile
	mux.HandleFunc("GET /api/v1/portal/profile/{discordID}", auth(s.handleGetProfile))
	mux.HandleFunc("PUT /api/v1/portal/profile/{discordID}", auth(s.handleUpdateProfile))
	mux.HandleFunc("PUT /api/v1/portal/profile/{discordID}/manager", auth(s.handleSetManager))
	mux.HandleFunc("PUT /api/v1/portal/profile/{discordID}/comp-tier", auth(s.handleSetCompTier))

	// Portal: Leads
	mux.HandleFunc("GET /api/v1/portal/agents/{discordID}/leads", auth(s.handleListAgentLeads))
	mux.HandleFunc("POST /api/v1/portal/agents/{discordID}/leads", auth(s.handleCreateAgentLead))
	mux.HandleFunc("PUT /api/v1/portal/agents/{discordID}/leads/{leadID}", auth(s.handleUpdateAgentLead))
	mux.HandleFunc("DELETE /api/v1/portal/agents/{discordID}/leads/{leadID}", auth(s.handleDeleteAgentLead))

	// Portal: Training
	mux.HandleFunc("GET /api/v1/portal/agents/{discordID}/training", auth(s.handleListTrainingItems))
	mux.HandleFunc("POST /api/v1/portal/agents/{discordID}/training", auth(s.handleCreateTrainingItem))
	mux.HandleFunc("PUT /api/v1/portal/agents/{discordID}/training/{itemID}", auth(s.handleUpdateTrainingItem))
	mux.HandleFunc("DELETE /api/v1/portal/agents/{discordID}/training/{itemID}", auth(s.handleDeleteTrainingItem))

	// Portal: Schedule
	mux.HandleFunc("GET /api/v1/portal/agents/{discordID}/schedule", auth(s.handleListScheduleEvents))
	mux.HandleFunc("POST /api/v1/portal/agents/{discordID}/schedule", auth(s.handleCreateScheduleEvent))
	mux.HandleFunc("PUT /api/v1/portal/agents/{discordID}/schedule/{eventID}", auth(s.handleUpdateScheduleEvent))
	mux.HandleFunc("DELETE /api/v1/portal/agents/{discordID}/schedule/{eventID}", auth(s.handleDeleteScheduleEvent))

	// Portal: Org Chart
	mux.HandleFunc("GET /api/v1/portal/org-chart", auth(s.handleGetOrgChart))
	mux.HandleFunc("PUT /api/v1/portal/org-chart/assign-manager", auth(s.handleAssignManager))

	// Portal: Comp Tiers
	mux.HandleFunc("GET /api/v1/portal/comp-tiers", auth(s.handleListCompTiers))
	mux.HandleFunc("POST /api/v1/portal/comp-tiers", auth(s.handleCreateCompTier))
	mux.HandleFunc("PUT /api/v1/portal/comp-tiers/{tierID}", auth(s.handleUpdateCompTier))
	mux.HandleFunc("DELETE /api/v1/portal/comp-tiers/{tierID}", auth(s.handleDeleteCompTier))
	mux.HandleFunc("PUT /api/v1/portal/comp-tiers/reorder", auth(s.handleReorderCompTiers))

	// Discord lookup
	mux.HandleFunc("GET /api/v1/discord/members/{discordID}", auth(s.handleDiscordMemberLookup))

	// Agent creation + bulk import
	mux.HandleFunc("POST /api/v1/agents", auth(s.handleCreateAgent))
	mux.HandleFunc("POST /api/v1/agents/bulk-import", auth(s.handleBulkImport))

	// Dashboard-triggered verification
	mux.HandleFunc("POST /api/v1/agents/{discordID}/verify", auth(s.handleVerifyAgent))

	handler := s.corsMiddleware(mux)

	s.srv = &http.Server{
		Addr:              ":" + cfg.APIPort,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s
}

// Hub returns the WebSocket hub for external event publishing.
func (s *Server) Hub() *websocket.Hub {
	return s.hub
}

// handleWebSocket upgrades the HTTP connection to a WebSocket.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authenticate via query param ?token=...
	token := r.URL.Query().Get("token")
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.APIToken)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := websocket.NewClient(s.hub, conn)
	go client.WritePump()
	go client.ReadPump()
}

// Start runs the API server. Call in a goroutine.
func (s *Server) Start(ctx context.Context) {
	log.Printf("API server starting on :%s", s.cfg.APIPort)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.srv.Shutdown(shutdownCtx)
	}()

	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("API server error: %v", err)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
