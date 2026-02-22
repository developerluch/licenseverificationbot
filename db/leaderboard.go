package db

import (
	"context"
	"time"
)

// ActivityEntry represents a single activity log row.
type ActivityEntry struct {
	ID           int
	DiscordID    int64
	GuildID      int64
	ActivityType string
	Count        int
	Notes        string
	LoggedAt     time.Time
	WeekStart    time.Time
}

// LeaderboardEntry represents a user's rank on the leaderboard.
type LeaderboardEntry struct {
	DiscordID  int64
	TotalCount int
	Rank       int
}

// WeekStart returns the Monday of the current ISO week.
func WeekStart(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())
}

// LogActivityEntry inserts an activity entry.
func (d *DB) LogActivityEntry(ctx context.Context, discordID, guildID int64, activityType string, count int, notes string) error {
	ws := WeekStart(time.Now())
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO activity_entries (discord_id, guild_id, activity_type, count, notes, week_start)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		discordID, guildID, activityType, count, notes, ws)
	return err
}

// GetWeeklyLeaderboard returns the top N users for a given activity type this week.
// If activityType is empty, it aggregates all types.
func (d *DB) GetWeeklyLeaderboard(ctx context.Context, activityType string, limit int) ([]LeaderboardEntry, error) {
	ws := WeekStart(time.Now())
	var query string
	var args []interface{}

	if activityType == "" || activityType == "all" {
		query = `SELECT discord_id, SUM(count) as total
			FROM activity_entries
			WHERE week_start = $1
			GROUP BY discord_id
			ORDER BY total DESC
			LIMIT $2`
		args = []interface{}{ws, limit}
	} else {
		query = `SELECT discord_id, SUM(count) as total
			FROM activity_entries
			WHERE week_start = $1 AND activity_type = $2
			GROUP BY discord_id
			ORDER BY total DESC
			LIMIT $3`
		args = []interface{}{ws, activityType, limit}
	}

	rows, err := d.pool.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.DiscordID, &e.TotalCount); err != nil {
			return nil, err
		}
		e.Rank = rank
		rank++
		result = append(result, e)
	}
	return result, rows.Err()
}

// GetMonthlyLeaderboard returns the top N users aggregated over the current month.
func (d *DB) GetMonthlyLeaderboard(ctx context.Context, activityType string, limit int) ([]LeaderboardEntry, error) {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var query string
	var args []interface{}

	if activityType == "" || activityType == "all" {
		query = `SELECT discord_id, SUM(count) as total
			FROM activity_entries
			WHERE logged_at >= $1
			GROUP BY discord_id
			ORDER BY total DESC
			LIMIT $2`
		args = []interface{}{monthStart, limit}
	} else {
		query = `SELECT discord_id, SUM(count) as total
			FROM activity_entries
			WHERE logged_at >= $1 AND activity_type = $2
			GROUP BY discord_id
			ORDER BY total DESC
			LIMIT $3`
		args = []interface{}{monthStart, activityType, limit}
	}

	rows, err := d.pool.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.DiscordID, &e.TotalCount); err != nil {
			return nil, err
		}
		e.Rank = rank
		rank++
		result = append(result, e)
	}
	return result, rows.Err()
}

// GetAgentWeeklyActivity returns all activity entries for a user this week.
func (d *DB) GetAgentWeeklyActivity(ctx context.Context, discordID int64) (map[string]int, error) {
	ws := WeekStart(time.Now())
	rows, err := d.pool.QueryContext(ctx,
		`SELECT activity_type, SUM(count)
		 FROM activity_entries
		 WHERE discord_id = $1 AND week_start = $2
		 GROUP BY activity_type`, discordID, ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var aType string
		var total int
		if err := rows.Scan(&aType, &total); err != nil {
			return nil, err
		}
		result[aType] = total
	}
	return result, rows.Err()
}
