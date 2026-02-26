package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ── Portal Structs ───────────────────────────────────────────────────────────

type AgentProfile struct {
	ID          string     `json:"id"`
	AgentID     string     `json:"agentId"`
	Bio         *string    `json:"bio"`
	City        *string    `json:"city"`
	Timezone    *string    `json:"timezone"`
	LinkedinURL *string    `json:"linkedinUrl"`
	PhotoURL    *string    `json:"photoUrl"`
	StartDate   *string    `json:"startDate"`
	CompTierID  *string    `json:"compTierId"`
	ManagerID   *string    `json:"managerId"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type CompTier struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Percentage int       `json:"percentage"`
	SortOrder  int       `json:"order"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type AgentLead struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agentId"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     *string   `json:"email"`
	Phone     *string   `json:"phone"`
	Source    *string   `json:"source"`
	Status    string    `json:"status"`
	Notes     *string   `json:"notes"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type AgentTrainingItem struct {
	ID          string     `json:"id"`
	AgentID     string     `json:"agentId"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Status      string     `json:"status"`
	DueDate     *string    `json:"dueDate"`
	CompletedAt *time.Time `json:"completedAt"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type AgentScheduleEvent struct {
	ID          string    `json:"id"`
	AgentID     string    `json:"agentId"`
	Title       string    `json:"title"`
	Description *string   `json:"description"`
	StartTime   time.Time `json:"startTime"`
	EndTime     time.Time `json:"endTime"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type OrgChartRow struct {
	AgentID    string  `json:"id"`
	FirstName  string  `json:"firstName"`
	LastName   string  `json:"lastName"`
	ManagerID  *string `json:"managerId"`
	CompTierID *string `json:"compTierId"`
	TierName   *string `json:"tierName"`
	TierPct    *int    `json:"tierPercentage"`
	TierOrder  *int    `json:"tierOrder"`
	StageName  *string `json:"stageName"`
	StageColor *string `json:"stageColor"`
}

type CompTierProfile struct {
	AgentID   string `json:"agentId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// ── Helper: nullable scan ────────────────────────────────────────────────────

func nullStr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func nullTime(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}

func nullInt(ni sql.NullInt64) *int {
	if ni.Valid {
		v := int(ni.Int64)
		return &v
	}
	return nil
}

// GetAgentByDiscordIDStr looks up an agent by string discord ID.
func (d *DB) GetAgentByDiscordIDStr(ctx context.Context, discordID string) (*Agent, error) {
	query := fmt.Sprintf(`SELECT %s FROM onboarding_agents WHERE discord_id::TEXT = $1`, AgentSelectColumns(""))
	row := d.pool.QueryRowContext(ctx, query, discordID)
	a, err := ScanAgent(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ── Profile CRUD ─────────────────────────────────────────────────────────────

func (d *DB) GetOrCreateProfile(ctx context.Context, agentID string) (*AgentProfile, error) {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO agent_profiles (agent_id) VALUES ($1) ON CONFLICT (agent_id) DO NOTHING`, agentID)
	if err != nil {
		return nil, fmt.Errorf("db: ensure profile: %w", err)
	}

	var p AgentProfile
	var bio, city, tz, linkedin, photo, startDate, compTier, manager sql.NullString
	err = d.pool.QueryRowContext(ctx,
		`SELECT id, agent_id, bio, city, timezone, linkedin_url, photo_url,
		        start_date::TEXT, comp_tier_id::TEXT, manager_id, created_at, updated_at
		 FROM agent_profiles WHERE agent_id = $1`, agentID).Scan(
		&p.ID, &p.AgentID, &bio, &city, &tz, &linkedin, &photo,
		&startDate, &compTier, &manager, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: get profile: %w", err)
	}
	p.Bio = nullStr(bio)
	p.City = nullStr(city)
	p.Timezone = nullStr(tz)
	p.LinkedinURL = nullStr(linkedin)
	p.PhotoURL = nullStr(photo)
	p.StartDate = nullStr(startDate)
	p.CompTierID = nullStr(compTier)
	p.ManagerID = nullStr(manager)
	return &p, nil
}

func (d *DB) UpdateProfile(ctx context.Context, agentID string, updates map[string]interface{}) (*AgentProfile, error) {
	// Ensure profile exists
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO agent_profiles (agent_id) VALUES ($1) ON CONFLICT (agent_id) DO NOTHING`, agentID)
	if err != nil {
		return nil, fmt.Errorf("db: ensure profile: %w", err)
	}

	sets := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argN := 1

	fieldMap := map[string]string{
		"bio":         "bio",
		"city":        "city",
		"timezone":    "timezone",
		"linkedinUrl": "linkedin_url",
		"photoUrl":    "photo_url",
		"startDate":   "start_date",
	}

	for jsonKey, col := range fieldMap {
		if v, ok := updates[jsonKey]; ok {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, argN))
			args = append(args, v)
			argN++
		}
	}

	if len(args) == 0 {
		return d.GetOrCreateProfile(ctx, agentID)
	}

	args = append(args, agentID)
	query := fmt.Sprintf("UPDATE agent_profiles SET %s WHERE agent_id = $%d",
		strings.Join(sets, ", "), argN)
	_, err = d.pool.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("db: update profile: %w", err)
	}
	return d.GetOrCreateProfile(ctx, agentID)
}

func (d *DB) SetManager(ctx context.Context, agentID string, managerID *string) (*AgentProfile, error) {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO agent_profiles (agent_id) VALUES ($1) ON CONFLICT (agent_id) DO NOTHING`, agentID)
	if err != nil {
		return nil, err
	}
	_, err = d.pool.ExecContext(ctx,
		`UPDATE agent_profiles SET manager_id = $1, updated_at = NOW() WHERE agent_id = $2`,
		managerID, agentID)
	if err != nil {
		return nil, fmt.Errorf("db: set manager: %w", err)
	}
	return d.GetOrCreateProfile(ctx, agentID)
}

func (d *DB) SetCompTier(ctx context.Context, agentID string, compTierID *string) (*AgentProfile, error) {
	_, err := d.pool.ExecContext(ctx,
		`INSERT INTO agent_profiles (agent_id) VALUES ($1) ON CONFLICT (agent_id) DO NOTHING`, agentID)
	if err != nil {
		return nil, err
	}
	_, err = d.pool.ExecContext(ctx,
		`UPDATE agent_profiles SET comp_tier_id = $1::UUID, updated_at = NOW() WHERE agent_id = $2`,
		compTierID, agentID)
	if err != nil {
		return nil, fmt.Errorf("db: set comp tier: %w", err)
	}
	return d.GetOrCreateProfile(ctx, agentID)
}

// ── Comp Tier CRUD ───────────────────────────────────────────────────────────

func (d *DB) ListCompTiers(ctx context.Context) ([]CompTier, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT id::TEXT, name, percentage, sort_order, created_at, updated_at
		 FROM comp_tiers ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CompTier
	for rows.Next() {
		var t CompTier
		if err := rows.Scan(&t.ID, &t.Name, &t.Percentage, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (d *DB) CreateCompTier(ctx context.Context, name string, percentage int) (*CompTier, error) {
	var t CompTier
	err := d.pool.QueryRowContext(ctx,
		`INSERT INTO comp_tiers (name, percentage, sort_order)
		 VALUES ($1, $2, (SELECT COALESCE(MAX(sort_order),0)+1 FROM comp_tiers))
		 RETURNING id::TEXT, name, percentage, sort_order, created_at, updated_at`,
		name, percentage).Scan(&t.ID, &t.Name, &t.Percentage, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: create comp tier: %w", err)
	}
	return &t, nil
}

func (d *DB) UpdateCompTier(ctx context.Context, id string, name string, percentage int) (*CompTier, error) {
	var t CompTier
	err := d.pool.QueryRowContext(ctx,
		`UPDATE comp_tiers SET name = $1, percentage = $2, updated_at = NOW()
		 WHERE id = $3::UUID
		 RETURNING id::TEXT, name, percentage, sort_order, created_at, updated_at`,
		name, percentage, id).Scan(&t.ID, &t.Name, &t.Percentage, &t.SortOrder, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: update comp tier: %w", err)
	}
	return &t, nil
}

func (d *DB) DeleteCompTier(ctx context.Context, id string) error {
	_, err := d.pool.ExecContext(ctx, `DELETE FROM comp_tiers WHERE id = $1::UUID`, id)
	return err
}

func (d *DB) ReorderCompTiers(ctx context.Context, ids []string) error {
	for i, id := range ids {
		_, err := d.pool.ExecContext(ctx,
			`UPDATE comp_tiers SET sort_order = $1, updated_at = NOW() WHERE id = $2::UUID`, i, id)
		if err != nil {
			return fmt.Errorf("db: reorder comp tier %s: %w", id, err)
		}
	}
	return nil
}

func (d *DB) GetCompTierProfiles(ctx context.Context, tierID string) ([]CompTierProfile, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT ap.agent_id, COALESCE(oa.first_name,''), COALESCE(oa.last_name,'')
		 FROM agent_profiles ap
		 LEFT JOIN onboarding_agents oa ON oa.discord_id::TEXT = ap.agent_id
		 WHERE ap.comp_tier_id = $1::UUID
		 ORDER BY oa.first_name`, tierID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CompTierProfile
	for rows.Next() {
		var p CompTierProfile
		if err := rows.Scan(&p.AgentID, &p.FirstName, &p.LastName); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

// ── Lead CRUD ────────────────────────────────────────────────────────────────

func (d *DB) ListAgentLeads(ctx context.Context, agentID string, status *string) ([]AgentLead, error) {
	query := `SELECT id::TEXT, agent_id, first_name, last_name, email, phone, source, status, notes, created_at, updated_at
	          FROM agent_leads WHERE agent_id = $1`
	args := []interface{}{agentID}
	if status != nil && *status != "" {
		query += ` AND status = $2`
		args = append(args, *status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := d.pool.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentLead
	for rows.Next() {
		var l AgentLead
		var email, phone, source, notes sql.NullString
		if err := rows.Scan(&l.ID, &l.AgentID, &l.FirstName, &l.LastName, &email, &phone, &source, &l.Status, &notes, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		l.Email = nullStr(email)
		l.Phone = nullStr(phone)
		l.Source = nullStr(source)
		l.Notes = nullStr(notes)
		result = append(result, l)
	}
	return result, rows.Err()
}

func (d *DB) CreateAgentLead(ctx context.Context, lead AgentLead) (*AgentLead, error) {
	var l AgentLead
	var email, phone, source, notes sql.NullString
	err := d.pool.QueryRowContext(ctx,
		`INSERT INTO agent_leads (agent_id, first_name, last_name, email, phone, source, status, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, COALESCE(NULLIF($7,''), 'new'), $8)
		 RETURNING id::TEXT, agent_id, first_name, last_name, email, phone, source, status, notes, created_at, updated_at`,
		lead.AgentID, lead.FirstName, lead.LastName, lead.Email, lead.Phone, lead.Source, lead.Status, lead.Notes,
	).Scan(&l.ID, &l.AgentID, &l.FirstName, &l.LastName, &email, &phone, &source, &l.Status, &notes, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: create lead: %w", err)
	}
	l.Email = nullStr(email)
	l.Phone = nullStr(phone)
	l.Source = nullStr(source)
	l.Notes = nullStr(notes)
	return &l, nil
}

func (d *DB) UpdateAgentLead(ctx context.Context, id, agentID string, updates map[string]interface{}) (*AgentLead, error) {
	sets := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argN := 1

	fieldMap := map[string]string{
		"firstName": "first_name",
		"lastName":  "last_name",
		"email":     "email",
		"phone":     "phone",
		"source":    "source",
		"status":    "status",
		"notes":     "notes",
	}

	for jsonKey, col := range fieldMap {
		if v, ok := updates[jsonKey]; ok {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, argN))
			args = append(args, v)
			argN++
		}
	}

	if len(args) == 0 {
		// nothing to update, just return existing
		var l AgentLead
		var email, phone, source, notes sql.NullString
		err := d.pool.QueryRowContext(ctx,
			`SELECT id::TEXT, agent_id, first_name, last_name, email, phone, source, status, notes, created_at, updated_at
			 FROM agent_leads WHERE id = $1::UUID AND agent_id = $2`, id, agentID,
		).Scan(&l.ID, &l.AgentID, &l.FirstName, &l.LastName, &email, &phone, &source, &l.Status, &notes, &l.CreatedAt, &l.UpdatedAt)
		if err != nil {
			return nil, err
		}
		l.Email = nullStr(email)
		l.Phone = nullStr(phone)
		l.Source = nullStr(source)
		l.Notes = nullStr(notes)
		return &l, nil
	}

	args = append(args, id, agentID)
	query := fmt.Sprintf("UPDATE agent_leads SET %s WHERE id = $%d::UUID AND agent_id = $%d RETURNING id::TEXT, agent_id, first_name, last_name, email, phone, source, status, notes, created_at, updated_at",
		strings.Join(sets, ", "), argN, argN+1)

	var l AgentLead
	var email, phone, source, notes sql.NullString
	err := d.pool.QueryRowContext(ctx, query, args...).Scan(
		&l.ID, &l.AgentID, &l.FirstName, &l.LastName, &email, &phone, &source, &l.Status, &notes, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: update lead: %w", err)
	}
	l.Email = nullStr(email)
	l.Phone = nullStr(phone)
	l.Source = nullStr(source)
	l.Notes = nullStr(notes)
	return &l, nil
}

func (d *DB) DeleteAgentLead(ctx context.Context, id, agentID string) error {
	_, err := d.pool.ExecContext(ctx,
		`DELETE FROM agent_leads WHERE id = $1::UUID AND agent_id = $2`, id, agentID)
	return err
}

// ── Training CRUD ────────────────────────────────────────────────────────────

func (d *DB) ListTrainingItems(ctx context.Context, agentID string) ([]AgentTrainingItem, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT id::TEXT, agent_id, title, description, status, due_date::TEXT, completed_at, created_at, updated_at
		 FROM agent_training_items WHERE agent_id = $1 ORDER BY created_at`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentTrainingItem
	for rows.Next() {
		var t AgentTrainingItem
		var desc, dueDate sql.NullString
		var completedAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.AgentID, &t.Title, &desc, &t.Status, &dueDate, &completedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Description = nullStr(desc)
		t.DueDate = nullStr(dueDate)
		t.CompletedAt = nullTime(completedAt)
		result = append(result, t)
	}
	return result, rows.Err()
}

func (d *DB) CreateTrainingItem(ctx context.Context, item AgentTrainingItem) (*AgentTrainingItem, error) {
	var t AgentTrainingItem
	var desc, dueDate sql.NullString
	var completedAt sql.NullTime
	err := d.pool.QueryRowContext(ctx,
		`INSERT INTO agent_training_items (agent_id, title, description, due_date)
		 VALUES ($1, $2, $3, $4::DATE)
		 RETURNING id::TEXT, agent_id, title, description, status, due_date::TEXT, completed_at, created_at, updated_at`,
		item.AgentID, item.Title, item.Description, item.DueDate,
	).Scan(&t.ID, &t.AgentID, &t.Title, &desc, &t.Status, &dueDate, &completedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: create training item: %w", err)
	}
	t.Description = nullStr(desc)
	t.DueDate = nullStr(dueDate)
	t.CompletedAt = nullTime(completedAt)
	return &t, nil
}

func (d *DB) UpdateTrainingItem(ctx context.Context, id, agentID string, updates map[string]interface{}) (*AgentTrainingItem, error) {
	sets := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argN := 1

	fieldMap := map[string]string{
		"title":       "title",
		"description": "description",
		"dueDate":     "due_date",
	}

	for jsonKey, col := range fieldMap {
		if v, ok := updates[jsonKey]; ok {
			if col == "due_date" {
				sets = append(sets, fmt.Sprintf("%s = $%d::DATE", col, argN))
			} else {
				sets = append(sets, fmt.Sprintf("%s = $%d", col, argN))
			}
			args = append(args, v)
			argN++
		}
	}

	if v, ok := updates["status"]; ok {
		sets = append(sets, fmt.Sprintf("status = $%d", argN))
		args = append(args, v)
		argN++
		if v == "completed" {
			sets = append(sets, "completed_at = NOW()")
		}
	}

	args = append(args, id, agentID)
	query := fmt.Sprintf(
		`UPDATE agent_training_items SET %s WHERE id = $%d::UUID AND agent_id = $%d
		 RETURNING id::TEXT, agent_id, title, description, status, due_date::TEXT, completed_at, created_at, updated_at`,
		strings.Join(sets, ", "), argN, argN+1)

	var t AgentTrainingItem
	var desc, dueDate sql.NullString
	var completedAt sql.NullTime
	err := d.pool.QueryRowContext(ctx, query, args...).Scan(
		&t.ID, &t.AgentID, &t.Title, &desc, &t.Status, &dueDate, &completedAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: update training item: %w", err)
	}
	t.Description = nullStr(desc)
	t.DueDate = nullStr(dueDate)
	t.CompletedAt = nullTime(completedAt)
	return &t, nil
}

func (d *DB) DeleteTrainingItem(ctx context.Context, id, agentID string) error {
	_, err := d.pool.ExecContext(ctx,
		`DELETE FROM agent_training_items WHERE id = $1::UUID AND agent_id = $2`, id, agentID)
	return err
}

// ── Schedule CRUD ────────────────────────────────────────────────────────────

func (d *DB) ListScheduleEvents(ctx context.Context, agentID string, start, end *time.Time) ([]AgentScheduleEvent, error) {
	query := `SELECT id::TEXT, agent_id, title, description, start_time, end_time, type, created_at, updated_at
	          FROM agent_schedule_events WHERE agent_id = $1`
	args := []interface{}{agentID}
	argN := 2

	if start != nil {
		query += fmt.Sprintf(` AND start_time >= $%d`, argN)
		args = append(args, *start)
		argN++
	}
	if end != nil {
		query += fmt.Sprintf(` AND end_time <= $%d`, argN)
		args = append(args, *end)
		argN++
	}
	query += ` ORDER BY start_time`

	rows, err := d.pool.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []AgentScheduleEvent
	for rows.Next() {
		var e AgentScheduleEvent
		var desc sql.NullString
		if err := rows.Scan(&e.ID, &e.AgentID, &e.Title, &desc, &e.StartTime, &e.EndTime, &e.Type, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.Description = nullStr(desc)
		result = append(result, e)
	}
	return result, rows.Err()
}

func (d *DB) CreateScheduleEvent(ctx context.Context, event AgentScheduleEvent) (*AgentScheduleEvent, error) {
	var e AgentScheduleEvent
	var desc sql.NullString
	eventType := event.Type
	if eventType == "" {
		eventType = "other"
	}
	err := d.pool.QueryRowContext(ctx,
		`INSERT INTO agent_schedule_events (agent_id, title, description, start_time, end_time, type)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id::TEXT, agent_id, title, description, start_time, end_time, type, created_at, updated_at`,
		event.AgentID, event.Title, event.Description, event.StartTime, event.EndTime, eventType,
	).Scan(&e.ID, &e.AgentID, &e.Title, &desc, &e.StartTime, &e.EndTime, &e.Type, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: create schedule event: %w", err)
	}
	e.Description = nullStr(desc)
	return &e, nil
}

func (d *DB) UpdateScheduleEvent(ctx context.Context, id, agentID string, updates map[string]interface{}) (*AgentScheduleEvent, error) {
	sets := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argN := 1

	fieldMap := map[string]string{
		"title":       "title",
		"description": "description",
		"startTime":   "start_time",
		"endTime":     "end_time",
		"type":        "type",
	}

	for jsonKey, col := range fieldMap {
		if v, ok := updates[jsonKey]; ok {
			sets = append(sets, fmt.Sprintf("%s = $%d", col, argN))
			args = append(args, v)
			argN++
		}
	}

	args = append(args, id, agentID)
	query := fmt.Sprintf(
		`UPDATE agent_schedule_events SET %s WHERE id = $%d::UUID AND agent_id = $%d
		 RETURNING id::TEXT, agent_id, title, description, start_time, end_time, type, created_at, updated_at`,
		strings.Join(sets, ", "), argN, argN+1)

	var e AgentScheduleEvent
	var desc sql.NullString
	err := d.pool.QueryRowContext(ctx, query, args...).Scan(
		&e.ID, &e.AgentID, &e.Title, &desc, &e.StartTime, &e.EndTime, &e.Type, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("db: update schedule event: %w", err)
	}
	e.Description = nullStr(desc)
	return &e, nil
}

func (d *DB) DeleteScheduleEvent(ctx context.Context, id, agentID string) error {
	_, err := d.pool.ExecContext(ctx,
		`DELETE FROM agent_schedule_events WHERE id = $1::UUID AND agent_id = $2`, id, agentID)
	return err
}

// ── Org Chart ────────────────────────────────────────────────────────────────

var stageNames = map[int]string{
	1: "Welcome",
	2: "Form Start",
	3: "Sorted",
	4: "Student",
	5: "Verified",
	6: "Contracting",
	7: "Setup",
	8: "Active",
}

var stageColors = map[int]string{
	1: "#94a3b8",
	2: "#60a5fa",
	3: "#a78bfa",
	4: "#f59e0b",
	5: "#22c55e",
	6: "#06b6d4",
	7: "#f97316",
	8: "#10b981",
}

func (d *DB) GetOrgChart(ctx context.Context) ([]OrgChartRow, error) {
	rows, err := d.pool.QueryContext(ctx,
		`SELECT oa.discord_id::TEXT, COALESCE(oa.first_name,''), COALESCE(oa.last_name,''),
		        ap.manager_id, ap.comp_tier_id::TEXT,
		        ct.name, ct.percentage, ct.sort_order,
		        COALESCE(oa.current_stage, 1)
		 FROM onboarding_agents oa
		 LEFT JOIN agent_profiles ap ON ap.agent_id = oa.discord_id::TEXT
		 LEFT JOIN comp_tiers ct ON ct.id = ap.comp_tier_id
		 WHERE oa.kicked_at IS NULL
		 ORDER BY oa.first_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []OrgChartRow
	for rows.Next() {
		var r OrgChartRow
		var managerID, compTierID, tierName sql.NullString
		var tierPct, tierOrder sql.NullInt64
		var stage int
		if err := rows.Scan(&r.AgentID, &r.FirstName, &r.LastName,
			&managerID, &compTierID, &tierName, &tierPct, &tierOrder, &stage); err != nil {
			return nil, err
		}
		r.ManagerID = nullStr(managerID)
		r.CompTierID = nullStr(compTierID)
		r.TierName = nullStr(tierName)
		r.TierPct = nullInt(tierPct)
		r.TierOrder = nullInt(tierOrder)
		if name, ok := stageNames[stage]; ok {
			r.StageName = &name
			color := stageColors[stage]
			r.StageColor = &color
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
