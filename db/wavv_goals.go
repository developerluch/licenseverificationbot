package db

import (
	"context"
	"database/sql"
	"time"
)

// UpsertWavvGoal creates or updates a production goal for an agent.
func (d *DB) UpsertWavvGoal(ctx context.Context, g WavvGoal) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO wavv_goals
		 (discord_id, guild_id, goal_type, dials, connections, talk_mins, appointments, policies)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (discord_id, goal_type) DO UPDATE SET
		   dials=$4, connections=$5, talk_mins=$6, appointments=$7, policies=$8,
		   updated_at=NOW()`,
		g.DiscordID, g.GuildID, g.GoalType, g.Dials, g.Connections,
		g.TalkMins, g.Appointments, g.Policies)
	return err
}

// GetWavvGoal returns the goal for an agent by type (weekly/monthly).
func (d *DB) GetWavvGoal(ctx context.Context, discordID int64, goalType string) (*WavvGoal, error) {
	var g WavvGoal
	err := d.pool.QueryRowContext(ctx,
		`SELECT id, discord_id, guild_id, goal_type, dials, connections,
		        talk_mins, appointments, policies, created_at, updated_at
		 FROM wavv_goals
		 WHERE discord_id = $1 AND goal_type = $2`, discordID, goalType).Scan(
		&g.ID, &g.DiscordID, &g.GuildID, &g.GoalType, &g.Dials,
		&g.Connections, &g.TalkMins, &g.Appointments, &g.Policies,
		&g.CreatedAt, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// GetWavvOverview returns a high-level summary across all agents for a date range.
func (d *DB) GetWavvOverview(ctx context.Context, from, to time.Time) (map[string]interface{}, error) {
	var totalDials, totalConnections, totalTalk, totalAppts, totalPolicies, agentCount int
	err := d.pool.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(dials),0), COALESCE(SUM(connections),0),
		        COALESCE(SUM(talk_time_mins),0), COALESCE(SUM(appointments),0),
		        COALESCE(SUM(policies),0), COUNT(DISTINCT discord_id)
		 FROM wavv_sessions
		 WHERE session_date >= $1 AND session_date <= $2`, from, to).Scan(
		&totalDials, &totalConnections, &totalTalk, &totalAppts, &totalPolicies, &agentCount)
	if err != nil {
		return nil, err
	}

	connectRate := 0.0
	if totalDials > 0 {
		connectRate = float64(totalConnections) / float64(totalDials) * 100
	}

	return map[string]interface{}{
		"totalDials":       totalDials,
		"totalConnections": totalConnections,
		"totalTalkMins":    totalTalk,
		"totalAppointments": totalAppts,
		"totalPolicies":    totalPolicies,
		"activeAgents":     agentCount,
		"connectRate":      connectRate,
	}, nil
}
