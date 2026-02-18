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
	CurrentStage    *string
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
	CurrentStage    string
}

func (d *DB) GetAgent(ctx context.Context, discordID int64) (*Agent, error) {
	var a Agent
	err := d.pool.QueryRowContext(ctx,
		`SELECT discord_id, guild_id,
         COALESCE(first_name, ''), COALESCE(last_name, ''),
         COALESCE(phone_number, ''), COALESCE(email, ''),
         COALESCE(email_opt_in, false),
         COALESCE(state, ''), COALESCE(license_verified, false),
         COALESCE(license_npn, ''), COALESCE(current_stage, 'welcome')
         FROM onboarding_agents WHERE discord_id = $1`, discordID,
	).Scan(&a.DiscordID, &a.GuildID, &a.FirstName, &a.LastName,
		&a.PhoneNumber, &a.Email, &a.EmailOptIn, &a.State, &a.LicenseVerified,
		&a.LicenseNPN, &a.CurrentStage)

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
	rows, err := d.pool.QueryContext(ctx,
		`SELECT discord_id, guild_id, COALESCE(first_name,''), COALESCE(last_name,''),
         COALESCE(home_state,''), COALESCE(license_status,'none'), deadline_at,
         COALESCE(auto_verified, false), last_reminder_at,
         COALESCE(admin_notified, false), created_at
         FROM verification_deadlines
         WHERE auto_verified = FALSE
           AND deadline_at > NOW()
           AND (last_reminder_at IS NULL OR last_reminder_at < NOW() - $1::interval)
         ORDER BY deadline_at ASC`, fmt.Sprintf("%d seconds", int(reminderInterval.Seconds())))
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
