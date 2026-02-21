package db

import (
	"context"
	"time"
)

// CheckinRecord represents a row from agent_weekly_checkins.
type CheckinRecord struct {
	DiscordID   int64
	WeekStart   string
	SentAt      *time.Time
	Response    string
	RespondedAt *time.Time
}

// GetStudentsForCheckin returns students in stages 1-4 who haven't been checked in this week.
func (d *DB) GetStudentsForCheckin(ctx context.Context, weekStart time.Time) ([]Agent, error) {
	return d.queryAgents(ctx,
		`SELECT a.discord_id, a.guild_id,
         COALESCE(a.first_name,''), COALESCE(a.last_name,''),
         COALESCE(a.phone_number,''), COALESCE(a.email,''),
         COALESCE(a.email_opt_in, false),
         COALESCE(a.state,''), COALESCE(a.license_verified, false),
         COALESCE(a.license_npn,''), COALESCE(a.current_stage, 1),
         COALESCE(a.agency,''), COALESCE(a.upline_manager,''),
         COALESCE(a.experience_level,''), COALESCE(a.license_status,'none'),
         COALESCE(a.production_written,''), COALESCE(a.lead_source,''),
         COALESCE(a.vision_goals,''), COALESCE(a.comp_pct,''),
         COALESCE(a.show_comp, false),
         COALESCE(a.role_background,''), COALESCE(a.fun_hobbies,''),
         COALESCE(a.notification_pref,'discord'),
         COALESCE(a.contracting_booked, false), COALESCE(a.contracting_completed, false),
         COALESCE(a.setup_completed, false),
         a.form_completed_at, a.sorted_at, a.activated_at, a.kicked_at,
         COALESCE(a.kicked_reason,''),
         a.last_active, a.created_at, a.updated_at
         FROM onboarding_agents a
         WHERE a.current_stage BETWEEN 1 AND 4
           AND a.kicked_at IS NULL
           AND a.license_status != 'licensed'
           AND NOT EXISTS (
               SELECT 1 FROM agent_weekly_checkins c
               WHERE c.discord_id = a.discord_id AND c.week_start = $1
           )
         ORDER BY a.created_at ASC`, weekStart)
}

// GetInactiveAgents returns agents whose last_active is older than the given threshold.
func (d *DB) GetInactiveAgents(ctx context.Context, threshold time.Time) ([]Agent, error) {
	return d.queryAgents(ctx,
		`SELECT discord_id, guild_id,
         COALESCE(first_name,''), COALESCE(last_name,''),
         COALESCE(phone_number,''), COALESCE(email,''),
         COALESCE(email_opt_in, false),
         COALESCE(state,''), COALESCE(license_verified, false),
         COALESCE(license_npn,''), COALESCE(current_stage, 1),
         COALESCE(agency,''), COALESCE(upline_manager,''),
         COALESCE(experience_level,''), COALESCE(license_status,'none'),
         COALESCE(production_written,''), COALESCE(lead_source,''),
         COALESCE(vision_goals,''), COALESCE(comp_pct,''),
         COALESCE(show_comp, false),
         COALESCE(role_background,''), COALESCE(fun_hobbies,''),
         COALESCE(notification_pref,'discord'),
         COALESCE(contracting_booked, false), COALESCE(contracting_completed, false),
         COALESCE(setup_completed, false),
         form_completed_at, sorted_at, activated_at, kicked_at,
         COALESCE(kicked_reason,''),
         last_active, created_at, updated_at
         FROM onboarding_agents
         WHERE current_stage BETWEEN 1 AND 4
           AND kicked_at IS NULL
           AND (last_active IS NULL OR last_active < $1)
         ORDER BY last_active ASC NULLS FIRST`, threshold)
}

// RecordCheckinSent records that a weekly check-in DM was sent.
func (d *DB) RecordCheckinSent(ctx context.Context, discordID int64, weekStart time.Time) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO agent_weekly_checkins (discord_id, week_start, sent_at)
         VALUES ($1, $2, NOW())
         ON CONFLICT (discord_id, week_start) DO UPDATE SET sent_at = NOW()`,
		discordID, weekStart)
	return err
}

// RecordCheckinResponse records an agent's response to a weekly check-in.
func (d *DB) RecordCheckinResponse(ctx context.Context, discordID int64, weekStart, response string) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO agent_weekly_checkins (discord_id, week_start, response, responded_at)
         VALUES ($1, $2, $3, NOW())
         ON CONFLICT (discord_id, week_start)
         DO UPDATE SET response = $3, responded_at = NOW()`,
		discordID, weekStart, response)
	return err
}

// GetCheckinHistory returns recent check-in records for an agent.
func (d *DB) GetCheckinHistory(ctx context.Context, discordID int64, limit int) ([]CheckinRecord, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT discord_id, week_start, sent_at, COALESCE(response,''), responded_at
         FROM agent_weekly_checkins
         WHERE discord_id = $1
         ORDER BY week_start DESC
         LIMIT $2`, discordID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CheckinRecord
	for rows.Next() {
		var r CheckinRecord
		if err := rows.Scan(&r.DiscordID, &r.WeekStart, &r.SentAt, &r.Response, &r.RespondedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
