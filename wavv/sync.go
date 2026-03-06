package wavv

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// SyncStore defines the database methods the sync service needs.
// Implemented by db.DB.
type SyncStore interface {
	GetWavvToken(ctx context.Context) (string, time.Time, error)
	SaveWavvToken(ctx context.Context, jwt string, expiresAt time.Time) error
	UpsertWavvSyncMember(ctx context.Context, wavvUserID, wavvMemberID, name string, syncAt time.Time) error
	UpsertWavvSyncDayStat(ctx context.Context, wavvMemberID, statDate string, calls, talkSec, convos, appts int, dispositions string) error
}

// SyncService runs background syncs from the WAVV API to the database.
type SyncService struct {
	db       SyncStore
	client   *Client
	interval time.Duration
	stopCh   chan struct{}
}

// NewSyncService creates a new sync service.
func NewSyncService(db SyncStore, syncInterval time.Duration) *SyncService {
	return &SyncService{
		db:       db,
		interval: syncInterval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the background sync loop.
func (s *SyncService) Start() {
	go s.run()
}

// Stop signals the sync loop to exit.
func (s *SyncService) Stop() {
	close(s.stopCh)
}

func (s *SyncService) run() {
	log.Println("[wavv-sync] Starting sync service")

	// Initial sync on startup
	if err := s.initClient(); err != nil {
		log.Printf("[wavv-sync] WARNING: No WAVV token available: %v", err)
		log.Println("[wavv-sync] Use POST /api/v1/wavv/sync/token to set the initial token")
	} else {
		s.doSync()
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.client == nil {
				if err := s.initClient(); err != nil {
					log.Printf("[wavv-sync] Still no token: %v", err)
					continue
				}
			}
			if s.client.IsExpired() {
				log.Println("[wavv-sync] Token expired, need re-auth via POST /api/v1/wavv/sync/token")
				s.client = nil
				continue
			}
			s.doSync()
		case <-s.stopCh:
			log.Println("[wavv-sync] Stopping sync service")
			return
		}
	}
}

func (s *SyncService) initClient() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	jwt, expiresAt, err := s.db.GetWavvToken(ctx)
	if err != nil {
		return fmt.Errorf("load token: %w", err)
	}
	if jwt == "" {
		return fmt.Errorf("no token stored")
	}
	if time.Now().After(expiresAt) {
		return fmt.Errorf("stored token expired at %s", expiresAt.Format(time.RFC3339))
	}

	s.client = NewClient(jwt, expiresAt)
	log.Printf("[wavv-sync] Loaded token (expires %s)", expiresAt.Format("2006-01-02"))
	return nil
}

// SetToken stores a new v1 session JWT and immediately syncs.
func (s *SyncService) SetToken(jwt string, expiresAt time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.db.SaveWavvToken(ctx, jwt, expiresAt); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	s.client = NewClient(jwt, expiresAt)
	log.Printf("[wavv-sync] New token saved (expires %s)", expiresAt.Format("2006-01-02"))

	go s.doSync()
	return nil
}

// SetTokenFromGHLAuth uses a GHL bridge token to obtain a v1 JWT.
func (s *SyncService) SetTokenFromGHLAuth(ghlAuthToken string) error {
	client, authResp, err := Authenticate(ghlAuthToken)
	if err != nil {
		return fmt.Errorf("authenticate: %w", err)
	}
	if authResp.User != nil && authResp.User.NewUser {
		return fmt.Errorf("auth created new user instead of existing — token may be stale")
	}

	s.client = client
	expiresAt := time.Now().Add(30 * 24 * time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.db.SaveWavvToken(ctx, client.JWT(), expiresAt); err != nil {
		return fmt.Errorf("save token: %w", err)
	}
	userName := ""
	if authResp.User != nil {
		userName = authResp.User.Email
	}
	log.Printf("[wavv-sync] Authenticated via GHL token (user: %s, expires %s)",
		userName, expiresAt.Format("2006-01-02"))

	go s.doSync()
	return nil
}

// ForceSync triggers an immediate sync.
func (s *SyncService) ForceSync() (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("no WAVV client — set token first")
	}
	return s.doSync(), nil
}

// TokenStatus returns current token info.
func (s *SyncService) TokenStatus() (hasToken bool, expiresAt time.Time) {
	if s.client == nil {
		return false, time.Time{}
	}
	return true, s.client.expires
}

func (s *SyncService) doSync() string {
	start := time.Now()
	log.Println("[wavv-sync] Starting sync...")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Date range: current week (Monday → Sunday)
	now := time.Now()
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	monday := now.AddDate(0, 0, -(wd - 1))
	sunday := monday.AddDate(0, 0, 6)
	startDate := monday.Format("2006-01-02")
	endDate := sunday.Format("2006-01-02")

	// 1. Sync team members
	membersResp, err := s.client.GetTeamMembers(startDate, endDate)
	if err != nil {
		msg := fmt.Sprintf("error fetching members: %v", err)
		log.Printf("[wavv-sync] %s", msg)
		return msg
	}
	memberCount := 0
	for _, m := range membersResp.Members {
		if err := s.db.UpsertWavvSyncMember(ctx, m.UserID, m.TeamMemberID, m.Name, now); err != nil {
			log.Printf("[wavv-sync] ERROR upserting member %s: %v", m.Name, err)
			continue
		}
		memberCount++
	}

	// 2. Sync day stats
	stats, err := s.client.GetTeamStats(startDate, endDate)
	if err != nil {
		msg := fmt.Sprintf("synced %d members, stats error: %v", memberCount, err)
		log.Printf("[wavv-sync] %s", msg)
		return msg
	}

	statCount := 0
	for memberID, days := range stats.Usage {
		for date, day := range days {
			if day.Calls == 0 && day.Conversations == 0 {
				continue
			}
			appts := countAppointments(day.Dispositions)
			dispJSON := marshalDispositions(day.Dispositions)

			if err := s.db.UpsertWavvSyncDayStat(ctx, memberID, date,
				day.Calls, day.TalkTime, day.Conversations, appts, dispJSON); err != nil {
				log.Printf("[wavv-sync] ERROR stat %s/%s: %v", memberID[:8], date, err)
				continue
			}
			statCount++
		}
	}

	elapsed := time.Since(start)
	summary := fmt.Sprintf("Synced %d members, %d stats in %s (%s → %s)",
		memberCount, statCount, elapsed.Round(time.Millisecond), startDate, endDate)
	log.Printf("[wavv-sync] %s", summary)
	return summary
}

func countAppointments(dispositions map[string]int) int {
	count := 0
	for name, n := range dispositions {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "appointment") || strings.Contains(lower, "appt") {
			count += n
		}
	}
	return count
}

func marshalDispositions(dispositions map[string]int) string {
	if len(dispositions) == 0 {
		return "{}"
	}
	b, err := json.Marshal(dispositions)
	if err != nil {
		return "{}"
	}
	return string(b)
}
