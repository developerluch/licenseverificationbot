package db

import (
	"context"
	"database/sql"
	"time"
)

// ApprovalRequest represents a row in the approval_requests table.
type ApprovalRequest struct {
	ID              int
	AgentDiscordID  int64
	GuildID         int64
	Agency          string
	OwnerDiscordID  int64
	Status          string
	DenialReason    string
	RequestedAt     time.Time
	RespondedAt     *time.Time
	DMMessageID     string
}

// CreateApprovalRequest inserts a new approval request (upserts on conflict).
func (d *DB) CreateApprovalRequest(ctx context.Context, req ApprovalRequest) (int, error) {
	var id int
	err := d.pool.QueryRowContext(ctx,
		`INSERT INTO approval_requests (agent_discord_id, guild_id, agency, owner_discord_id, status)
		 VALUES ($1, $2, $3, $4, 'pending')
		 ON CONFLICT (agent_discord_id, guild_id) DO UPDATE SET
		   agency = $3, owner_discord_id = $4, status = 'pending',
		   denial_reason = NULL, responded_at = NULL, requested_at = NOW()
		 RETURNING id`,
		req.AgentDiscordID, req.GuildID, req.Agency, req.OwnerDiscordID).Scan(&id)
	return id, err
}

// GetApprovalRequest returns the approval request by ID.
func (d *DB) GetApprovalRequest(ctx context.Context, id int) (*ApprovalRequest, error) {
	var r ApprovalRequest
	err := d.pool.QueryRowContext(ctx,
		`SELECT id, agent_discord_id, guild_id, agency, owner_discord_id,
		        status, COALESCE(denial_reason,''), requested_at, responded_at,
		        COALESCE(dm_message_id,'')
		 FROM approval_requests WHERE id = $1`, id).Scan(
		&r.ID, &r.AgentDiscordID, &r.GuildID, &r.Agency, &r.OwnerDiscordID,
		&r.Status, &r.DenialReason, &r.RequestedAt, &r.RespondedAt, &r.DMMessageID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// GetPendingApproval returns the pending approval for an agent.
func (d *DB) GetPendingApproval(ctx context.Context, agentDiscordID int64) (*ApprovalRequest, error) {
	var r ApprovalRequest
	err := d.pool.QueryRowContext(ctx,
		`SELECT id, agent_discord_id, guild_id, agency, owner_discord_id,
		        status, COALESCE(denial_reason,''), requested_at, responded_at,
		        COALESCE(dm_message_id,'')
		 FROM approval_requests
		 WHERE agent_discord_id = $1 AND status = 'pending'
		 LIMIT 1`, agentDiscordID).Scan(
		&r.ID, &r.AgentDiscordID, &r.GuildID, &r.Agency, &r.OwnerDiscordID,
		&r.Status, &r.DenialReason, &r.RequestedAt, &r.RespondedAt, &r.DMMessageID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// ApproveAgent marks an approval request as approved.
func (d *DB) ApproveAgent(ctx context.Context, requestID int) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE approval_requests SET status = 'approved', responded_at = NOW() WHERE id = $1`,
		requestID)
	return err
}

// DenyAgent marks an approval request as denied with an optional reason.
func (d *DB) DenyAgent(ctx context.Context, requestID int, reason string) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE approval_requests SET status = 'denied', denial_reason = $2, responded_at = NOW() WHERE id = $1`,
		requestID, reason)
	return err
}

// UpdateApprovalDMMessageID stores the DM message ID for editing later.
func (d *DB) UpdateApprovalDMMessageID(ctx context.Context, requestID int, messageID string) error {
	_, err := d.pool.ExecContext(ctx,
		`UPDATE approval_requests SET dm_message_id = $2 WHERE id = $1`,
		requestID, messageID)
	return err
}
