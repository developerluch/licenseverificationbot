package db

import (
	"context"
	"time"
)

// WavvSession represents a single dialing session logged by an agent.
type WavvSession struct {
	ID           int       `json:"id"`
	DiscordID    int64     `json:"discordId"`
	GuildID      int64     `json:"guildId"`
	SessionDate  time.Time `json:"sessionDate"`
	Dials        int       `json:"dials"`
	Connections  int       `json:"connections"`
	TalkTimeMins int       `json:"talkTimeMins"`
	Appointments int       `json:"appointments"`
	Callbacks    int       `json:"callbacks"`
	Policies     int       `json:"policies"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"createdAt"`
}

// WavvGoal represents a weekly or monthly production goal for an agent.
type WavvGoal struct {
	ID          int       `json:"id"`
	DiscordID   int64     `json:"discordId"`
	GuildID     int64     `json:"guildId"`
	GoalType    string    `json:"goalType"` // "weekly" or "monthly"
	Dials       int       `json:"dials"`
	Connections int       `json:"connections"`
	TalkMins    int       `json:"talkMins"`
	Appointments int      `json:"appointments"`
	Policies    int       `json:"policies"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// WavvDailySummary is a precomputed daily rollup for a single agent.
type WavvDailySummary struct {
	SessionDate  string `json:"sessionDate"`
	Dials        int    `json:"dials"`
	Connections  int    `json:"connections"`
	TalkTimeMins int    `json:"talkTimeMins"`
	Appointments int    `json:"appointments"`
	Callbacks    int    `json:"callbacks"`
	Policies     int    `json:"policies"`
}

// WavvAgentRollup is a summary of production metrics for one agent.
type WavvAgentRollup struct {
	DiscordID    int64  `json:"discordId"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Agency       string `json:"agency"`
	Dials        int    `json:"dials"`
	Connections  int    `json:"connections"`
	TalkTimeMins int    `json:"talkTimeMins"`
	Appointments int    `json:"appointments"`
	Callbacks    int    `json:"callbacks"`
	Policies     int    `json:"policies"`
	SessionCount int    `json:"sessionCount"`
}

// WavvTeamSummary holds aggregate production for a team/agency.
type WavvTeamSummary struct {
	Agency       string `json:"agency"`
	AgentCount   int    `json:"agentCount"`
	Dials        int    `json:"dials"`
	Connections  int    `json:"connections"`
	TalkTimeMins int    `json:"talkTimeMins"`
	Appointments int    `json:"appointments"`
	Policies     int    `json:"policies"`
}

// LogWavvSession inserts a new dialing session record.
func (d *DB) LogWavvSession(ctx context.Context, s WavvSession) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO wavv_sessions
		 (discord_id, guild_id, session_date, dials, connections, talk_time_mins,
		  appointments, callbacks, policies, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		s.DiscordID, s.GuildID, s.SessionDate, s.Dials, s.Connections,
		s.TalkTimeMins, s.Appointments, s.Callbacks, s.Policies, s.Notes)
	return err
}

// GetWavvSessions returns sessions for an agent within a date range.
func (d *DB) GetWavvSessions(ctx context.Context, discordID int64, from, to time.Time) ([]WavvSession, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT id, discord_id, guild_id, session_date, dials, connections,
		        talk_time_mins, appointments, callbacks, policies, COALESCE(notes,''), created_at
		 FROM wavv_sessions
		 WHERE discord_id = $1 AND session_date >= $2 AND session_date <= $3
		 ORDER BY session_date DESC`, discordID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WavvSession
	for rows.Next() {
		var ws WavvSession
		if err := rows.Scan(&ws.ID, &ws.DiscordID, &ws.GuildID, &ws.SessionDate,
			&ws.Dials, &ws.Connections, &ws.TalkTimeMins, &ws.Appointments,
			&ws.Callbacks, &ws.Policies, &ws.Notes, &ws.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, ws)
	}
	return result, rows.Err()
}

// GetWavvDailySummary returns daily rollups for an agent in a date range.
func (d *DB) GetWavvDailySummary(ctx context.Context, discordID int64, from, to time.Time) ([]WavvDailySummary, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT session_date::TEXT,
		        SUM(dials), SUM(connections), SUM(talk_time_mins),
		        SUM(appointments), SUM(callbacks), SUM(policies)
		 FROM wavv_sessions
		 WHERE discord_id = $1 AND session_date >= $2 AND session_date <= $3
		 GROUP BY session_date
		 ORDER BY session_date DESC`, discordID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WavvDailySummary
	for rows.Next() {
		var ds WavvDailySummary
		if err := rows.Scan(&ds.SessionDate, &ds.Dials, &ds.Connections,
			&ds.TalkTimeMins, &ds.Appointments, &ds.Callbacks, &ds.Policies); err != nil {
			return nil, err
		}
		result = append(result, ds)
	}
	return result, rows.Err()
}
