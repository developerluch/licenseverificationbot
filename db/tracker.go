package db

import (
	"context"
	"fmt"
)

// TrackerStats holds overall license verification statistics.
type TrackerStats struct {
	TotalAgents    int
	LicensedAgents int
	Percentage     float64
}

// AgencyStats holds license verification statistics per agency.
type AgencyStats struct {
	Agency         string
	TotalAgents    int
	LicensedAgents int
	Percentage     float64
}

// RecruiterStats holds license verification statistics per recruiter.
type RecruiterStats struct {
	RecruiterName      string
	RecruiterDiscordID int64
	TotalRecruits      int
	LicensedRecruits   int
}

// GetOverallTrackerStats returns the total and licensed agent counts (excluding kicked agents).
func (d *DB) GetOverallTrackerStats(ctx context.Context) (TrackerStats, error) {
	var stats TrackerStats
	err := d.pool.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN license_verified THEN 1 ELSE 0 END), 0)
		 FROM onboarding_agents
		 WHERE kicked_at IS NULL`).Scan(&stats.TotalAgents, &stats.LicensedAgents)
	if err != nil {
		return stats, err
	}
	if stats.TotalAgents > 0 {
		stats.Percentage = float64(stats.LicensedAgents) / float64(stats.TotalAgents) * 100
	}
	return stats, nil
}

// GetAgencyTrackerStats returns license verification statistics grouped by agency.
func (d *DB) GetAgencyTrackerStats(ctx context.Context) ([]AgencyStats, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT COALESCE(agency, ''),
		        COUNT(*),
		        COALESCE(SUM(CASE WHEN license_verified THEN 1 ELSE 0 END), 0)
		 FROM onboarding_agents
		 WHERE kicked_at IS NULL AND agency != ''
		 GROUP BY agency
		 ORDER BY agency`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgencyStats
	for rows.Next() {
		var s AgencyStats
		if err := rows.Scan(&s.Agency, &s.TotalAgents, &s.LicensedAgents); err != nil {
			return nil, err
		}
		if s.TotalAgents > 0 {
			s.Percentage = float64(s.LicensedAgents) / float64(s.TotalAgents) * 100
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// GetRecruiterTrackerStats returns license verification statistics grouped by recruiter for a given agency.
func (d *DB) GetRecruiterTrackerStats(ctx context.Context, agency string) ([]RecruiterStats, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT COALESCE(upline_manager, ''),
		        COALESCE(upline_manager_discord_id, 0),
		        COUNT(*),
		        COALESCE(SUM(CASE WHEN license_verified THEN 1 ELSE 0 END), 0)
		 FROM onboarding_agents
		 WHERE kicked_at IS NULL AND agency = $1 AND upline_manager != ''
		 GROUP BY upline_manager, upline_manager_discord_id
		 ORDER BY upline_manager`, agency)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []RecruiterStats
	for rows.Next() {
		var s RecruiterStats
		if err := rows.Scan(&s.RecruiterName, &s.RecruiterDiscordID, &s.TotalRecruits, &s.LicensedRecruits); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// GetAgentsNeedingNudge returns agents who have not verified their license and are
// past the nudge threshold, with a cooldown on repeat nudges.
func (d *DB) GetAgentsNeedingNudge(ctx context.Context, nudgeAfterDays, nudgeCooldownDays int) ([]Agent, error) {
	query := fmt.Sprintf(`SELECT %s FROM onboarding_agents
		WHERE license_verified = false
		  AND kicked_at IS NULL
		  AND created_at < NOW() - $1 * interval '1 day'
		  AND (last_nudge_sent_at IS NULL OR last_nudge_sent_at < NOW() - $2 * interval '1 day')
		ORDER BY created_at ASC`, AgentSelectColumns(""))
	return d.queryAgents(ctx, query, nudgeAfterDays, nudgeCooldownDays)
}

// UpdateNudgeSent marks the last nudge time for an agent.
func (d *DB) UpdateNudgeSent(ctx context.Context, discordID int64) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE onboarding_agents SET last_nudge_sent_at = NOW() WHERE discord_id = $1`, discordID)
	return err
}
