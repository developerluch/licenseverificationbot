package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"license-bot-go/config"
)

// Stage constants for the 8-stage onboarding pipeline.
const (
	StageWelcome     = 1
	StageFormStart   = 2
	StageSorted      = 3
	StageStudent     = 4
	StageVerified    = 5
	StageContracting = 6
	StageSetup       = 7
	StageActive      = 8
)

// StageMap maps legacy text stage values to integer stages.
var StageMap = map[string]int{
	"welcome":     StageWelcome,
	"form_start":  StageFormStart,
	"sorted":      StageSorted,
	"student":     StageStudent,
	"verified":    StageVerified,
	"contracting": StageContracting,
	"setup":       StageSetup,
	"active":      StageActive,
}

type DB struct {
	pool *sql.DB
}

func New(cfg *config.Config) (*DB, error) {
	pool, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: open failed: %w", err)
	}

	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pool.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("db: ping failed: %w", err)
	}

	d := &DB{pool: pool}
	if err := d.migrate(ctx); err != nil {
		return nil, fmt.Errorf("db: migration failed: %w", err)
	}

	log.Println("Database connected and migrated")
	return d, nil
}

func (d *DB) Close() error {
	return d.pool.Close()
}

func (d *DB) migrate(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS license_checks (
            id SERIAL PRIMARY KEY,
            discord_id BIGINT NOT NULL,
            guild_id BIGINT NOT NULL,
            first_name TEXT,
            last_name TEXT,
            state TEXT,
            npn TEXT,
            license_number TEXT,
            license_type TEXT,
            license_status TEXT,
            expiration_date TEXT,
            loas TEXT,
            found BOOLEAN DEFAULT FALSE,
            error TEXT,
            checked_at TIMESTAMPTZ DEFAULT NOW()
        )`,
		`CREATE TABLE IF NOT EXISTS onboarding_agents (
            discord_id BIGINT PRIMARY KEY,
            guild_id BIGINT NOT NULL,
            first_name TEXT,
            last_name TEXT,
            phone_number TEXT,
            email TEXT,
            email_opt_in BOOLEAN DEFAULT FALSE,
            state TEXT,
            license_verified BOOLEAN DEFAULT FALSE,
            license_npn TEXT,
            current_stage TEXT DEFAULT 'welcome',
            created_at TIMESTAMPTZ DEFAULT NOW(),
            updated_at TIMESTAMPTZ DEFAULT NOW()
        )`,
		// Migration: add email_opt_in column if it doesn't exist (for existing DBs)
		`DO $$ BEGIN
            ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS email_opt_in BOOLEAN DEFAULT FALSE;
         EXCEPTION WHEN others THEN NULL;
         END $$`,
		`CREATE TABLE IF NOT EXISTS verification_deadlines (
            discord_id BIGINT PRIMARY KEY,
            guild_id BIGINT NOT NULL,
            first_name TEXT,
            last_name TEXT,
            home_state TEXT,
            license_status TEXT DEFAULT 'none',
            deadline_at TIMESTAMPTZ NOT NULL,
            auto_verified BOOLEAN DEFAULT FALSE,
            last_reminder_at TIMESTAMPTZ,
            admin_notified BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMPTZ DEFAULT NOW()
        )`,

		// Migration: convert current_stage from TEXT to INTEGER
		`DO $$ BEGIN
            IF EXISTS (
                SELECT 1 FROM information_schema.columns
                WHERE table_name='onboarding_agents' AND column_name='current_stage' AND data_type='text'
            ) THEN
                ALTER TABLE onboarding_agents ADD COLUMN current_stage_int INTEGER DEFAULT 1;
                UPDATE onboarding_agents SET current_stage_int = CASE
                    WHEN current_stage='welcome' THEN 1
                    WHEN current_stage='form_start' THEN 2
                    WHEN current_stage='sorted' THEN 3
                    WHEN current_stage='student' THEN 4
                    WHEN current_stage='verified' THEN 5
                    WHEN current_stage='contracting' THEN 6
                    WHEN current_stage='setup' THEN 7
                    WHEN current_stage='active' THEN 8
                    ELSE 1 END;
                ALTER TABLE onboarding_agents DROP COLUMN current_stage;
                ALTER TABLE onboarding_agents RENAME COLUMN current_stage_int TO current_stage;
            END IF;
        END $$`,

		// New columns on onboarding_agents for onboarding pipeline
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS agency TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS upline_manager TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS experience_level TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS license_status TEXT DEFAULT 'none'`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS production_written TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS lead_source TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS vision_goals TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS comp_pct TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS show_comp BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS role_background TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS fun_hobbies TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS notification_pref TEXT DEFAULT 'discord'`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS contracting_booked BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS contracting_completed BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS setup_completed BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS form_completed_at TIMESTAMPTZ`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS sorted_at TIMESTAMPTZ`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS activated_at TIMESTAMPTZ`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS kicked_at TIMESTAMPTZ`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS kicked_reason TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS last_active TIMESTAMPTZ DEFAULT NOW()`,

		// New tables for onboarding pipeline
		`CREATE TABLE IF NOT EXISTS agent_activity_log (
            id SERIAL PRIMARY KEY,
            discord_id BIGINT NOT NULL,
            event_type TEXT NOT NULL,
            details TEXT,
            created_at TIMESTAMPTZ DEFAULT NOW()
        )`,
		`CREATE INDEX IF NOT EXISTS idx_activity_discord ON agent_activity_log(discord_id)`,

		`CREATE TABLE IF NOT EXISTS agent_weekly_checkins (
            id SERIAL PRIMARY KEY,
            discord_id BIGINT NOT NULL,
            week_start DATE NOT NULL,
            sent_at TIMESTAMPTZ,
            response TEXT,
            responded_at TIMESTAMPTZ,
            UNIQUE(discord_id, week_start)
        )`,

		`CREATE TABLE IF NOT EXISTS contracting_managers (
            id SERIAL PRIMARY KEY,
            manager_name TEXT NOT NULL,
            discord_user_id BIGINT,
            calendly_url TEXT NOT NULL,
            priority INTEGER DEFAULT 1,
            is_active BOOLEAN DEFAULT TRUE,
            created_at TIMESTAMPTZ DEFAULT NOW()
        )`,

		`CREATE TABLE IF NOT EXISTS agent_setup_progress (
            id SERIAL PRIMARY KEY,
            discord_id BIGINT NOT NULL,
            item_key TEXT NOT NULL,
            completed BOOLEAN DEFAULT FALSE,
            completed_at TIMESTAMPTZ,
            UNIQUE(discord_id, item_key)
        )`,
	}

	for _, m := range migrations {
		if _, err := d.pool.ExecContext(ctx, m); err != nil {
			return err
		}
	}
	return nil
}

// LicenseCheck represents a row in the license_checks table.
type LicenseCheck struct {
	DiscordID      int64
	GuildID        int64
	FirstName      string
	LastName       string
	State          string
	NPN            string
	LicenseNumber  string
	LicenseType    string
	LicenseStatus  string
	ExpirationDate string
	LOAs           string
	Found          bool
	Error          string
}

func (d *DB) SaveLicenseCheck(ctx context.Context, c LicenseCheck) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO license_checks
         (discord_id, guild_id, first_name, last_name, state, npn, license_number,
          license_type, license_status, expiration_date, loas, found, error)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		c.DiscordID, c.GuildID, c.FirstName, c.LastName, c.State, c.NPN,
		c.LicenseNumber, c.LicenseType, c.LicenseStatus, c.ExpirationDate,
		c.LOAs, c.Found, c.Error,
	)
	return err
}

// AgentUpdate holds fields to update for an agent.
type AgentUpdate struct {
	FirstName       *string
	LastName        *string
	PhoneNumber     *string
	Email           *string
	EmailOptIn      *bool
	State           *string
	LicenseVerified *bool
	LicenseNPN      *string
	CurrentStage    *int

	// Onboarding fields
	Agency               *string
	UplineManager        *string
	ExperienceLevel      *string
	LicenseStatus        *string
	ProductionWritten    *string
	LeadSource           *string
	VisionGoals          *string
	CompPct              *string
	ShowComp             *bool
	RoleBackground       *string
	FunHobbies           *string
	NotificationPref     *string
	ContractingBooked    *bool
	ContractingCompleted *bool
	SetupCompleted       *bool
	FormCompletedAt      *time.Time
	SortedAt             *time.Time
	ActivatedAt          *time.Time
	KickedAt             *time.Time
	KickedReason         *string
	LastActive           *time.Time
}

func (d *DB) UpsertAgent(ctx context.Context, discordID, guildID int64, updates AgentUpdate) error {
	// Insert if not exists
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO onboarding_agents (discord_id, guild_id)
         VALUES ($1, $2)
         ON CONFLICT (discord_id) DO NOTHING`,
		discordID, guildID,
	)
	if err != nil {
		return fmt.Errorf("db: upsert insert: %w", err)
	}

	// Build dynamic UPDATE
	sets := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argN := 1

	if updates.FirstName != nil {
		sets = append(sets, fmt.Sprintf("first_name = $%d", argN))
		args = append(args, *updates.FirstName)
		argN++
	}
	if updates.LastName != nil {
		sets = append(sets, fmt.Sprintf("last_name = $%d", argN))
		args = append(args, *updates.LastName)
		argN++
	}
	if updates.PhoneNumber != nil {
		sets = append(sets, fmt.Sprintf("phone_number = $%d", argN))
		args = append(args, *updates.PhoneNumber)
		argN++
	}
	if updates.Email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", argN))
		args = append(args, *updates.Email)
		argN++
	}
	if updates.EmailOptIn != nil {
		sets = append(sets, fmt.Sprintf("email_opt_in = $%d", argN))
		args = append(args, *updates.EmailOptIn)
		argN++
	}
	if updates.State != nil {
		sets = append(sets, fmt.Sprintf("state = $%d", argN))
		args = append(args, *updates.State)
		argN++
	}
	if updates.LicenseVerified != nil {
		sets = append(sets, fmt.Sprintf("license_verified = $%d", argN))
		args = append(args, *updates.LicenseVerified)
		argN++
	}
	if updates.LicenseNPN != nil {
		sets = append(sets, fmt.Sprintf("license_npn = $%d", argN))
		args = append(args, *updates.LicenseNPN)
		argN++
	}
	if updates.CurrentStage != nil {
		sets = append(sets, fmt.Sprintf("current_stage = $%d", argN))
		args = append(args, *updates.CurrentStage)
		argN++
	}
	if updates.Agency != nil {
		sets = append(sets, fmt.Sprintf("agency = $%d", argN))
		args = append(args, *updates.Agency)
		argN++
	}
	if updates.UplineManager != nil {
		sets = append(sets, fmt.Sprintf("upline_manager = $%d", argN))
		args = append(args, *updates.UplineManager)
		argN++
	}
	if updates.ExperienceLevel != nil {
		sets = append(sets, fmt.Sprintf("experience_level = $%d", argN))
		args = append(args, *updates.ExperienceLevel)
		argN++
	}
	if updates.LicenseStatus != nil {
		sets = append(sets, fmt.Sprintf("license_status = $%d", argN))
		args = append(args, *updates.LicenseStatus)
		argN++
	}
	if updates.ProductionWritten != nil {
		sets = append(sets, fmt.Sprintf("production_written = $%d", argN))
		args = append(args, *updates.ProductionWritten)
		argN++
	}
	if updates.LeadSource != nil {
		sets = append(sets, fmt.Sprintf("lead_source = $%d", argN))
		args = append(args, *updates.LeadSource)
		argN++
	}
	if updates.VisionGoals != nil {
		sets = append(sets, fmt.Sprintf("vision_goals = $%d", argN))
		args = append(args, *updates.VisionGoals)
		argN++
	}
	if updates.CompPct != nil {
		sets = append(sets, fmt.Sprintf("comp_pct = $%d", argN))
		args = append(args, *updates.CompPct)
		argN++
	}
	if updates.ShowComp != nil {
		sets = append(sets, fmt.Sprintf("show_comp = $%d", argN))
		args = append(args, *updates.ShowComp)
		argN++
	}
	if updates.RoleBackground != nil {
		sets = append(sets, fmt.Sprintf("role_background = $%d", argN))
		args = append(args, *updates.RoleBackground)
		argN++
	}
	if updates.FunHobbies != nil {
		sets = append(sets, fmt.Sprintf("fun_hobbies = $%d", argN))
		args = append(args, *updates.FunHobbies)
		argN++
	}
	if updates.NotificationPref != nil {
		sets = append(sets, fmt.Sprintf("notification_pref = $%d", argN))
		args = append(args, *updates.NotificationPref)
		argN++
	}
	if updates.ContractingBooked != nil {
		sets = append(sets, fmt.Sprintf("contracting_booked = $%d", argN))
		args = append(args, *updates.ContractingBooked)
		argN++
	}
	if updates.ContractingCompleted != nil {
		sets = append(sets, fmt.Sprintf("contracting_completed = $%d", argN))
		args = append(args, *updates.ContractingCompleted)
		argN++
	}
	if updates.SetupCompleted != nil {
		sets = append(sets, fmt.Sprintf("setup_completed = $%d", argN))
		args = append(args, *updates.SetupCompleted)
		argN++
	}
	if updates.FormCompletedAt != nil {
		sets = append(sets, fmt.Sprintf("form_completed_at = $%d", argN))
		args = append(args, *updates.FormCompletedAt)
		argN++
	}
	if updates.SortedAt != nil {
		sets = append(sets, fmt.Sprintf("sorted_at = $%d", argN))
		args = append(args, *updates.SortedAt)
		argN++
	}
	if updates.ActivatedAt != nil {
		sets = append(sets, fmt.Sprintf("activated_at = $%d", argN))
		args = append(args, *updates.ActivatedAt)
		argN++
	}
	if updates.KickedAt != nil {
		sets = append(sets, fmt.Sprintf("kicked_at = $%d", argN))
		args = append(args, *updates.KickedAt)
		argN++
	}
	if updates.KickedReason != nil {
		sets = append(sets, fmt.Sprintf("kicked_reason = $%d", argN))
		args = append(args, *updates.KickedReason)
		argN++
	}
	if updates.LastActive != nil {
		sets = append(sets, fmt.Sprintf("last_active = $%d", argN))
		args = append(args, *updates.LastActive)
		argN++
	}

	if len(args) == 0 {
		return nil // Nothing to update beyond updated_at
	}

	args = append(args, discordID)
	query := fmt.Sprintf("UPDATE onboarding_agents SET %s WHERE discord_id = $%d",
		joinStrings(sets, ", "), argN)

	_, err = d.pool.ExecContext(ctx, query, args...)
	return err
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

// Agent represents a row from onboarding_agents
type Agent struct {
	DiscordID       int64
	GuildID         int64
	FirstName       string
	LastName        string
	PhoneNumber     string
	Email           string
	EmailOptIn      bool
	State           string
	LicenseVerified bool
	LicenseNPN      string
	CurrentStage    int

	// Onboarding fields
	Agency               string
	UplineManager        string
	ExperienceLevel      string
	LicenseStatus        string
	ProductionWritten    string
	LeadSource           string
	VisionGoals          string
	CompPct              string
	ShowComp             bool
	RoleBackground       string
	FunHobbies           string
	NotificationPref     string
	ContractingBooked    bool
	ContractingCompleted bool
	SetupCompleted       bool
	FormCompletedAt      *time.Time
	SortedAt             *time.Time
	ActivatedAt          *time.Time
	KickedAt             *time.Time
	KickedReason         string
	LastActive           *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func (d *DB) GetAgent(ctx context.Context, discordID int64) (*Agent, error) {
	var a Agent
	err := d.pool.QueryRowContext(ctx,
		`SELECT discord_id, guild_id,
         COALESCE(first_name, ''), COALESCE(last_name, ''),
         COALESCE(phone_number, ''), COALESCE(email, ''),
         COALESCE(email_opt_in, false),
         COALESCE(state, ''), COALESCE(license_verified, false),
         COALESCE(license_npn, ''), COALESCE(current_stage, 1),
         COALESCE(agency, ''), COALESCE(upline_manager, ''),
         COALESCE(experience_level, ''), COALESCE(license_status, 'none'),
         COALESCE(production_written, ''), COALESCE(lead_source, ''),
         COALESCE(vision_goals, ''), COALESCE(comp_pct, ''),
         COALESCE(show_comp, false),
         COALESCE(role_background, ''), COALESCE(fun_hobbies, ''),
         COALESCE(notification_pref, 'discord'),
         COALESCE(contracting_booked, false), COALESCE(contracting_completed, false),
         COALESCE(setup_completed, false),
         form_completed_at, sorted_at, activated_at, kicked_at,
         COALESCE(kicked_reason, ''),
         last_active, created_at, updated_at
         FROM onboarding_agents WHERE discord_id = $1`, discordID,
	).Scan(&a.DiscordID, &a.GuildID, &a.FirstName, &a.LastName,
		&a.PhoneNumber, &a.Email, &a.EmailOptIn, &a.State, &a.LicenseVerified,
		&a.LicenseNPN, &a.CurrentStage,
		&a.Agency, &a.UplineManager,
		&a.ExperienceLevel, &a.LicenseStatus,
		&a.ProductionWritten, &a.LeadSource,
		&a.VisionGoals, &a.CompPct,
		&a.ShowComp,
		&a.RoleBackground, &a.FunHobbies,
		&a.NotificationPref,
		&a.ContractingBooked, &a.ContractingCompleted,
		&a.SetupCompleted,
		&a.FormCompletedAt, &a.SortedAt, &a.ActivatedAt, &a.KickedAt,
		&a.KickedReason,
		&a.LastActive, &a.CreatedAt, &a.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// VerificationDeadline represents a row in the verification_deadlines table.
type VerificationDeadline struct {
	DiscordID     int64
	GuildID       int64
	FirstName     string
	LastName      string
	HomeState     string
	LicenseStatus string
	DeadlineAt    time.Time
	AutoVerified  bool
	LastReminder  *time.Time
	AdminNotified bool
	CreatedAt     time.Time
}

func (d *DB) CreateDeadline(ctx context.Context, dl VerificationDeadline) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO verification_deadlines
         (discord_id, guild_id, first_name, last_name, home_state, license_status, deadline_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         ON CONFLICT (discord_id) DO UPDATE SET
           guild_id = $2, first_name = $3, last_name = $4, home_state = $5,
           license_status = $6, deadline_at = $7, auto_verified = FALSE,
           admin_notified = FALSE`,
		dl.DiscordID, dl.GuildID, dl.FirstName, dl.LastName,
		dl.HomeState, dl.LicenseStatus, dl.DeadlineAt,
	)
	return err
}

func (d *DB) MarkDeadlineVerified(ctx context.Context, discordID int64) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE verification_deadlines SET auto_verified = TRUE, license_status = 'verified'
         WHERE discord_id = $1`, discordID)
	return err
}

func (d *DB) MarkAdminNotified(ctx context.Context, discordID int64) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE verification_deadlines SET admin_notified = TRUE
         WHERE discord_id = $1`, discordID)
	return err
}

func (d *DB) UpdateReminderSent(ctx context.Context, discordID int64) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE verification_deadlines SET last_reminder_at = NOW()
         WHERE discord_id = $1`, discordID)
	return err
}

func (d *DB) DeleteDeadline(ctx context.Context, discordID int64) error {
	_, err := d.pool.ExecContext(ctx,
		`DELETE FROM verification_deadlines WHERE discord_id = $1`, discordID)
	return err
}

// GetPendingDeadlines returns non-verified deadlines that need reminders.
// It returns deadlines where the last reminder was more than `reminderInterval` ago (or never sent).
func (d *DB) GetPendingDeadlines(ctx context.Context, reminderInterval time.Duration) ([]VerificationDeadline, error) {
	cutoff := time.Now().Add(-reminderInterval)
	rows, err := d.pool.QueryContext(ctx,
		`SELECT discord_id, guild_id, COALESCE(first_name,''), COALESCE(last_name,''),
         COALESCE(home_state,''), COALESCE(license_status,'none'), deadline_at,
         COALESCE(auto_verified, false), last_reminder_at,
         COALESCE(admin_notified, false), created_at
         FROM verification_deadlines
         WHERE auto_verified = FALSE
           AND deadline_at > NOW()
           AND (last_reminder_at IS NULL OR last_reminder_at < $1)
         ORDER BY deadline_at ASC`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []VerificationDeadline
	for rows.Next() {
		var dl VerificationDeadline
		if err := rows.Scan(&dl.DiscordID, &dl.GuildID, &dl.FirstName, &dl.LastName,
			&dl.HomeState, &dl.LicenseStatus, &dl.DeadlineAt, &dl.AutoVerified,
			&dl.LastReminder, &dl.AdminNotified, &dl.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, dl)
	}
	return result, rows.Err()
}

// GetExpiredDeadlines returns deadlines that have passed without verification.
func (d *DB) GetExpiredDeadlines(ctx context.Context) ([]VerificationDeadline, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT discord_id, guild_id, COALESCE(first_name,''), COALESCE(last_name,''),
         COALESCE(home_state,''), COALESCE(license_status,'none'), deadline_at,
         COALESCE(auto_verified, false), last_reminder_at,
         COALESCE(admin_notified, false), created_at
         FROM verification_deadlines
         WHERE auto_verified = FALSE
           AND admin_notified = FALSE
           AND deadline_at <= NOW()
         ORDER BY deadline_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []VerificationDeadline
	for rows.Next() {
		var dl VerificationDeadline
		if err := rows.Scan(&dl.DiscordID, &dl.GuildID, &dl.FirstName, &dl.LastName,
			&dl.HomeState, &dl.LicenseStatus, &dl.DeadlineAt, &dl.AutoVerified,
			&dl.LastReminder, &dl.AdminNotified, &dl.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, dl)
	}
	return result, rows.Err()
}

// CheckHistoryRow represents a license check history row.
type CheckHistoryRow struct {
	State         string
	LicenseStatus string
	Found         bool
	Error         string
	CheckedAt     time.Time
}

func (d *DB) GetCheckHistory(ctx context.Context, discordID int64, limit int) ([]CheckHistoryRow, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT COALESCE(state, ''), COALESCE(license_status, ''),
         COALESCE(found, false), COALESCE(error, ''), checked_at
         FROM license_checks
         WHERE discord_id = $1
         ORDER BY checked_at DESC
         LIMIT $2`, discordID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CheckHistoryRow
	for rows.Next() {
		var r CheckHistoryRow
		if err := rows.Scan(&r.State, &r.LicenseStatus, &r.Found, &r.Error, &r.CheckedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
