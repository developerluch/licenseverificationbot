package db

import (
	"context"
	"database/sql"
	"time"
)

// ZoomVertical represents a zoom training vertical.
type ZoomVertical struct {
	ID          int
	Name        string
	Description string
	ZoomLink    string
	Schedule    string
	CreatedBy   int64
	IsActive    bool
	CreatedAt   time.Time
}

// ZoomAssignment represents a user's membership in a vertical.
type ZoomAssignment struct {
	ID         int
	DiscordID  int64
	VerticalID int
	JoinedAt   time.Time
}

// CreateZoomVertical creates a new zoom vertical.
func (d *DB) CreateZoomVertical(ctx context.Context, v ZoomVertical) (int, error) {
	var id int
	err := d.pool.QueryRowContext(ctx,
		`INSERT INTO zoom_verticals (name, description, zoom_link, schedule, created_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		v.Name, v.Description, v.ZoomLink, v.Schedule, v.CreatedBy).Scan(&id)
	return id, err
}

// GetZoomVerticals returns all active zoom verticals.
func (d *DB) GetZoomVerticals(ctx context.Context) ([]ZoomVertical, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT id, name, COALESCE(description,''), COALESCE(zoom_link,''),
		        COALESCE(schedule,''), COALESCE(created_by,0), is_active, created_at
		 FROM zoom_verticals WHERE is_active = true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ZoomVertical
	for rows.Next() {
		var v ZoomVertical
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.ZoomLink, &v.Schedule, &v.CreatedBy, &v.IsActive, &v.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// GetZoomVertical returns a single vertical by ID.
func (d *DB) GetZoomVertical(ctx context.Context, id int) (*ZoomVertical, error) {
	var v ZoomVertical
	err := d.pool.QueryRowContext(ctx,
		`SELECT id, name, COALESCE(description,''), COALESCE(zoom_link,''),
		        COALESCE(schedule,''), COALESCE(created_by,0), is_active, created_at
		 FROM zoom_verticals WHERE id = $1`, id).Scan(
		&v.ID, &v.Name, &v.Description, &v.ZoomLink, &v.Schedule, &v.CreatedBy, &v.IsActive, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &v, err
}

// JoinZoomVertical adds a user to a vertical.
func (d *DB) JoinZoomVertical(ctx context.Context, discordID int64, verticalID int) error {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO zoom_assignments (discord_id, vertical_id)
		 VALUES ($1, $2) ON CONFLICT (discord_id, vertical_id) DO NOTHING`, discordID, verticalID)
	return err
}

// LeaveZoomVertical removes a user from a vertical.
func (d *DB) LeaveZoomVertical(ctx context.Context, discordID int64, verticalID int) error {
	_, err := d.pool.ExecContext(ctx,
		`DELETE FROM zoom_assignments WHERE discord_id = $1 AND vertical_id = $2`,
		discordID, verticalID)
	return err
}

// GetUserZoomVerticals returns all verticals a user has joined.
func (d *DB) GetUserZoomVerticals(ctx context.Context, discordID int64) ([]ZoomVertical, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT v.id, v.name, COALESCE(v.description,''), COALESCE(v.zoom_link,''),
		        COALESCE(v.schedule,''), COALESCE(v.created_by,0), v.is_active, v.created_at
		 FROM zoom_verticals v
		 JOIN zoom_assignments a ON a.vertical_id = v.id
		 WHERE a.discord_id = $1 AND v.is_active = true
		 ORDER BY v.name`, discordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ZoomVertical
	for rows.Next() {
		var v ZoomVertical
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.ZoomLink, &v.Schedule, &v.CreatedBy, &v.IsActive, &v.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

// DeleteZoomVertical deactivates a vertical.
func (d *DB) DeleteZoomVertical(ctx context.Context, id int) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE zoom_verticals SET is_active = false WHERE id = $1`, id)
	return err
}
