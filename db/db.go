package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
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
	dsn := cfg.DatabaseURL
	// Railway internal Postgres doesn't use SSL; ensure sslmode is set
	if !strings.Contains(dsn, "sslmode=") {
		if strings.Contains(dsn, "?") {
			dsn += "&sslmode=disable"
		} else {
			dsn += "?sslmode=disable"
		}
	}

	log.Printf("Connecting to database...")
	pool, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open failed: %w", err)
	}

	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)

	// Retry connection up to 5 times (Railway services may start before DB is ready)
	var pingErr error
	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		pingErr = pool.PingContext(ctx)
		cancel()
		if pingErr == nil {
			break
		}
		log.Printf("DB ping attempt %d/5 failed: %v", attempt, pingErr)
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
	if pingErr != nil {
		return nil, fmt.Errorf("db: ping failed after 5 attempts: %w", pingErr)
	}
	log.Println("Database connected successfully")

	d := &DB{pool: pool}
	migCtx, migCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer migCancel()
	if err := d.migrate(migCtx); err != nil {
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
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS course_enrolled BOOLEAN DEFAULT FALSE`,

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

		// Phase 2: Recruiter nudge columns
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS upline_manager_discord_id BIGINT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS last_nudge_sent_at TIMESTAMPTZ`,

		// Phase 3: Approval flow columns
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS direct_manager_discord_id BIGINT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS direct_manager_name TEXT`,
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS approval_status TEXT DEFAULT 'none'`,
		`CREATE TABLE IF NOT EXISTS approval_requests (
			id SERIAL PRIMARY KEY,
			agent_discord_id BIGINT NOT NULL,
			guild_id BIGINT NOT NULL,
			agency TEXT NOT NULL,
			owner_discord_id BIGINT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			denial_reason TEXT,
			requested_at TIMESTAMPTZ DEFAULT NOW(),
			responded_at TIMESTAMPTZ,
			dm_message_id TEXT,
			UNIQUE(agent_discord_id, guild_id)
		)`,

		// Phase 4: GHL integration
		`ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS ghl_contact_id TEXT`,

		// Phase 5: Activity entries for leaderboard
		`CREATE TABLE IF NOT EXISTS activity_entries (
			id SERIAL PRIMARY KEY,
			discord_id BIGINT NOT NULL,
			guild_id BIGINT NOT NULL,
			activity_type TEXT NOT NULL,
			count INT NOT NULL DEFAULT 1,
			notes TEXT,
			logged_at TIMESTAMPTZ DEFAULT NOW(),
			week_start DATE NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_entries_week ON activity_entries(discord_id, week_start)`,
		`CREATE INDEX IF NOT EXISTS idx_activity_entries_type ON activity_entries(activity_type, week_start)`,

		// Phase 6: Zoom verticals
		`CREATE TABLE IF NOT EXISTS zoom_verticals (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT,
			zoom_link TEXT,
			schedule TEXT,
			created_by BIGINT,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS zoom_assignments (
			id SERIAL PRIMARY KEY,
			discord_id BIGINT NOT NULL,
			vertical_id INT NOT NULL REFERENCES zoom_verticals(id),
			joined_at TIMESTAMPTZ DEFAULT NOW(),
			UNIQUE(discord_id, vertical_id)
		)`,

		// Phase 7: Portal tables
		`CREATE TABLE IF NOT EXISTS agent_profiles (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			agent_id TEXT NOT NULL UNIQUE,
			bio TEXT,
			city TEXT,
			timezone TEXT,
			linkedin_url TEXT,
			photo_url TEXT,
			start_date DATE,
			comp_tier_id UUID,
			manager_id TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS comp_tiers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL,
			percentage INT NOT NULL DEFAULT 0,
			sort_order INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE constraint_name = 'fk_agent_profiles_comp_tier'
			) THEN
				ALTER TABLE agent_profiles
				ADD CONSTRAINT fk_agent_profiles_comp_tier
				FOREIGN KEY (comp_tier_id) REFERENCES comp_tiers(id) ON DELETE SET NULL;
			END IF;
		END $$`,
		`CREATE TABLE IF NOT EXISTS agent_leads (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			agent_id TEXT NOT NULL,
			first_name TEXT NOT NULL,
			last_name TEXT NOT NULL,
			email TEXT,
			phone TEXT,
			source TEXT,
			status TEXT NOT NULL DEFAULT 'new',
			notes TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS agent_training_items (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			agent_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			due_date DATE,
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS agent_schedule_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			agent_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ NOT NULL,
			type TEXT NOT NULL DEFAULT 'other',
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
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
	CourseEnrolled       *bool
	ContractingBooked    *bool
	ContractingCompleted *bool
	SetupCompleted       *bool
	FormCompletedAt      *time.Time
	SortedAt             *time.Time
	ActivatedAt          *time.Time
	KickedAt                 *time.Time
	KickedReason             *string
	LastActive               *time.Time
	UplineManagerDiscordID   *int64
	LastNudgeSentAt          *time.Time
	DirectManagerDiscordID   *int64
	DirectManagerName        *string
	ApprovalStatus           *string
	GHLContactID             *string
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
	if updates.CourseEnrolled != nil {
		sets = append(sets, fmt.Sprintf("course_enrolled = $%d", argN))
		args = append(args, *updates.CourseEnrolled)
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
	if updates.UplineManagerDiscordID != nil {
		sets = append(sets, fmt.Sprintf("upline_manager_discord_id = $%d", argN))
		args = append(args, *updates.UplineManagerDiscordID)
		argN++
	}
	if updates.LastNudgeSentAt != nil {
		sets = append(sets, fmt.Sprintf("last_nudge_sent_at = $%d", argN))
		args = append(args, *updates.LastNudgeSentAt)
		argN++
	}
	if updates.DirectManagerDiscordID != nil {
		sets = append(sets, fmt.Sprintf("direct_manager_discord_id = $%d", argN))
		args = append(args, *updates.DirectManagerDiscordID)
		argN++
	}
	if updates.DirectManagerName != nil {
		sets = append(sets, fmt.Sprintf("direct_manager_name = $%d", argN))
		args = append(args, *updates.DirectManagerName)
		argN++
	}
	if updates.ApprovalStatus != nil {
		sets = append(sets, fmt.Sprintf("approval_status = $%d", argN))
		args = append(args, *updates.ApprovalStatus)
		argN++
	}
	if updates.GHLContactID != nil {
		sets = append(sets, fmt.Sprintf("ghl_contact_id = $%d", argN))
		args = append(args, *updates.GHLContactID)
		argN++
	}

	if len(args) == 0 {
		return nil // Nothing to update beyond updated_at
	}

	args = append(args, discordID)
	query := fmt.Sprintf("UPDATE onboarding_agents SET %s WHERE discord_id = $%d",
		strings.Join(sets, ", "), argN)

	_, err = d.pool.ExecContext(ctx, query, args...)
	return err
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
	CourseEnrolled       bool
	ContractingBooked    bool
	ContractingCompleted bool
	SetupCompleted       bool
	FormCompletedAt      *time.Time
	SortedAt             *time.Time
	ActivatedAt          *time.Time
	KickedAt             *time.Time
	KickedReason             string
	LastActive               *time.Time
	UplineManagerDiscordID   int64
	LastNudgeSentAt          *time.Time
	DirectManagerDiscordID   int64
	DirectManagerName        string
	ApprovalStatus           string
	GHLContactID             string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// AgentSelectColumns returns the COALESCE'd column list for SELECT queries on onboarding_agents.
// Pass an optional table alias prefix (e.g. "a") or empty string for no prefix.
func AgentSelectColumns(prefix string) string {
	p := ""
	if prefix != "" {
		p = prefix + "."
	}
	return fmt.Sprintf(`%[1]sdiscord_id, %[1]sguild_id,
         COALESCE(%[1]sfirst_name,''), COALESCE(%[1]slast_name,''),
         COALESCE(%[1]sphone_number,''), COALESCE(%[1]semail,''),
         COALESCE(%[1]semail_opt_in, false),
         COALESCE(%[1]sstate,''), COALESCE(%[1]slicense_verified, false),
         COALESCE(%[1]slicense_npn,''), COALESCE(%[1]scurrent_stage, 1),
         COALESCE(%[1]sagency,''), COALESCE(%[1]supline_manager,''),
         COALESCE(%[1]sexperience_level,''), COALESCE(%[1]slicense_status,'none'),
         COALESCE(%[1]sproduction_written,''), COALESCE(%[1]slead_source,''),
         COALESCE(%[1]svision_goals,''), COALESCE(%[1]scomp_pct,''),
         COALESCE(%[1]sshow_comp, false),
         COALESCE(%[1]srole_background,''), COALESCE(%[1]sfun_hobbies,''),
         COALESCE(%[1]snotification_pref,'discord'),
         COALESCE(%[1]scourse_enrolled, false),
         COALESCE(%[1]scontracting_booked, false), COALESCE(%[1]scontracting_completed, false),
         COALESCE(%[1]ssetup_completed, false),
         %[1]sform_completed_at, %[1]ssorted_at, %[1]sactivated_at, %[1]skicked_at,
         COALESCE(%[1]skicked_reason,''),
         %[1]slast_active,
         COALESCE(%[1]supline_manager_discord_id, 0),
         %[1]slast_nudge_sent_at,
         COALESCE(%[1]sdirect_manager_discord_id, 0),
         COALESCE(%[1]sdirect_manager_name, ''),
         COALESCE(%[1]sapproval_status, 'none'),
         COALESCE(%[1]sghl_contact_id, ''),
         %[1]screated_at, %[1]supdated_at`, p)
}

// ScanAgent scans a row into an Agent struct. The column order must match AgentSelectColumns.
func ScanAgent(scan func(dest ...interface{}) error) (Agent, error) {
	var a Agent
	err := scan(
		&a.DiscordID, &a.GuildID, &a.FirstName, &a.LastName,
		&a.PhoneNumber, &a.Email, &a.EmailOptIn, &a.State, &a.LicenseVerified,
		&a.LicenseNPN, &a.CurrentStage,
		&a.Agency, &a.UplineManager,
		&a.ExperienceLevel, &a.LicenseStatus,
		&a.ProductionWritten, &a.LeadSource,
		&a.VisionGoals, &a.CompPct,
		&a.ShowComp,
		&a.RoleBackground, &a.FunHobbies,
		&a.NotificationPref,
		&a.CourseEnrolled,
		&a.ContractingBooked, &a.ContractingCompleted,
		&a.SetupCompleted,
		&a.FormCompletedAt, &a.SortedAt, &a.ActivatedAt, &a.KickedAt,
		&a.KickedReason,
		&a.LastActive,
		&a.UplineManagerDiscordID,
		&a.LastNudgeSentAt,
		&a.DirectManagerDiscordID,
		&a.DirectManagerName,
		&a.ApprovalStatus,
		&a.GHLContactID,
		&a.CreatedAt, &a.UpdatedAt,
	)
	return a, err
}

func (d *DB) GetAgent(ctx context.Context, discordID int64) (*Agent, error) {
	query := fmt.Sprintf(`SELECT %s FROM onboarding_agents WHERE discord_id = $1`, AgentSelectColumns(""))
	row := d.pool.QueryRowContext(ctx, query, discordID)
	a, err := ScanAgent(row.Scan)

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
