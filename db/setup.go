package db

import (
	"context"
	"time"
)

// SetupItem defines a setup checklist item.
type SetupItem struct {
	Key   string
	Label string
	Emoji string
}

// SetupItems lists the 5 setup checklist items.
var SetupItems = []SetupItem{
	{Key: "ghl_account", Label: "GHL Account Created", Emoji: "\U0001f4bc"},
	{Key: "wavv_dialer", Label: "WAVV Dialer Configured", Emoji: "\U0001f4de"},
	{Key: "eo_insurance", Label: "E&O Insurance Confirmed", Emoji: "\U0001f6e1\ufe0f"},
	{Key: "direct_deposit", Label: "Direct Deposit Set Up", Emoji: "\U0001f4b0"},
	{Key: "training_modules", Label: "Training Modules Completed", Emoji: "\U0001f4da"},
}

// CompleteSetupItem marks a setup item as completed for an agent.
func (d *DB) CompleteSetupItem(ctx context.Context, discordID int64, itemKey string) error {
	now := time.Now()
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO agent_setup_progress (discord_id, item_key, completed, completed_at)
         VALUES ($1, $2, TRUE, $3)
         ON CONFLICT (discord_id, item_key) DO UPDATE SET completed = TRUE, completed_at = $3`,
		discordID, itemKey, now)
	return err
}

// GetSetupProgress returns a map of item_key -> completed for an agent.
func (d *DB) GetSetupProgress(ctx context.Context, discordID int64) (map[string]bool, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT item_key, completed FROM agent_setup_progress WHERE discord_id = $1`,
		discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	progress := make(map[string]bool)
	for rows.Next() {
		var key string
		var completed bool
		if err := rows.Scan(&key, &completed); err != nil {
			return nil, err
		}
		progress[key] = completed
	}
	return progress, rows.Err()
}

// IsSetupComplete returns true if all setup items are completed.
func (d *DB) IsSetupComplete(ctx context.Context, discordID int64) (bool, error) {
	progress, err := d.GetSetupProgress(ctx, discordID)
	if err != nil {
		return false, err
	}
	for _, item := range SetupItems {
		if !progress[item.Key] {
			return false, nil
		}
	}
	return true, nil
}
