# VIPA Advanced Dashboard — Claude Code Prompt

You are working on the VIPA License Bot (Go) and VIPA Recruit Dashboard (Electron/React).

## CODEBASE CONTEXT

### Go Bot (`license-bot/`)
- **Go 1.22+** with `net/http` routing patterns (`GET /api/v1/...`)
- **PostgreSQL** via `database/sql` + `github.com/lib/pq`, params use `$1, $2`
- **Discord**: `github.com/bwmarrin/discordgo` — session accessible via `bot.Session()` method
- **Auth**: Bearer token in `Authorization` header, validated by `s.authMiddleware`
- **WebSocket**: `api/websocket/` — Hub broadcasts events to all connected dashboard clients
- **Config**: `config/config.go` loads from env vars. Key fields: `GuildID`, `LicensedAgentRoleID`, `StudentRoleID`, `APIToken`
- **API Server struct** (`api/server.go`): has `cfg *config.Config`, `db *db.DB`, `discord *discordgo.Session`, `hub *websocket.Hub`
- **Bot struct** (`bot/bot.go`): has `cfg`, `db`, `session`, `registry *scrapers.Registry`, `hub`
- **Existing `performVerification`** in `bot/verify.go:432`: `func (b *Bot) performVerification(ctx context.Context, firstName, lastName, state string, discordID, guildID int64) VerifyResult`
- **UpsertAgent** in `db/db.go:443`: `func (d *DB) UpsertAgent(ctx context.Context, discordID, guildID int64, updates AgentUpdate) error` — INSERT ON CONFLICT DO NOTHING then dynamic UPDATE
- **Stage constants** in `db/db.go`: StageWelcome=1, StageFormStart=2, StageSorted=3, StageStudent=4, StageVerified=5, StageContracting=6, StageSetup=7, StageActive=8
- **Existing role assignment** in `bot/verify.go:401`: `s.GuildMemberRoleAdd(guildID, userID, roleID)` / `s.GuildMemberRoleRemove(guildID, userID, roleID)`
- **Routes registered** in `api/server.go` NewServer() — all under `mux.HandleFunc(...)` before `handler := s.corsMiddleware(mux)` on line 115

### Electron Dashboard (`vipa-recruit-dashboard/`)
- **React + TypeScript + Vite**, Tailwind CSS
- **API client**: `src/services/api.ts` — `VIPAApi` class with `fetch<T>(path)` for GET and `fetchMutate<T>(path, method, body?)` for POST/PUT/DELETE
- **Types**: `src/types/portal.ts` — OrgChartNode, AgentProfile, CompTier, etc.
- **OrgChartPage**: `src/pages/OrgChartPage.tsx` — uses ReactFlow with dagre layout, drag-to-connect for manager assignment
- **Hooks pattern**: `src/hooks/useOrgChart.ts` — React Query hooks for data fetching
- **Existing agents page**: `src/pages/AgentsPage.tsx` — lists agents with stage/agency filters

## IMPORTANT PATTERNS TO FOLLOW

### Go API Handlers
All handlers follow this exact pattern (see `api/portal.go` for examples):
```go
func (s *Server) handleSomething(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    // ... logic ...
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(result)
}
```

### Route Registration
Add new routes in `api/server.go` NewServer(), BEFORE `handler := s.corsMiddleware(mux)` (line 115). Use the `auth` wrapper for protected routes.

### WebSocket Events
Broadcast via `s.hub.Broadcast(websocket.NewEvent(eventType, data).JSON())`. Define new event types and data structs in `api/websocket/events.go`.

### Electron API calls
- GET: `this.fetch<T>(path)`
- POST/PUT/DELETE: `this.fetchMutate<T>(path, method, body)`
- All paths start with `/api/v1/...`

### JSON Output
All Go API responses MUST use camelCase JSON tags matching the TypeScript interfaces in `types/portal.ts`.

---

## TASK 1: Discord Member Lookup Endpoint

Create `GET /api/v1/discord/members/{discordID}` that checks if a Discord user exists in the guild.

### Go Backend (`api/discord.go` — NEW FILE)

```go
// Handler: handleDiscordMemberLookup
// Uses s.discord.GuildMember(s.cfg.GuildID, discordID) to fetch member
// Returns JSON:
// {
//   "exists": true,
//   "discordId": "123456789",
//   "username": "JohnDoe",
//   "displayName": "John",
//   "avatarUrl": "https://cdn.discordapp.com/avatars/...",
//   "roles": ["role_id_1", "role_id_2"],
//   "joinedAt": "2024-01-15T..."
// }
// If member not found, return: { "exists": false, "discordId": "123456789" }
// Do NOT return 404 — always return 200 with exists: true/false
```

Build the avatar URL: `fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", member.User.ID, member.User.Avatar)`

If `member.User.Avatar` is empty, use Discord's default avatar: `fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", discriminatorInt%5)` (or just return empty string).

Register route in `api/server.go`:
```go
mux.HandleFunc("GET /api/v1/discord/members/{discordID}", auth(s.handleDiscordMemberLookup))
```

### Electron Frontend

Add to `src/services/api.ts`:
```typescript
interface DiscordMemberLookup {
  exists: boolean
  discordId: string
  username?: string
  displayName?: string
  avatarUrl?: string
  roles?: string[]
  joinedAt?: string
}

async lookupDiscordMember(discordId: string): Promise<DiscordMemberLookup> {
  return this.fetch<DiscordMemberLookup>(`/api/v1/discord/members/${discordId}`)
}
```

---

## TASK 2: Add Agent from Dashboard Endpoint

Create `POST /api/v1/agents` that adds a new agent to the system.

### Go Backend (`api/discord.go` — same file as Task 1)

```go
// Handler: handleCreateAgent
// 1. Parse JSON body: { "discordId": "123...", "firstName": "John", "lastName": "Doe", "state": "FL", "agency": "TFC" }
//    - discordId, firstName, lastName are REQUIRED. state and agency are optional.
// 2. Validate member exists in guild: s.discord.GuildMember(s.cfg.GuildID, discordID)
//    - If not found, return 400: {"error": "User not found in Discord server"}
// 3. Check if agent already exists in DB: s.db.GetAgent(ctx, discordIDint64)
//    - If exists, return 409: {"error": "Agent already exists", "agent": existingAgent}
// 4. Create agent via s.db.UpsertAgent(ctx, discordIDint64, guildIDint64, AgentUpdate{FirstName, LastName, State, Agency})
//    - Set initial stage to StageWelcome (1)
// 5. Also create an agent_profile row: s.db.GetOrCreateProfile(ctx, discordIDstr) — this already exists in db/portal.go
// 6. Broadcast WebSocket event: EventAgentJoined with AgentJoinedData{DiscordID, Username, Stage: 1}
// 7. Return the created agent JSON (fetch it back with GetAgent)
```

Register route:
```go
mux.HandleFunc("POST /api/v1/agents", auth(s.handleCreateAgent))
```

---

## TASK 3: Dashboard-Triggered Verification Endpoint

Create `POST /api/v1/agents/{discordID}/verify` that triggers license verification from the dashboard.

### Architecture Decision
The `performVerification` function lives on the `Bot` struct, not the `Server` struct. The API server needs access to the bot's scraper registry.

**Solution**: Add the scraper registry to the API server. In `api/server.go`:
1. Add `registry *scrapers.Registry` field to Server struct
2. Update `NewServer` to accept and store the registry
3. Update the call site in `main.go` to pass `bot.Registry()` (you'll need to add a `Registry()` method to Bot)

Alternatively, create a standalone verify function in a shared package. Pick whichever is cleaner — the key is the API server must be able to call scraper lookups.

### Go Backend (`api/verify.go` — NEW FILE)

```go
// Handler: handleVerifyAgent
// 1. Parse discordID from URL path
// 2. Parse optional JSON body: { "firstName": "John", "lastName": "Doe", "state": "FL" }
//    - If body is empty/missing fields, try to load from DB: s.db.GetAgent(ctx, discordIDint64)
//    - Use agent's stored first_name, last_name, state as defaults
//    - If still missing firstName, lastName, or state, return 400: {"error": "firstName, lastName, and state are required"}
// 3. Get scraper for state: s.registry.GetScraper(state)
// 4. Call scraper.LookupByName(ctx, firstName, lastName) — same logic as bot/verify.go performVerification
// 5. If match found (prefer life-licensed active, then any active):
//    a. Update DB: s.db.UpsertAgent with LicenseVerified=true, LicenseNPN=match.NPN, CurrentStage=StageVerified
//    b. Save license check: s.db.SaveLicenseCheck(...)
//    c. Assign Discord role: s.discord.GuildMemberRoleAdd(s.cfg.GuildID, discordIDstr, s.cfg.LicensedAgentRoleID)
//    d. Remove student role: s.discord.GuildMemberRoleRemove(s.cfg.GuildID, discordIDstr, s.cfg.StudentRoleID)
//    e. Mark deadline verified: s.db.MarkDeadlineVerified(ctx, discordIDint64)
//    f. Broadcast EventLicenseVerified websocket event
//    g. Broadcast EventStageChanged websocket event (newStage = StageVerified)
//    h. Return 200: {"verified": true, "npn": "...", "licenseNumber": "...", "licenseType": "...", "status": "...", "expirationDate": "...", "loas": [...]}
// 6. If no match found:
//    a. Set stage to StageStudent (4) if currently below StageStudent
//    b. Create 30-day verification deadline: s.db.CreateDeadline(...)
//    c. Broadcast EventStageChanged if stage changed
//    d. Return 200: {"verified": false, "error": "No active license found for [firstName] [lastName] in [state]"}
```

Register route:
```go
mux.HandleFunc("POST /api/v1/agents/{discordID}/verify", auth(s.handleVerifyAgent))
```

---

## TASK 4: Bulk Import — Scan Discord Server

Create `POST /api/v1/agents/bulk-import` that scans all guild members and imports them.

### Go Backend (`api/discord.go` — same file)

```go
// Handler: handleBulkImport
// 1. Fetch all guild members using pagination:
//    Use s.discord.GuildMembers(s.cfg.GuildID, "", 1000) to get first batch
//    Then loop: s.discord.GuildMembers(s.cfg.GuildID, lastUserID, 1000) until len(members) < 1000
// 2. Filter: skip bots (member.User.Bot == true)
// 3. For each member:
//    a. Check if agent already exists: s.db.GetAgent(ctx, discordIDint64)
//    b. If NOT exists: create via UpsertAgent with FirstName from member.User.GlobalName or member.User.Username
//    c. Also GetOrCreateProfile
//    d. Track: imported count, skipped count (already exists), bot count (skipped)
// 4. Broadcast one EventAgentJoined for each newly imported agent (or a single bulk event)
// 5. Return: {"imported": 15, "skipped": 42, "bots": 3, "total": 60}
```

**IMPORTANT**: Use a longer timeout for this handler (30s instead of 5s). Discord rate limits GuildMembers to about 10 req/10s, so large servers may take time.

Register route:
```go
mux.HandleFunc("POST /api/v1/agents/bulk-import", auth(s.handleBulkImport))
```

---

## TASK 5: New WebSocket Events

Add these new event types to `api/websocket/events.go`:

```go
const (
    // ... existing events ...
    EventAgentCreated     = "agent_created"      // Dashboard-created agent
    EventAgentVerified    = "agent_verified"      // Dashboard-triggered verification result
    EventBulkImport       = "bulk_import"         // Bulk import completed
)

type AgentCreatedData struct {
    DiscordID   string `json:"discord_id"`
    Username    string `json:"username"`
    DisplayName string `json:"display_name"`
    AvatarURL   string `json:"avatar_url"`
    Stage       int    `json:"stage"`
    CreatedBy   string `json:"created_by"` // "dashboard"
}

type AgentVerifiedFromDashboard struct {
    DiscordID string `json:"discord_id"`
    Verified  bool   `json:"verified"`
    NPN       string `json:"npn,omitempty"`
    NewStage  int    `json:"new_stage"`
}

type BulkImportData struct {
    Imported int `json:"imported"`
    Skipped  int `json:"skipped"`
    Total    int `json:"total"`
}
```

---

## TASK 6: Electron Dashboard — Add Agent Modal

Create a new component `src/components/AddAgentModal.tsx`:

### UI Design
- Trigger: "Add Agent" button in the top-right of OrgChartPage (next to the page title)
- Modal with dark theme (bg-zinc-900, border-zinc-700) matching existing app style
- Fields:
  1. **Discord User ID** (text input, required) — with a "Verify" button next to it
     - On click "Verify": calls `api.lookupDiscordMember(id)`
     - If found: shows green checkmark, username, avatar preview, and populates display name
     - If not found: shows red X with "User not found in server"
  2. **First Name** (text input, required)
  3. **Last Name** (text input, required)
  4. **State** (dropdown of US state codes, optional) — 2-letter codes
  5. **Agency** (dropdown matching existing agencies: TFC, Radiant, GBU, TruLight, Thrive, The Point, Synergy, Illuminate, Elite One)
  6. **Auto-verify license** (checkbox, default checked) — if checked and state is provided, triggers verification after creation
- Submit button: "Add Agent"
- On submit:
  1. Call `POST /api/v1/agents` with the form data
  2. If auto-verify is checked AND state is provided: immediately call `POST /api/v1/agents/{discordID}/verify`
  3. Show toast notification: "Agent added successfully" (green) or "Agent added — license not found, moved to Pre-License" (yellow)
  4. Close modal and refresh org chart

### Integration with OrgChartPage
Add the button and modal to `src/pages/OrgChartPage.tsx`:
- Import the AddAgentModal component
- Add state: `const [showAddAgent, setShowAddAgent] = useState(false)`
- Add button in the header area (top-right):
```tsx
<button onClick={() => setShowAddAgent(true)} className="flex items-center gap-2 px-3 py-1.5 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm">
  <IconPlus size={16} /> Add Agent
</button>
```

### API Methods to Add (`src/services/api.ts`)
```typescript
interface CreateAgentRequest {
  discordId: string
  firstName: string
  lastName: string
  state?: string
  agency?: string
}

interface CreateAgentResponse {
  // full agent object returned from server
  discordId: string
  firstName: string
  lastName: string
  currentStage: number
  // ... other agent fields
}

interface VerifyAgentRequest {
  firstName?: string
  lastName?: string
  state?: string
}

interface VerifyAgentResponse {
  verified: boolean
  npn?: string
  licenseNumber?: string
  licenseType?: string
  status?: string
  expirationDate?: string
  loas?: string[]
  error?: string
}

async createAgent(data: CreateAgentRequest): Promise<CreateAgentResponse> {
  return this.fetchMutate<CreateAgentResponse>('/api/v1/agents', 'POST', data)
}

async verifyAgent(discordId: string, data?: VerifyAgentRequest): Promise<VerifyAgentResponse> {
  return this.fetchMutate<VerifyAgentResponse>(`/api/v1/agents/${discordId}/verify`, 'POST', data || {})
}

async bulkImportAgents(): Promise<{ imported: number; skipped: number; total: number }> {
  return this.fetchMutate<{ imported: number; skipped: number; total: number }>('/api/v1/agents/bulk-import', 'POST', {})
}
```

---

## TASK 7: Agent Profile Card on Org Chart

When clicking an agent node on the org chart, show a slide-out panel or expanded card with:

### Panel Contents
1. **Header**: Avatar (from Discord or profile photoUrl), full name, stage badge with color
2. **Discord Info**: Username, Discord ID, roles in server
3. **License Details** (if verified): NPN, license number, type, status, expiration, LOAs
4. **Profile Quick Stats**:
   - Leads count (fetch from `/api/v1/portal/agents/{id}/leads`)
   - Training progress (completed / total from `/api/v1/portal/agents/{id}/training`)
   - Comp tier name + percentage
5. **Quick Actions**:
   - "Verify License" button (calls POST verify endpoint) — only show if not yet verified (stage < 5)
   - "Send Message" button (calls existing POST `/api/v1/agents/{discordID}/message`)
   - "Change Stage" dropdown (calls existing POST `/api/v1/agents/{discordID}/stage`)

### Implementation
- Create `src/components/AgentProfilePanel.tsx`
- On OrgChartPage, add state: `const [selectedAgent, setSelectedAgent] = useState<string | null>(null)`
- On node click: `setSelectedAgent(agentId)`
- Panel slides in from the right (absolute positioned, z-50, w-80 or w-96)
- Fetch full agent data on open: `GET /api/v1/agents/{discordID}` + `GET /api/v1/portal/profile/{discordID}`

---

## TASK 8: Bulk Import Button

Add a "Scan Server" button to the AgentsPage (or a settings/admin section):

- Button text: "Import from Discord"
- On click: show confirmation dialog: "This will scan your Discord server and import all members as agents. Already-tracked agents will be skipped."
- On confirm: call `POST /api/v1/agents/bulk-import`
- Show progress/result: "Imported 15 new agents, 42 already tracked, 3 bots skipped"
- Refresh agent list after import

Add the button to `src/pages/AgentsPage.tsx` in the header area.

---

## TASK 9: Real-Time WebSocket Updates for Org Chart

The WebSocket connection already exists in the dashboard. Update the OrgChartPage to listen for real-time events:

### In `src/pages/OrgChartPage.tsx` or a new hook `src/hooks/useRealtimeOrgChart.ts`:

1. Listen for `agent_created` events → add new node to the org chart without full refresh
2. Listen for `agent_verified` / `license_verified` events → update the agent's stage badge color
3. Listen for `stage_changed` events → update stage badge
4. Listen for `bulk_import` events → show toast and trigger full refresh

Use the existing WebSocket connection from the app. If there's a `useWebSocket` hook, use it. Otherwise, check how the dashboard currently subscribes to WS events and follow that pattern.

### Invalidate React Query Cache
When WS events arrive, invalidate the org chart query so it refetches:
```typescript
const queryClient = useQueryClient()
// On receiving event:
queryClient.invalidateQueries({ queryKey: ['org-chart'] })
```

---

## TASK 10: Build Verification

After completing all tasks:

1. Run `go build ./...` in the `license-bot/` directory. Fix ALL compile errors before proceeding.
2. Run `cd vipa-recruit-dashboard && npm run build`. Fix ALL TypeScript errors.
3. Verify no unused imports remain.
4. Ensure all new files have proper package declarations and imports.

**CRITICAL**: Do NOT modify any existing functionality. Only ADD new endpoints, components, and features. Existing routes, handlers, and pages must continue working exactly as before.

---

## FILE SUMMARY

### New Go Files:
- `api/discord.go` — Discord member lookup, create agent, bulk import handlers
- `api/verify.go` — Dashboard-triggered verification handler

### Modified Go Files:
- `api/server.go` — Add new route registrations + registry field on Server struct
- `api/websocket/events.go` — Add new event types and data structs
- `bot/bot.go` — Add `Registry()` method to expose scraper registry
- `main.go` — Pass registry to NewServer (check how main.go creates the server)

### New Electron Files:
- `src/components/AddAgentModal.tsx` — Add agent modal with Discord ID verification
- `src/components/AgentProfilePanel.tsx` — Slide-out agent profile card

### Modified Electron Files:
- `src/services/api.ts` — Add lookupDiscordMember, createAgent, verifyAgent, bulkImportAgents methods + types
- `src/pages/OrgChartPage.tsx` — Add "Add Agent" button, modal state, agent profile panel, real-time WS updates
- `src/pages/AgentsPage.tsx` — Add "Import from Discord" button
- `src/types/portal.ts` — Add DiscordMemberLookup, CreateAgentRequest, VerifyAgentResponse types
