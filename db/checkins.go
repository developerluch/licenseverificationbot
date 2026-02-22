package db

import (
	"context"
	"fmt"
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
	query := fmt.Sprintf(`SELECT %s FROM onboarding_agents a
         WHERE a.current_stage BETWEEN 1 AND 4
           AND a.kicked_at IS NULL
           AND a.license_status != 'licensed'
           AND NOT EXISTS (
               SELECT 1 FROM agent_weekly_checkins c
               WHERE c.discord_id = a.discord_id AND c.week_start = $1
           )
         ORDER BY a.created_at ASC`, AgentSelectColumns("a"))
	return d.queryAgents(ctx, query, weekStart)
}

// GetInactiveAgents returns agents whose last_active is older than the given threshold.
func (d *DB) GetInactiveAgents(ctx context.Context, threshold time.Time) ([]Agent, error) {
	query := fmt.Sprintf(`SELECT %s FROM onboarding_agents
         WHERE current_stage BETWEEN 1 AND 4
           AND kicked_at IS NULL
           AND (last_active IS NULL OR last_active < $1)
         ORDER BY last_active ASC NULLS FIRST`, AgentSelectColumns(""))
	return d.queryAgents(ctx, query, threshold)
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
