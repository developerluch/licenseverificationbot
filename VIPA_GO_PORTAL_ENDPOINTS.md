# VIPA Go Bot — Portal Endpoints (Agent Profiles, Leads, Training, Schedule, Org Chart, Comp Tiers)

## Goal
Add ALL agent portal endpoints directly into the Go bot so the Electron dashboard only needs ONE backend. No more separate admin portal API. Everything goes through the existing `http.NewServeMux` server on the same port with the same Bearer token auth.

## CRITICAL: Match Existing Patterns
- **Database**: Raw SQL with `database/sql` + `github.com/lib/pq`. NO ORM. Use `$1, $2` parameter placeholders. Use `COALESCE` for nullable columns in SELECTs.
- **HTTP**: Standard `net/http` with Go 1.22+ routing patterns (`"GET /api/v1/..."`, `"POST /api/v1/..."`). All routes wrapped with `auth()` middleware.
- **JSON**: Use `encoding/json` for request/response marshaling. Set `Content-Type: application/json` header.
- **Context**: All DB methods take `context.Context` as first param.
- **Errors**: Return JSON `{"error": "message"}` with appropriate HTTP status codes.
- **IDs**: Use UUID strings (`gen_random_uuid()`) for portal table primary keys. Agents are identified by `discord_id BIGINT` (matching `onboarding_agents`).

---

## TASK 1: Database Migrations

**File**: `db/db.go` — Add these migrations to the existing `migrate()` function's `migrations` slice, AFTER all existing migrations:

```sql
-- Agent profiles (extends onboarding_agents with portal-specific fields)
CREATE TABLE IF NOT EXISTS agent_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discord_id BIGINT NOT NULL UNIQUE REFERENCES onboarding_agents(discord_id) ON DELETE CASCADE,
    bio TEXT,
    city TEXT,
    timezone TEXT,
    linkedin_url TEXT,
    photo_url TEXT,
    start_date DATE,
    comp_tier_id UUID,
    manager_id BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Comp tiers
CREATE TABLE IF NOT EXISTS comp_tiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    percentage NUMERIC(5,2) NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Agent leads (recruitment pipeline per agent)
CREATE TABLE IF NOT EXISTS agent_leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discord_id BIGINT NOT NULL REFERENCES onboarding_agents(discord_id) ON DELETE CASCADE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT,
    phone TEXT,
    source TEXT,
    status TEXT NOT NULL DEFAULT 'new',
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_agent_leads_discord ON agent_leads(discord_id);

-- Agent training items
CREATE TABLE IF NOT EXISTS agent_training_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discord_id BIGINT NOT NULL REFERENCES onboarding_agents(discord_id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    due_date DATE,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_agent_training_discord ON agent_training_items(discord_id);

-- Agent schedule events
CREATE TABLE IF NOT EXISTS agent_schedule_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    discord_id BIGINT NOT NULL REFERENCES onboarding_agents(discord_id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    event_type TEXT NOT NULL DEFAULT 'other',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_agent_schedule_discord ON agent_schedule_events(discord_id);
```

Add the foreign key constraint for comp_tier_id AFTER the comp_tiers table creation:
```sql
DO $$ BEGIN
    ALTER TABLE agent_profiles ADD CONSTRAINT fk_profile_comp_tier
        FOREIGN KEY (comp_tier_id) REFERENCES comp_tiers(id) ON DELETE SET NULL;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;
```

---

## TASK 2: DB Methods — Portal Models & CRUD

**Create file**: `db/portal.go`

This file contains ALL portal DB types and methods. Use this exact structure:

### Types

```go
package db

import (
    "context"
    "database/sql"
    "fmt"
    "time"
)

// AgentProfile represents a row in agent_profiles
type AgentProfile struct {
    ID          string
    DiscordID   int64
    Bio         sql.NullString
    City        sql.NullString
    Timezone    sql.NullString
    LinkedinURL sql.NullString
    PhotoURL    sql.NullString
    StartDate   *time.Time
    CompTierID  sql.NullString
    ManagerID   sql.NullInt64
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type CompTier struct {
    ID         string
    Name       string
    Percentage float64
    SortOrder  int
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

type AgentLead struct {
    ID        string
    DiscordID int64
    FirstName string
    LastName  string
    Email     sql.NullString
    Phone     sql.NullString
    Source    sql.NullString
    Status    string
    Notes     sql.NullString
    CreatedAt time.Time
    UpdatedAt time.Time
}

type AgentTrainingItem struct {
    ID          string
    DiscordID   int64
    Title       string
    Description sql.NullString
    Status      string
    DueDate     *time.Time
    CompletedAt *time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type AgentScheduleEvent struct {
    ID          string
    DiscordID   int64
    Title       string
    Description sql.NullString
    StartTime   time.Time
    EndTime     time.Time
    EventType   string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Profile Methods

```go
// GetOrCreateProfile returns the profile for a discord_id, creating one if it doesn't exist.
func (d *DB) GetOrCreateProfile(ctx context.Context, discordID int64) (*AgentProfile, error) {
    // INSERT ON CONFLICT DO NOTHING, then SELECT
    _, err := d.pool.ExecContext(ctx,
        `INSERT INTO agent_profiles (discord_id) VALUES ($1) ON CONFLICT (discord_id) DO NOTHING`, discordID)
    if err != nil {
        return nil, fmt.Errorf("db: create profile: %w", err)
    }

    var p AgentProfile
    err = d.pool.QueryRowContext(ctx,
        `SELECT id, discord_id, bio, city, timezone, linkedin_url, photo_url,
         start_date, comp_tier_id, manager_id, created_at, updated_at
         FROM agent_profiles WHERE discord_id = $1`, discordID).Scan(
        &p.ID, &p.DiscordID, &p.Bio, &p.City, &p.Timezone, &p.LinkedinURL,
        &p.PhotoURL, &p.StartDate, &p.CompTierID, &p.ManagerID, &p.CreatedAt, &p.UpdatedAt)
    if err != nil {
        return nil, fmt.Errorf("db: get profile: %w", err)
    }
    return &p, nil
}

// UpdateProfile updates profile fields. Only non-nil fields are updated.
func (d *DB) UpdateProfile(ctx context.Context, discordID int64, bio, city, timezone, linkedinUrl, photoUrl *string, startDate *time.Time) error {
    // Build dynamic UPDATE similar to UpsertAgent pattern
    // SET updated_at = NOW() plus any non-nil fields
    // WHERE discord_id = $N
    // ... (follow the UpsertAgent pattern with sets/args/argN)
}

// SetManager sets or clears the manager_id on a profile
func (d *DB) SetManager(ctx context.Context, discordID int64, managerID *int64) error {
    _, err := d.pool.ExecContext(ctx,
        `UPDATE agent_profiles SET manager_id = $1, updated_at = NOW() WHERE discord_id = $2`,
        managerID, discordID)
    return err
}

// SetCompTier sets or clears the comp_tier_id on a profile
func (d *DB) SetCompTier(ctx context.Context, discordID int64, compTierID *string) error {
    _, err := d.pool.ExecContext(ctx,
        `UPDATE agent_profiles SET comp_tier_id = $1, updated_at = NOW() WHERE discord_id = $2`,
        compTierID, discordID)
    return err
}
```

### Comp Tier Methods

```go
// GetCompTiers returns all tiers ordered by sort_order
func (d *DB) GetCompTiers(ctx context.Context) ([]CompTier, error) { ... }

// GetCompTierWithAgents returns a tier with its assigned agents' names
// JOIN agent_profiles ON comp_tier_id, then JOIN onboarding_agents for first_name/last_name
func (d *DB) GetCompTiersWithAgents(ctx context.Context) ([]CompTierWithAgents, error) { ... }
// CompTierWithAgents includes the tier plus a slice of {DiscordID, FirstName, LastName}

// CreateCompTier inserts a new tier. Set sort_order = (SELECT COALESCE(MAX(sort_order),0)+1 FROM comp_tiers)
func (d *DB) CreateCompTier(ctx context.Context, name string, percentage float64) (*CompTier, error) { ... }

// UpdateCompTier updates name and/or percentage
func (d *DB) UpdateCompTier(ctx context.Context, id string, name *string, percentage *float64) (*CompTier, error) { ... }

// DeleteCompTier deletes a tier by id
func (d *DB) DeleteCompTier(ctx context.Context, id string) error { ... }

// ReorderCompTiers accepts an ordered slice of tier IDs and sets sort_order = index
func (d *DB) ReorderCompTiers(ctx context.Context, ids []string) error {
    // Use a transaction: tx.ExecContext for each id with SET sort_order = $1 WHERE id = $2
}
```

### Agent Leads Methods

```go
// GetAgentLeads returns leads for a discord_id, optionally filtered by status
func (d *DB) GetAgentLeads(ctx context.Context, discordID int64, status string) ([]AgentLead, error) { ... }

// CreateAgentLead inserts a new lead
func (d *DB) CreateAgentLead(ctx context.Context, discordID int64, firstName, lastName string, email, phone, source, notes *string) (*AgentLead, error) {
    // INSERT ... RETURNING id, discord_id, first_name, last_name, email, phone, source, status, notes, created_at, updated_at
}

// UpdateAgentLead updates lead fields (dynamic SET like UpsertAgent pattern)
func (d *DB) UpdateAgentLead(ctx context.Context, id string, discordID int64, firstName, lastName, email, phone, source, status, notes *string) (*AgentLead, error) { ... }

// DeleteAgentLead deletes a lead. Verify it belongs to the discord_id.
func (d *DB) DeleteAgentLead(ctx context.Context, id string, discordID int64) error {
    _, err := d.pool.ExecContext(ctx, `DELETE FROM agent_leads WHERE id = $1 AND discord_id = $2`, id, discordID)
    return err
}
```

### Training Methods

```go
// GetTrainingItems returns all training items for a discord_id
func (d *DB) GetTrainingItems(ctx context.Context, discordID int64) ([]AgentTrainingItem, error) { ... }

// CreateTrainingItem inserts a new item
func (d *DB) CreateTrainingItem(ctx context.Context, discordID int64, title string, description *string, dueDate *time.Time) (*AgentTrainingItem, error) { ... }

// UpdateTrainingItem updates fields. If status changes to "completed", set completed_at = NOW()
func (d *DB) UpdateTrainingItem(ctx context.Context, id string, discordID int64, title, description, status *string, dueDate *time.Time) (*AgentTrainingItem, error) { ... }

// DeleteTrainingItem deletes an item. Verify ownership via discord_id.
func (d *DB) DeleteTrainingItem(ctx context.Context, id string, discordID int64) error { ... }
```

### Schedule Methods

```go
// GetScheduleEvents returns events for a discord_id, optionally filtered by time range
func (d *DB) GetScheduleEvents(ctx context.Context, discordID int64, start, end *time.Time) ([]AgentScheduleEvent, error) { ... }

// CreateScheduleEvent inserts a new event
func (d *DB) CreateScheduleEvent(ctx context.Context, discordID int64, title string, description *string, startTime, endTime time.Time, eventType string) (*AgentScheduleEvent, error) { ... }

// UpdateScheduleEvent updates fields
func (d *DB) UpdateScheduleEvent(ctx context.Context, id string, discordID int64, title, description *string, startTime, endTime *time.Time, eventType *string) (*AgentScheduleEvent, error) { ... }

// DeleteScheduleEvent deletes an event. Verify ownership via discord_id.
func (d *DB) DeleteScheduleEvent(ctx context.Context, id string, discordID int64) error { ... }
```

### Org Chart Method

```go
// GetOrgChart returns all agents with their profile info for building the org chart.
// JOIN onboarding_agents LEFT JOIN agent_profiles LEFT JOIN comp_tiers
// Return: discord_id, first_name, last_name, manager_id (from profile), comp_tier name/percentage, current_stage
type OrgChartNode struct {
    DiscordID    int64
    FirstName    string
    LastName     string
    ManagerID    sql.NullInt64
    CompTierName sql.NullString
    CompTierPct  sql.NullFloat64
    CurrentStage int
}

func (d *DB) GetOrgChart(ctx context.Context) ([]OrgChartNode, error) {
    // SELECT a.discord_id, COALESCE(a.first_name,''), COALESCE(a.last_name,''),
    //   p.manager_id, ct.name, ct.percentage, COALESCE(a.current_stage, 1)
    // FROM onboarding_agents a
    // LEFT JOIN agent_profiles p ON p.discord_id = a.discord_id
    // LEFT JOIN comp_tiers ct ON ct.id = p.comp_tier_id
    // WHERE a.kicked_at IS NULL
    // ORDER BY a.first_name
}
```

---

## TASK 3: API Route Handlers

**Create file**: `api/portal.go`

This file contains ALL portal HTTP handlers. Follow existing patterns from `api/agents.go`:
- Parse path params with `r.PathValue("discordID")`
- Parse query params with `r.URL.Query().Get("status")`
- Decode JSON bodies with `json.NewDecoder(r.Body).Decode(&req)`
- Respond with helper: create a `jsonResponse(w, status, data)` helper and a `jsonError(w, status, msg)` helper at the top
- Use `strconv.ParseInt(discordIDStr, 10, 64)` to convert discord IDs from path params

### Response JSON Shapes (match what the Electron app expects)

The Electron app's `api.ts` currently calls these paths via `adminFetch`. We're moving them to the Go bot under `/api/v1/portal/...` prefix. Here are the exact response shapes:

**Profile** (`GET /api/v1/portal/profile/{discordID}`):
```json
{
  "id": "uuid",
  "agentId": "123456789",
  "bio": "string|null",
  "city": "string|null",
  "timezone": "string|null",
  "linkedinUrl": "string|null",
  "photoUrl": "string|null",
  "startDate": "2024-01-15|null",
  "compTierId": "uuid|null",
  "managerId": "123456789|null",
  "createdAt": "2024-01-15T00:00:00Z",
  "updatedAt": "2024-01-15T00:00:00Z",
  "compTier": {"id":"uuid","name":"Gold","percentage":80,"order":1} | null,
  "manager": {"id":"123456789","firstName":"John","lastName":"Doe"} | null
}
```

**Comp Tiers** (`GET /api/v1/portal/comp-tiers`):
```json
[
  {
    "id": "uuid",
    "name": "Gold",
    "percentage": 80,
    "order": 1,
    "createdAt": "...",
    "updatedAt": "...",
    "profiles": [
      {"agentId": "123456789", "agent": {"firstName": "John", "lastName": "Doe"}}
    ]
  }
]
```

**Agent Leads** (`GET /api/v1/portal/agents/{discordID}/leads`):
```json
[
  {
    "id": "uuid",
    "agentId": "123456789",
    "firstName": "Lead",
    "lastName": "Person",
    "email": "string|null",
    "phone": "string|null",
    "source": "string|null",
    "status": "new",
    "notes": "string|null",
    "createdAt": "...",
    "updatedAt": "..."
  }
]
```

**Training Items** (`GET /api/v1/portal/agents/{discordID}/training`):
```json
[
  {
    "id": "uuid",
    "agentId": "123456789",
    "title": "Complete Onboarding",
    "description": "string|null",
    "status": "pending",
    "dueDate": "2024-02-01|null",
    "completedAt": "...|null",
    "createdAt": "...",
    "updatedAt": "..."
  }
]
```

**Schedule Events** (`GET /api/v1/portal/agents/{discordID}/schedule`):
```json
[
  {
    "id": "uuid",
    "agentId": "123456789",
    "title": "Team Meeting",
    "description": "string|null",
    "startTime": "2024-01-15T09:00:00Z",
    "endTime": "2024-01-15T10:00:00Z",
    "type": "meeting",
    "createdAt": "...",
    "updatedAt": "..."
  }
]
```

**Org Chart** (`GET /api/v1/portal/org-chart`):
```json
[
  {
    "id": "123456789",
    "firstName": "John",
    "lastName": "Doe",
    "profile": {
      "managerId": "987654321|null",
      "compTier": {"id":"uuid","name":"Gold","percentage":80,"order":1} | null
    } | null,
    "stage": {"name": "Active", "color": "#22c55e"} | null
  }
]
```

IMPORTANT: `agentId` in JSON responses should be the discord_id as a STRING (not integer). The Electron app treats agent IDs as strings.

### Stage-to-name/color mapping for org chart:
```go
var stageInfo = map[int]struct{ Name, Color string }{
    1: {"Welcome", "#6b7280"},
    2: {"Form Started", "#3b82f6"},
    3: {"Sorted", "#8b5cf6"},
    4: {"Student", "#f59e0b"},
    5: {"Verified", "#10b981"},
    6: {"Contracting", "#06b6d4"},
    7: {"Setup", "#ec4899"},
    8: {"Active", "#22c55e"},
}
```

### All Route Handlers to implement:

```
// Profile
GET  /api/v1/portal/profile/{discordID}        → handleGetProfile
PUT  /api/v1/portal/profile/{discordID}        → handleUpdateProfile
PUT  /api/v1/portal/profile/{discordID}/manager → handleSetManager
PUT  /api/v1/portal/profile/{discordID}/comp-tier → handleSetCompTier

// Leads
GET    /api/v1/portal/agents/{discordID}/leads           → handleGetLeads (query: ?status=new)
POST   /api/v1/portal/agents/{discordID}/leads           → handleCreateLead
PUT    /api/v1/portal/agents/{discordID}/leads/{leadID}  → handleUpdateLead
DELETE /api/v1/portal/agents/{discordID}/leads/{leadID}  → handleDeleteLead

// Training
GET    /api/v1/portal/agents/{discordID}/training           → handleGetTraining
POST   /api/v1/portal/agents/{discordID}/training           → handleCreateTraining
PUT    /api/v1/portal/agents/{discordID}/training/{itemID}  → handleUpdateTraining
DELETE /api/v1/portal/agents/{discordID}/training/{itemID}  → handleDeleteTraining

// Schedule
GET    /api/v1/portal/agents/{discordID}/schedule            → handleGetSchedule (query: ?start=&end=)
POST   /api/v1/portal/agents/{discordID}/schedule            → handleCreateSchedule
PUT    /api/v1/portal/agents/{discordID}/schedule/{eventID}  → handleUpdateSchedule
DELETE /api/v1/portal/agents/{discordID}/schedule/{eventID}  → handleDeleteSchedule

// Org Chart
GET /api/v1/portal/org-chart                    → handleGetOrgChart
PUT /api/v1/portal/org-chart/assign-manager     → handleAssignManager

// Comp Tiers
GET    /api/v1/portal/comp-tiers           → handleGetCompTiers
POST   /api/v1/portal/comp-tiers           → handleCreateCompTier
PUT    /api/v1/portal/comp-tiers/{tierID}  → handleUpdateCompTier
DELETE /api/v1/portal/comp-tiers/{tierID}  → handleDeleteCompTier
PUT    /api/v1/portal/comp-tiers/reorder   → handleReorderCompTiers
```

---

## TASK 4: Register Routes in server.go

**File**: `api/server.go` — In `NewServer()`, add all portal routes after the existing admin action endpoints. ALL wrapped with `auth()`:

```go
// Portal: Profile
mux.HandleFunc("GET /api/v1/portal/profile/{discordID}", auth(s.handleGetProfile))
mux.HandleFunc("PUT /api/v1/portal/profile/{discordID}", auth(s.handleUpdateProfile))
mux.HandleFunc("PUT /api/v1/portal/profile/{discordID}/manager", auth(s.handleSetManager))
mux.HandleFunc("PUT /api/v1/portal/profile/{discordID}/comp-tier", auth(s.handleSetCompTier))

// Portal: Leads
mux.HandleFunc("GET /api/v1/portal/agents/{discordID}/leads", auth(s.handleGetLeads))
mux.HandleFunc("POST /api/v1/portal/agents/{discordID}/leads", auth(s.handleCreateLead))
mux.HandleFunc("PUT /api/v1/portal/agents/{discordID}/leads/{leadID}", auth(s.handleUpdateLead))
mux.HandleFunc("DELETE /api/v1/portal/agents/{discordID}/leads/{leadID}", auth(s.handleDeleteLead))

// Portal: Training
mux.HandleFunc("GET /api/v1/portal/agents/{discordID}/training", auth(s.handleGetTraining))
mux.HandleFunc("POST /api/v1/portal/agents/{discordID}/training", auth(s.handleCreateTraining))
mux.HandleFunc("PUT /api/v1/portal/agents/{discordID}/training/{itemID}", auth(s.handleUpdateTraining))
mux.HandleFunc("DELETE /api/v1/portal/agents/{discordID}/training/{itemID}", auth(s.handleDeleteTraining))

// Portal: Schedule
mux.HandleFunc("GET /api/v1/portal/agents/{discordID}/schedule", auth(s.handleGetSchedule))
mux.HandleFunc("POST /api/v1/portal/agents/{discordID}/schedule", auth(s.handleCreateSchedule))
mux.HandleFunc("PUT /api/v1/portal/agents/{discordID}/schedule/{eventID}", auth(s.handleUpdateSchedule))
mux.HandleFunc("DELETE /api/v1/portal/agents/{discordID}/schedule/{eventID}", auth(s.handleDeleteSchedule))

// Portal: Org Chart
mux.HandleFunc("GET /api/v1/portal/org-chart", auth(s.handleGetOrgChart))
mux.HandleFunc("PUT /api/v1/portal/org-chart/assign-manager", auth(s.handleAssignManager))

// Portal: Comp Tiers — IMPORTANT: register /reorder BEFORE /{tierID} so it matches first
mux.HandleFunc("GET /api/v1/portal/comp-tiers", auth(s.handleGetCompTiers))
mux.HandleFunc("POST /api/v1/portal/comp-tiers", auth(s.handleCreateCompTier))
mux.HandleFunc("PUT /api/v1/portal/comp-tiers/reorder", auth(s.handleReorderCompTiers))
mux.HandleFunc("PUT /api/v1/portal/comp-tiers/{tierID}", auth(s.handleUpdateCompTier))
mux.HandleFunc("DELETE /api/v1/portal/comp-tiers/{tierID}", auth(s.handleDeleteCompTier))
```

---

## TASK 5: Update Electron App API Client

**File**: `../vipa-recruit-dashboard/src/services/api.ts`

Remove the ENTIRE `adminFetch` method and `getAdminBaseUrl` method. Change ALL portal methods to use the existing `this.fetch()` method (or a new `this.fetchWithBody()` for POST/PUT/DELETE) that goes through the Go bot's base URL.

### Changes:
1. Remove `getAdminBaseUrl()` method
2. Remove `adminFetch()` method
3. Add a generic `fetchMutate<T>(path, method, body?)` method:
```typescript
async fetchMutate<T>(path: string, method: string, body?: unknown): Promise<T> {
    const url = `${this.getBaseUrl()}${path}`
    const res = await fetch(url, {
        method,
        headers: this.getHeaders(),
        body: body ? JSON.stringify(body) : undefined,
    })
    if (!res.ok) {
        const text = await res.text().catch(() => 'Unknown error')
        throw new Error(`API ${res.status}: ${text}`)
    }
    // For DELETE responses that may be empty
    const text = await res.text()
    return text ? JSON.parse(text) : ({} as T)
}
```

4. Update ALL portal methods to use the new paths under `/api/v1/portal/...`:

```typescript
// Profile — was: this.adminFetch(`/api/profile/${agentId}`)
async getProfile(agentId: string): Promise<AgentProfile> {
    return this.fetch(`/api/v1/portal/profile/${agentId}`)
}
async updateProfile(agentId: string, data: Partial<AgentProfile>): Promise<AgentProfile> {
    return this.fetchMutate(`/api/v1/portal/profile/${agentId}`, 'PUT', data)
}
async setManager(agentId: string, managerId: string | null): Promise<AgentProfile> {
    return this.fetchMutate(`/api/v1/portal/profile/${agentId}/manager`, 'PUT', { managerId })
}
async setCompTier(agentId: string, compTierId: string | null): Promise<AgentProfile> {
    return this.fetchMutate(`/api/v1/portal/profile/${agentId}/comp-tier`, 'PUT', { compTierId })
}

// Leads — was: this.adminFetch(`/api/agents/${agentId}/leads`)
async getAgentLeads(agentId: string, params?: { status?: string }): Promise<AgentLead[]> {
    const query = new URLSearchParams()
    if (params?.status) query.set('status', params.status)
    const qs = query.toString()
    return this.fetch(`/api/v1/portal/agents/${agentId}/leads${qs ? `?${qs}` : ''}`)
}
async createAgentLead(agentId: string, data: CreateAgentLeadRequest): Promise<AgentLead> {
    return this.fetchMutate(`/api/v1/portal/agents/${agentId}/leads`, 'POST', data)
}
async updateAgentLead(agentId: string, leadId: string, data: UpdateAgentLeadRequest): Promise<AgentLead> {
    return this.fetchMutate(`/api/v1/portal/agents/${agentId}/leads/${leadId}`, 'PUT', data)
}
async deleteAgentLead(agentId: string, leadId: string): Promise<void> {
    await this.fetchMutate(`/api/v1/portal/agents/${agentId}/leads/${leadId}`, 'DELETE')
}

// Training — same pattern, path: /api/v1/portal/agents/{discordID}/training[/{itemID}]
// Schedule — same pattern, path: /api/v1/portal/agents/{discordID}/schedule[/{eventID}]
// Org Chart — path: /api/v1/portal/org-chart
// Comp Tiers — path: /api/v1/portal/comp-tiers[/{tierID}] and /api/v1/portal/comp-tiers/reorder
```

5. Update ALL remaining portal methods following the same pattern (training, schedule, org-chart, comp-tiers). Every `this.adminFetch(...)` call becomes either `this.fetch(...)` for GETs or `this.fetchMutate(...)` for POST/PUT/DELETE, with the new `/api/v1/portal/...` paths.

---

## TASK 6: Remove Admin Portal URL from Electron Settings

**File**: `../vipa-recruit-dashboard/src/services/storage.ts`
- Remove `adminApiUrl` from `ServerAccount` interface
- Remove `adminApiUrl` from `loadSettings()` return
- Remove any `adminApiUrl` default values

**File**: `../vipa-recruit-dashboard/src/pages/SettingsPage.tsx`
- Remove the "Admin Portal Connection" card/section that was added for configuring the admin portal URL
- The portal features now go through the same Go bot URL that's already configured

---

## TASK 7: Build & Verify

1. In `license-bot/`: Run `go build ./...` — must compile with zero errors
2. In `vipa-recruit-dashboard/`: Run `npx tsc --noEmit` — must pass with zero type errors
3. Verify no references to `adminFetch` or `adminApiUrl` remain in the Electron app

---

## Summary of files:

### Go bot (license-bot/):
- **MODIFIED**: `db/db.go` — Add 6 new migration statements to migrate()
- **NEW**: `db/portal.go` — All portal DB types + CRUD methods (~400 lines)
- **NEW**: `api/portal.go` — All portal HTTP handlers (~500 lines)
- **MODIFIED**: `api/server.go` — Register 20 new routes in NewServer()

### Electron app (vipa-recruit-dashboard/):
- **MODIFIED**: `src/services/api.ts` — Remove adminFetch, add fetchMutate, update all portal paths
- **MODIFIED**: `src/services/storage.ts` — Remove adminApiUrl
- **MODIFIED**: `src/pages/SettingsPage.tsx` — Remove Admin Portal Connection card
