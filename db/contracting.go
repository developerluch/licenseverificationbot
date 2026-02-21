package db

import (
	"context"
	"time"
)

// ContractingManager represents a row from contracting_managers.
type ContractingManager struct {
	ID            int
	ManagerName   string
	DiscordUserID int64
	CalendlyURL   string
	Priority      int
	IsActive      bool
	CreatedAt     time.Time
}

// GetContractingManagers returns all active contracting managers ordered by priority.
func (d *DB) GetContractingManagers(ctx context.Context) ([]ContractingManager, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT id, manager_name, COALESCE(discord_user_id, 0), calendly_url,
         COALESCE(priority, 1), is_active, created_at
         FROM contracting_managers
         WHERE is_active = TRUE
         ORDER BY priority ASC, created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ContractingManager
	for rows.Next() {
		var m ContractingManager
		if err := rows.Scan(&m.ID, &m.ManagerName, &m.DiscordUserID,
			&m.CalendlyURL, &m.Priority, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// AddContractingManager adds a new contracting manager.
func (d *DB) AddContractingManager(ctx context.Context, name, url string, priority int) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO contracting_managers (manager_name, calendly_url, priority)
         VALUES ($1, $2, $3)`, name, url, priority)
	return err
}

// DeactivateContractingManager deactivates a contracting manager by name.
func (d *DB) DeactivateContractingManager(ctx context.Context, name string) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE contracting_managers SET is_active = FALSE
         WHERE LOWER(manager_name) = LOWER($1)`, name)
	return err
}
