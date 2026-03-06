package db

import (
	"context"
	"database/sql"
	"time"
)

// --- Token storage ---

// GetWavvToken returns the stored WAVV JWT and its expiry.
func (d *DB) GetWavvToken(ctx context.Context) (string, time.Time, error) {
	var jwt string
	var expiresAt time.Time
	err := d.pool.QueryRowContext(ctx,
		`SELECT token, expires_at FROM wavv_api_tokens ORDER BY id DESC LIMIT 1`,
	).Scan(&jwt, &expiresAt)
	if err == sql.ErrNoRows {
		return "", time.Time{}, nil
	}
	return jwt, expiresAt, err
}

// SaveWavvToken inserts or replaces the WAVV API token.
func (d *DB) SaveWavvToken(ctx context.Context, jwt string, expiresAt time.Time) error {
	// Delete old tokens and insert new one
	_, _ = d.pool.ExecContext(ctx, `DELETE FROM wavv_api_tokens`)
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO wavv_api_tokens (token, expires_at) VALUES ($1, $2)`,
		jwt, expiresAt)
	return err
}

// --- Member mapping ---

// UpsertWavvMember inserts or updates a WAVV team member record.
func (d *DB) UpsertWavvMember(ctx context.Context, m WavvMemberRecord) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO wavv_members (wavv_user_id, wavv_member_id, name, last_sync_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (wavv_member_id)
		 DO UPDATE SET name = EXCLUDED.name, wavv_user_id = EXCLUDED.wavv_user_id, last_sync_at = EXCLUDED.last_sync_at`,
		m.WavvUserID, m.WavvMemberID, m.Name, m.LastSyncAt)
	return err
}

// WavvMemberRecord is the DB representation of a WAVV team member.
type WavvMemberRecord struct {
	ID           int       `json:"id"`
	WavvUserID   string    `json:"wavvUserId"`
	WavvMemberID string    `json:"wavvMemberId"`
	Name         string    `json:"name"`
	DiscordID    int64     `json:"discordId"`
	LastSyncAt   time.Time `json:"lastSyncAt"`
}

// GetWavvMemberByWavvID looks up a member by their WAVV member ID.
func (d *DB) GetWavvMemberByWavvID(ctx context.Context, wavvMemberID string) (*WavvMemberRecord, error) {
	var m WavvMemberRecord
	err := d.pool.QueryRowContext(ctx,
		`SELECT id, wavv_user_id, wavv_member_id, name, COALESCE(discord_id, 0), last_sync_at
		 FROM wavv_members WHERE wavv_member_id = $1`, wavvMemberID,
	).Scan(&m.ID, &m.WavvUserID, &m.WavvMemberID, &m.Name, &m.DiscordID, &m.LastSyncAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &m, err
}

// LinkWavvMember associates a WAVV member with a Discord user.
func (d *DB) LinkWavvMember(ctx context.Context, wavvMemberID string, discordID int64) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE wavv_members SET discord_id = $1 WHERE wavv_member_id = $2`,
		discordID, wavvMemberID)
	return err
}

// GetAllWavvMembers returns all WAVV team members.
func (d *DB) GetAllWavvMembers(ctx context.Context) ([]WavvMemberRecord, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT id, wavv_user_id, wavv_member_id, name, COALESCE(discord_id, 0), last_sync_at
		 FROM wavv_members ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WavvMemberRecord
	for rows.Next() {
		var m WavvMemberRecord
		if err := rows.Scan(&m.ID, &m.WavvUserID, &m.WavvMemberID, &m.Name, &m.DiscordID, &m.LastSyncAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// --- Day stats ---

// WavvDayStatRecord is the DB representation of a daily stat row.
type WavvDayStatRecord struct {
	ID            int    `json:"id"`
	WavvMemberID  string `json:"wavvMemberId"`
	StatDate      string `json:"statDate"`
	Calls         int    `json:"calls"`
	TalkTimeSec   int    `json:"talkTimeSec"`
	Conversations int    `json:"conversations"`
	Appointments  int    `json:"appointments"`
	Dispositions  string `json:"dispositions"`
}

// UpsertWavvDayStat inserts or updates a daily stat for a WAVV member.
func (d *DB) UpsertWavvDayStat(ctx context.Context, s WavvDayStatRecord) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO wavv_day_stats (wavv_member_id, stat_date, calls, talk_time_sec, conversations, appointments, dispositions)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (wavv_member_id, stat_date)
		 DO UPDATE SET calls = EXCLUDED.calls, talk_time_sec = EXCLUDED.talk_time_sec,
		              conversations = EXCLUDED.conversations, appointments = EXCLUDED.appointments,
		              dispositions = EXCLUDED.dispositions`,
		s.WavvMemberID, s.StatDate, s.Calls, s.TalkTimeSec, s.Conversations, s.Appointments, s.Dispositions)
	return err
}

// GetWavvTeamLeaderboard returns aggregated stats for all members in a date range,
// joined with member names and optional Discord IDs.
func (d *DB) GetWavvTeamLeaderboard(ctx context.Context, from, to string) ([]WavvLeaderboardEntry, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT m.name, COALESCE(m.discord_id, 0),
		        SUM(s.calls), SUM(s.talk_time_sec), SUM(s.conversations), SUM(s.appointments)
		 FROM wavv_day_stats s
		 JOIN wavv_members m ON m.wavv_member_id = s.wavv_member_id
		 WHERE s.stat_date >= $1 AND s.stat_date <= $2
		 GROUP BY m.wavv_member_id, m.name, m.discord_id
		 HAVING SUM(s.calls) > 0
		 ORDER BY SUM(s.calls) DESC`, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []WavvLeaderboardEntry
	for rows.Next() {
		var e WavvLeaderboardEntry
		if err := rows.Scan(&e.Name, &e.DiscordID, &e.Calls, &e.TalkTimeSec, &e.Conversations, &e.Appointments); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// WavvLeaderboardEntry is a row in the WAVV leaderboard.
type WavvLeaderboardEntry struct {
	Name          string `json:"name"`
	DiscordID     int64  `json:"discordId"`
	Calls         int    `json:"calls"`
	TalkTimeSec   int    `json:"talkTimeSec"`
	Conversations int    `json:"conversations"`
	Appointments  int    `json:"appointments"`
}

// GetWavvSyncStats returns the latest sync info.
func (d *DB) GetWavvSyncStats(ctx context.Context) (memberCount int, statCount int, lastSync time.Time, err error) {
	_ = d.pool.QueryRowContext(ctx, `SELECT COUNT(*) FROM wavv_members`).Scan(&memberCount)
	_ = d.pool.QueryRowContext(ctx, `SELECT COUNT(*) FROM wavv_day_stats`).Scan(&statCount)
	_ = d.pool.QueryRowContext(ctx, `SELECT COALESCE(MAX(last_sync_at), '1970-01-01') FROM wavv_members`).Scan(&lastSync)
	return
}

// --- SyncStore interface adapters ---
// These methods satisfy the wavv.SyncStore interface with flat parameter signatures.

// UpsertWavvSyncMember satisfies wavv.SyncStore — upserts a member with individual params.
func (d *DB) UpsertWavvSyncMember(ctx context.Context, wavvUserID, wavvMemberID, name string, syncAt time.Time) error {
	return d.UpsertWavvMember(ctx, WavvMemberRecord{
		WavvUserID:   wavvUserID,
		WavvMemberID: wavvMemberID,
		Name:         name,
		LastSyncAt:   syncAt,
	})
}

// UpsertWavvSyncDayStat satisfies wavv.SyncStore — upserts a day stat with individual params.
func (d *DB) UpsertWavvSyncDayStat(ctx context.Context, wavvMemberID, statDate string, calls, talkSec, convos, appts int, dispositions string) error {
	return d.UpsertWavvDayStat(ctx, WavvDayStatRecord{
		WavvMemberID:  wavvMemberID,
		StatDate:      statDate,
		Calls:         calls,
		TalkTimeSec:   talkSec,
		Conversations: convos,
		Appointments:  appts,
		Dispositions:  dispositions,
	})
}
