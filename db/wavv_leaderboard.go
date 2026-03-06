package db

import (
	"context"
	"time"
)

// GetWavvAgentLeaderboard returns production rollups for all agents in a date range.
func (d *DB) GetWavvAgentLeaderboard(ctx context.Context, from, to time.Time) ([]WavvAgentRollup, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT ws.discord_id,
		        COALESCE(oa.first_name, ''), COALESCE(oa.last_name, ''), COALESCE(oa.agency, ''),
		        SUM(ws.dials), SUM(ws.connections), SUM(ws.talk_time_mins),
		        SUM(ws.appointments), SUM(ws.callbacks), SUM(ws.policies),
		        COUNT(ws.id)
		 FROM wavv_sessions ws
		 LEFT JOIN onboarding_agents oa ON ws.discord_id = oa.discord_id
		 WHERE ws.session_date >= $1 AND ws.session_date <= $2
		 GROUP BY ws.discord_id, oa.first_name, oa.last_name, oa.agency
		 ORDER BY SUM(ws.dials) DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WavvAgentRollup
	for rows.Next() {
		var r WavvAgentRollup
		if err := rows.Scan(&r.DiscordID, &r.FirstName, &r.LastName, &r.Agency,
			&r.Dials, &r.Connections, &r.TalkTimeMins, &r.Appointments,
			&r.Callbacks, &r.Policies, &r.SessionCount); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// GetWavvTeamSummary returns production rollups grouped by agency.
func (d *DB) GetWavvTeamSummary(ctx context.Context, from, to time.Time) ([]WavvTeamSummary, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT COALESCE(oa.agency, 'Unknown'),
		        COUNT(DISTINCT ws.discord_id),
		        SUM(ws.dials), SUM(ws.connections), SUM(ws.talk_time_mins),
		        SUM(ws.appointments), SUM(ws.policies)
		 FROM wavv_sessions ws
		 LEFT JOIN onboarding_agents oa ON ws.discord_id = oa.discord_id
		 WHERE ws.session_date >= $1 AND ws.session_date <= $2
		 GROUP BY oa.agency
		 ORDER BY SUM(ws.dials) DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WavvTeamSummary
	for rows.Next() {
		var t WavvTeamSummary
		if err := rows.Scan(&t.Agency, &t.AgentCount, &t.Dials, &t.Connections,
			&t.TalkTimeMins, &t.Appointments, &t.Policies); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// GetWavvAgentStats returns production stats for a single agent in a date range.
func (d *DB) GetWavvAgentStats(ctx context.Context, discordID int64, from, to time.Time) (*WavvAgentRollup, error) {
	var r WavvAgentRollup
	err := d.pool.QueryRowContext(ctx,
		`SELECT ws.discord_id,
		        COALESCE(oa.first_name, ''), COALESCE(oa.last_name, ''), COALESCE(oa.agency, ''),
		        COALESCE(SUM(ws.dials),0), COALESCE(SUM(ws.connections),0), COALESCE(SUM(ws.talk_time_mins),0),
		        COALESCE(SUM(ws.appointments),0), COALESCE(SUM(ws.callbacks),0), COALESCE(SUM(ws.policies),0),
		        COUNT(ws.id)
		 FROM wavv_sessions ws
		 LEFT JOIN onboarding_agents oa ON ws.discord_id = oa.discord_id
		 WHERE ws.discord_id = $1 AND ws.session_date >= $2 AND ws.session_date <= $3
		 GROUP BY ws.discord_id, oa.first_name, oa.last_name, oa.agency`,
		discordID, from, to).Scan(
		&r.DiscordID, &r.FirstName, &r.LastName, &r.Agency,
		&r.Dials, &r.Connections, &r.TalkTimeMins, &r.Appointments,
		&r.Callbacks, &r.Policies, &r.SessionCount)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
