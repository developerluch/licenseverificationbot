package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"license-bot-go/config"
	"license-bot-go/db"
)

// Server is the REST API server.
type Server struct {
	cfg *config.Config
	db  *db.DB
	srv *http.Server
}

// NewServer creates a new API server.
func NewServer(cfg *config.Config, database *db.DB) *Server {
	s := &Server{cfg: cfg, db: database}

	mux := http.NewServeMux()

	// Health check (unauthenticated)
	mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	// Authenticated routes
	auth := s.authMiddleware

	// Agent endpoints
	mux.HandleFunc("GET /api/v1/agents", auth(s.handleListAgents))
	mux.HandleFunc("GET /api/v1/agents/{discordID}", auth(s.handleGetAgent))

	// Dashboard endpoints
	mux.HandleFunc("GET /api/v1/dashboard/funnel", auth(s.handleFunnel))
	mux.HandleFunc("GET /api/v1/dashboard/summary", auth(s.handleSummary))
	mux.HandleFunc("GET /api/v1/dashboard/at-risk", auth(s.handleAtRisk))

	// Tracker endpoints
	mux.HandleFunc("GET /api/v1/tracker/overview", auth(s.handleTrackerOverview))
	mux.HandleFunc("GET /api/v1/tracker/agencies", auth(s.handleTrackerAgencies))
	mux.HandleFunc("GET /api/v1/tracker/recruiters", auth(s.handleTrackerRecruiters))

	// Leaderboard endpoints
	mux.HandleFunc("GET /api/v1/leaderboard/weekly", auth(s.handleWeeklyLeaderboard))
	mux.HandleFunc("GET /api/v1/leaderboard/monthly", auth(s.handleMonthlyLeaderboard))

	// Approval endpoints
	mux.HandleFunc("GET /api/v1/approvals/{id}", auth(s.handleGetApproval))

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
