# VIPA License Bot — Complete Implementation Plan

> **For Claude Code / AI-assisted development**
> This document contains EVERYTHING needed to implement all 6 phases of the VIPA Platform.
> Every file, every function, every database table, every endpoint, every slash command.

---

## TABLE OF CONTENTS

1. [Current Architecture](#current-architecture)
2. [Target File Structure](#target-file-structure)
3. [Phase 1: License Verify Log + All 56 Jurisdictions](#phase-1)
4. [Phase 2: License Tracker + Recruiter Nudge System](#phase-2)
5. [Phase 3: Agency Owner Approval + Manager Assignment](#phase-3)
6. [Phase 4: GoHighLevel CRM Integration](#phase-4)
7. [Phase 5: Leaderboard + Activity Tracking](#phase-5)
8. [Phase 6: Mobile UX + Zoom Verticals + Role Cleanup](#phase-6)
9. [Database Schema (Complete)](#database-schema)
10. [API Endpoints (Complete)](#api-endpoints)
11. [Slash Commands (Complete)](#slash-commands)
12. [Environment Variables](#environment-variables)
13. [Implementation Order](#implementation-order)
14. [Verification Checkpoints](#verification-checkpoints)

---

## CURRENT ARCHITECTURE

### Existing Go Files (34 total)

```
license-bot/
├── main.go                          # Entry point — starts bot + API
├── go.mod / go.sum                  # Dependencies
├── Dockerfile                       # Multi-stage Docker build
├── config/
│   └── config.go                    # Env var loader, agency role mapping
├── bot/
│   ├── bot.go                       # Bot struct, Run(), interaction router
│   ├── commands.go                  # Slash command registration
│   ├── onboarding.go                # Member join → welcome DM
│   ├── intake.go                    # 2-step onboarding form (modals)
│   ├── verify.go                    # /verify command + performVerification()
│   ├── auto_verify.go               # Background auto-verification
│   ├── npn_lookup.go                # /npn multi-state NPN search
│   ├── scheduler.go                 # 24h scheduler (reminders, deadlines, checkins)
│   ├── checkins.go                  # Weekly check-in prompts + responses
│   ├── activation.go                # Stage 8 activation
│   ├── admin.go                     # Admin panel commands
│   ├── email_prefs.go               # /email-optin, /email-optout
│   ├── embeds.go                    # All Discord embed builders
│   └── history.go                   # /license-history command
├── db/
│   ├── db.go                        # Connection, pool, migrations, core ops
│   ├── agents.go                    # Agent queries (list, search, counts)
│   ├── checkins.go                  # Check-in DB operations
│   ├── contracting.go               # Contracting manager CRUD
│   └── setup.go                     # Setup checklist progress
├── api/
│   ├── server.go                    # HTTP server, routing, auth middleware
│   ├── agents.go                    # Agent list/detail endpoints
│   └── dashboard.go                 # Funnel, summary, at-risk endpoints
├── scrapers/
│   ├── types.go                     # LicenseResult struct + Scraper interface
│   ├── registry.go                  # State → scraper routing
│   ├── naic.go                      # NAIC SBS API (31 states)
│   ├── florida.go                   # FL custom scraper
│   ├── california.go                # CA custom scraper (Turnstile CAPTCHA)
│   ├── texas.go                     # TX custom scraper (CAPTCHA)
│   └── captcha/
│       └── capsolver.go             # CapSolver API for Turnstile/reCAPTCHA
├── email/
│   └── resend.go                    # Resend email client
└── tlsclient/
    └── client.go                    # TLS client with Chrome fingerprint
```

### Current Scraper Coverage

| Category | States | Count |
|----------|--------|-------|
| Custom scrapers | FL, CA, TX | 3 |
| NAIC SBS API | AL, AK, AZ, AR, CT, DE, DC, HI, ID, IL, IA, KS, MA, MD, MO, MT, NE, NH, NJ, NM, NC, ND, OK, OR, RI, SC, SD, TN, VT, WI, WV | 31 |
| Manual fallback | CO, GA, IN, KY, LA, ME, MI, MN, MS, NV, NY, OH, PA, UT, VA, WA, WY | 17 |
| **Territories (missing)** | PR, GU, VI, MP, AS | 5 |
| **Total** | | **56** |

### Current Database Tables

1. `license_checks` — Verification history
2. `onboarding_agents` — Agent profiles (main table)
3. `verification_deadlines` — 30-day deadline tracking
4. `agent_activity_log` — Activity events
5. `agent_weekly_checkins` — Check-in records
6. `contracting_managers` — Manager info
7. `agent_setup_progress` — Setup checklist

### Current API Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| GET | `/api/v1/health` | No | Health check |
| GET | `/api/v1/agents` | Yes | List agents |
| GET | `/api/v1/agents/{id}` | Yes | Agent detail |
| GET | `/api/v1/dashboard/funnel` | Yes | Stage counts |
| GET | `/api/v1/dashboard/summary` | Yes | Summary metrics |
| GET | `/api/v1/dashboard/at-risk` | Yes | At-risk agents |

### Current Slash Commands

| Command | Handler | Purpose |
|---------|---------|---------|
| `/verify` | `handleVerify` | License verification |
| `/npn` | `handleNPNLookup` | NPN lookup |
| `/license-history` | `handleHistory` | Check history |
| `/contract` | `handleContract` | Book contracting |
| `/email-optin` | `handleEmailOptIn` | Opt in to email |
| `/email-optout` | `handleEmailOptOut` | Opt out of email |
| `/setup` | `handleSetup` | Setup checklist |
| `/agent` | `handleAgentCommand` | Admin agent mgmt |
| `/contracting` | `handleContractingCommand` | Manage contracting managers |
| `/restart` | `handleRestart` | Restart onboarding |
| `/onboarding-setup` | `handleOnboardingSetup` | Post onboarding message |
| `/setup-rules` | `handleSetupRules` | Post rules embed |

---

## TARGET FILE STRUCTURE

```
license-bot/
├── main.go                                # MODIFY — add GHL sync goroutine
├── go.mod / go.sum                        # MODIFY — add new dependencies
├── Dockerfile                             # NO CHANGE
│
├── config/
│   └── config.go                          # MODIFY — new env vars (see §12)
│
├── bot/
│   ├── bot.go                             # MODIFY — add new handlers
│   ├── commands.go                        # MODIFY — register new commands
│   ├── onboarding.go                      # MODIFY — trigger approval flow
│   ├── intake.go                          # MODIFY — collect manager field
│   ├── verify.go                          # MODIFY — post to new log channel
│   ├── auto_verify.go                     # NO CHANGE
│   ├── npn_lookup.go                      # NO CHANGE
│   ├── scheduler.go                       # MODIFY — add nudge + tracker jobs
│   ├── checkins.go                        # NO CHANGE
│   ├── activation.go                      # NO CHANGE
│   ├── admin.go                           # MODIFY — add /agent assign-manager
│   ├── email_prefs.go                     # NO CHANGE
│   ├── embeds.go                          # MODIFY — new embed builders
│   ├── history.go                         # NO CHANGE
│   │
│   │ ── NEW FILES ──
│   ├── approval.go                        # NEW — agency owner approval flow
│   ├── tracker.go                         # NEW — license tracker embed + commands
│   ├── leaderboard.go                     # NEW — leaderboard commands + embeds
│   ├── activity_log.go                    # NEW — /log command for activity logging
│   └── zoom.go                            # NEW — Zoom vertical management
│
├── db/
│   ├── db.go                              # MODIFY — new migrations
│   ├── agents.go                          # MODIFY — new query methods
│   ├── checkins.go                        # NO CHANGE
│   ├── contracting.go                     # NO CHANGE
│   ├── setup.go                           # NO CHANGE
│   │
│   │ ── NEW FILES ──
│   ├── approval.go                        # NEW — approval request CRUD
│   ├── tracker.go                         # NEW — tracker queries (licensed counts)
│   ├── leaderboard.go                     # NEW — leaderboard aggregation queries
│   └── activity.go                        # NEW — activity logging queries
│
├── api/
│   ├── server.go                          # MODIFY — add new routes
│   ├── agents.go                          # MODIFY — add manager field
│   ├── dashboard.go                       # MODIFY — add tracker endpoint
│   │
│   │ ── NEW FILES ──
│   ├── tracker.go                         # NEW — tracker API endpoints
│   ├── leaderboard.go                     # NEW — leaderboard API endpoints
│   └── approval.go                        # NEW — approval API endpoints
│
├── scrapers/
│   ├── types.go                           # NO CHANGE
│   ├── registry.go                        # MODIFY — add SIRCON + state DOI scrapers
│   ├── naic.go                            # NO CHANGE
│   ├── florida.go                         # NO CHANGE
│   ├── california.go                      # NO CHANGE
│   ├── texas.go                           # NO CHANGE
│   │
│   │ ── NEW FILES ──
│   ├── sircon.go                          # NEW — Vertafore SIRCON scraper (8 states)
│   ├── nipr.go                            # NEW — NIPR PDB lookup (backup)
│   ├── colorado.go                        # NEW — CO DOI scraper
│   ├── georgia.go                         # NEW — GA DOI scraper
│   ├── indiana.go                         # NEW — IN DOI scraper
│   ├── kentucky.go                        # NEW — KY DOI scraper
│   ├── louisiana.go                       # NEW — LA DOI scraper
│   ├── michigan.go                        # NEW — MI DOI scraper
│   ├── minnesota.go                       # NEW — MN DOI scraper
│   ├── mississippi.go                     # NEW — MS DOI scraper
│   ├── nevada.go                          # NEW — NV DOI scraper
│   ├── newyork.go                         # NEW — NY DFS scraper
│   ├── ohio.go                            # NEW — OH DOI scraper
│   ├── pennsylvania.go                    # NEW — PA DOI scraper
│   ├── utah.go                            # NEW — UT DOI scraper
│   ├── virginia.go                        # NEW — VA BOI scraper
│   ├── washington.go                      # NEW — WA OIC scraper
│   ├── wyoming.go                         # NEW — WY DOI scraper
│   ├── maine.go                           # NEW — ME DOI scraper
│   └── territories.go                     # NEW — PR, GU, VI, MP, AS lookups
│   └── captcha/
│       └── capsolver.go                   # NO CHANGE
│
├── ghl/                                   # NEW PACKAGE — GoHighLevel CRM
│   ├── client.go                          # NEW — GHL API client
│   ├── contacts.go                        # NEW — Contact sync
│   ├── pipeline.go                        # NEW — Pipeline stage mapping
│   └── webhooks.go                        # NEW — GHL webhook handlers
│
├── sms/                                   # NEW PACKAGE — SMS (Twilio)
│   └── twilio.go                          # NEW — Twilio SMS client
│
├── email/
│   └── resend.go                          # MODIFY — new email templates
│
└── tlsclient/
    └── client.go                          # NO CHANGE
```

---

## PHASE 1: License Verify Log + All 56 Jurisdictions
<a name="phase-1"></a>

### 1A. Separate License Verify Log Channel

**Goal:** Post verification results to channel `1474946201450184726` instead of the hiring log channel.

#### File: `config/config.go` — ADD

```go
// Add new field to Config struct:
LicenseVerifyLogChannelID string // env: LICENSE_VERIFY_LOG_CHANNEL_ID

// In MustLoad():
LicenseVerifyLogChannelID: os.Getenv("LICENSE_VERIFY_LOG_CHANNEL_ID"),
```

#### File: `bot/verify.go` — MODIFY `postToChannel()`

Current behavior: Posts to `LICENSE_CHECK_CHANNEL_ID` (falls back to `HIRING_LOG_CHANNEL_ID`).

New behavior: Post verification results to `LICENSE_VERIFY_LOG_CHANNEL_ID`. Keep the existing channel for other purposes.

```go
func (b *Bot) postVerifyToLogChannel(s *discordgo.Session, embed *discordgo.MessageEmbed) {
    channelID := b.cfg.LicenseVerifyLogChannelID
    if channelID == "" {
        channelID = b.cfg.LicenseCheckChannelID // fallback
    }
    _, err := s.ChannelMessageSendEmbed(channelID, embed)
    if err != nil {
        log.Printf("Failed to post verify log: %v", err)
    }
}
```

Update `handleVerify()` to call `postVerifyToLogChannel()` instead of existing `postToChannel()`.

#### File: `bot/embeds.go` — ADD

```go
func buildVerifyLogEmbed(agent *db.Agent, result *scrapers.LicenseResult) *discordgo.MessageEmbed {
    // Green embed for verified, red for failed
    // Fields: Name, NPN, State, License #, Status, Lines of Authority
    // Footer: timestamp
}
```

### 1B. SIRCON Scraper (8 States)

**States:** CO, GA, IN, KY, LA, MI, NV, OH

**Research required:** Use Perplexity MCP to search:
- `"Vertafore SIRCON producer lookup API" site:sircon.com OR site:vertafore.com`
- `"SIRCON agent search" insurance license lookup`

#### File: `scrapers/sircon.go` — NEW

```go
package scrapers

import (
    "fmt"
    "strings"
    "license-bot-go/tlsclient"
    "github.com/PuerkitoBio/goquery"
)

const sirconBaseURL = "https://sircon.com/ComplianceExpress/Inquiry/consumerInquiry.do"

type SIRCONScraper struct {
    stateCode string
    client    *tlsclient.Client
}

func NewSIRCONScraper(state string, client *tlsclient.Client) *SIRCONScraper {
    return &SIRCONScraper{stateCode: state, client: client}
}

func (s *SIRCONScraper) StateCode() string { return s.stateCode }

func (s *SIRCONScraper) LookupByName(firstName, lastName string) ([]LicenseResult, error) {
    // POST to sirconBaseURL with:
    //   state={stateCode}&lastName={lastName}&firstName={firstName}
    // Parse HTML results table
    // Extract: name, license number, NPN, status, type, expiration
    // Return []LicenseResult
}

func (s *SIRCONScraper) LookupByNPN(npn string) ([]LicenseResult, error) {
    // POST with npn={npn}&state={stateCode}
}

func (s *SIRCONScraper) LookupByLicenseNumber(licNum string) ([]LicenseResult, error) {
    // POST with licenseNumber={licNum}&state={stateCode}
}

func (s *SIRCONScraper) ManualLookupURL() string {
    return fmt.Sprintf("https://sircon.com/ComplianceExpress/Inquiry/consumerInquiry.do?state=%s", s.stateCode)
}
```

**Implementation notes:**
- SIRCON uses a form POST interface, parse response HTML with goquery
- Use Perplexity to verify the exact form fields and URL structure
- Each SIRCON state may have slightly different form parameters
- Test with a known agent name in each state

### 1C. State DOI Scrapers (9 States)

Each state Department of Insurance has its own web interface. Create one file per state.

**States:** ME, MN, MS, NY, PA, UT, VA, WA, WY

**For each state, use Perplexity MCP to research:**
```
"[State] department of insurance producer license lookup" site:.gov
```

#### Template for each state scraper:

```go
package scrapers

// File: scrapers/{state}.go

import (
    "license-bot-go/tlsclient"
    "github.com/PuerkitoBio/goquery"
)

type {State}Scraper struct {
    client *tlsclient.Client
}

func New{State}Scraper(client *tlsclient.Client) *{State}Scraper {
    return &{State}Scraper{client: client}
}

func (s *{State}Scraper) StateCode() string { return "{XX}" }

func (s *{State}Scraper) LookupByName(firstName, lastName string) ([]LicenseResult, error) {
    // 1. GET search page (establish cookies/CSRF)
    // 2. POST search form
    // 3. Parse HTML results
    // 4. Return []LicenseResult
}

func (s *{State}Scraper) LookupByNPN(npn string) ([]LicenseResult, error) {
    // If state supports NPN lookup, implement
    // Otherwise: return nil, fmt.Errorf("NPN lookup not supported for {XX}")
}

func (s *{State}Scraper) LookupByLicenseNumber(licNum string) ([]LicenseResult, error) {
    // If state supports license number lookup, implement
    // Otherwise: return nil, fmt.Errorf("License number lookup not supported for {XX}")
}

func (s *{State}Scraper) ManualLookupURL() string {
    return "{state DOI URL}"
}
```

#### State-specific URLs to research and implement:

| State | DOI URL | Notes |
|-------|---------|-------|
| ME | `https://pfr.maine.gov/ALMSOnline/` | Maine Bureau of Insurance |
| MN | `https://mn.gov/commerce/industries/insurance/` | MN Commerce Dept |
| MS | `https://www.mid.ms.gov/` | MS Insurance Dept |
| NY | `https://myportal.dfs.ny.gov/` | NY DFS — may require CAPTCHA |
| PA | `https://www.insurance.pa.gov/` | PA Insurance Dept |
| UT | `https://insurance.utah.gov/licensee-search` | UT Insurance Dept |
| VA | `https://scc.virginia.gov/boi/` | VA Bureau of Insurance |
| WA | `https://fortress.wa.gov/oic/` | WA Office of Insurance Commissioner |
| WY | `https://doi.wyo.gov/` | WY DOI |

**CRITICAL: Use Perplexity MCP for each state before implementing:**
```
perplexity_ask: "[State full name] insurance department producer license lookup URL 2025"
```

### 1D. Territories

#### File: `scrapers/territories.go` — NEW

```go
package scrapers

// Territories: PR, GU, VI, MP, AS
// Most territories don't have online lookup systems.
// Use NIPR PDB as primary, manual fallback otherwise.

type TerritoryScraper struct {
    stateCode string
}

func NewTerritoryScraper(code string) *TerritoryScraper {
    return &TerritoryScraper{stateCode: code}
}

func (t *TerritoryScraper) StateCode() string { return t.stateCode }

func (t *TerritoryScraper) LookupByName(firstName, lastName string) ([]LicenseResult, error) {
    return nil, fmt.Errorf("automated lookup not available for %s — use NIPR", t.stateCode)
}

func (t *TerritoryScraper) LookupByNPN(npn string) ([]LicenseResult, error) {
    return nil, fmt.Errorf("automated lookup not available for %s — use NIPR", t.stateCode)
}

func (t *TerritoryScraper) LookupByLicenseNumber(licNum string) ([]LicenseResult, error) {
    return nil, fmt.Errorf("automated lookup not available for %s — use NIPR", t.stateCode)
}

func (t *TerritoryScraper) ManualLookupURL() string {
    urls := map[string]string{
        "PR": "https://ocs.gobierno.pr/",
        "GU": "https://doa.guam.gov/",
        "VI": "https://ltg.gov.vi/division-of-banking-insurance-and-financial-regulation/",
        "MP": "", // No online system
        "AS": "", // No online system
    }
    if url, ok := urls[t.stateCode]; ok && url != "" {
        return url
    }
    return "https://nipr.com/products-and-services/pdb"
}
```

### 1E. NIPR PDB Lookup (Backup)

#### File: `scrapers/nipr.go` — NEW

**Research with Perplexity MCP:**
```
"NIPR producer database API" OR "NIPR PDB lookup" site:nipr.com
```

```go
package scrapers

// NIPR PDB (Producer Database) - National lookup
// Used as backup for states without automated scrapers
// Requires NIPR API credentials (paid service)
// If credentials not available, falls back to manual URL

type NIPRScraper struct {
    apiKey string
    client *tlsclient.Client
}

func NewNIPRScraper(apiKey string, client *tlsclient.Client) *NIPRScraper {
    return &NIPRScraper{apiKey: apiKey, client: client}
}

// Implement Scraper interface...
// API endpoint: https://pdb-xml.nipr.com/ (research exact URL with Perplexity)
```

### 1F. Update Registry

#### File: `scrapers/registry.go` — MODIFY

```go
// Replace ManualLookupURLs map with proper scrapers:

func NewRegistry(tlsClient *tlsclient.Client, cs *captcha.CapSolver) *Registry {
    r := &Registry{scrapers: make(map[string]Scraper)}

    // Custom scrapers (existing)
    r.scrapers["FL"] = NewFloridaScraper(tlsClient)
    r.scrapers["CA"] = NewCaliforniaScraper(tlsClient, cs)
    r.scrapers["TX"] = NewTexasScraper(tlsClient, cs)

    // NAIC SBS API (31 states — existing)
    for _, code := range NAICStates {
        r.scrapers[code] = NewNAICScraper(code, tlsClient)
    }

    // SIRCON scrapers (8 states — NEW)
    sirconStates := []string{"CO", "GA", "IN", "KY", "LA", "MI", "NV", "OH"}
    for _, code := range sirconStates {
        r.scrapers[code] = NewSIRCONScraper(code, tlsClient)
    }

    // State DOI scrapers (9 states — NEW)
    r.scrapers["ME"] = NewMaineScraper(tlsClient)
    r.scrapers["MN"] = NewMinnesotaScraper(tlsClient)
    r.scrapers["MS"] = NewMississippiScraper(tlsClient)
    r.scrapers["NY"] = NewNewYorkScraper(tlsClient, cs) // may need CAPTCHA
    r.scrapers["PA"] = NewPennsylvaniaScraper(tlsClient)
    r.scrapers["UT"] = NewUtahScraper(tlsClient)
    r.scrapers["VA"] = NewVirginiaScraper(tlsClient)
    r.scrapers["WA"] = NewWashingtonScraper(tlsClient)
    r.scrapers["WY"] = NewWyomingScraper(tlsClient)

    // Territories (manual fallback — NEW)
    for _, code := range []string{"PR", "GU", "VI", "MP", "AS"} {
        r.scrapers[code] = NewTerritoryScraper(code)
    }

    return r
}

// Remove ManualLookupURLs map entirely
// Remove ManualScraper struct — replaced by proper scrapers
```

### 1G. Verification Checkpoint for Phase 1

After implementing each scraper:
1. **Use Perplexity MCP** to verify the DOI website URL is correct and operational
2. **Write a test** in `scrapers/{state}_test.go` that searches for a known public agent name
3. **Verify the Scraper interface** is fully implemented (all 5 methods)
4. **Test via Discord** with `/verify state:{XX} name:Test Agent`
5. **Check registry** resolves the state code correctly
6. **Confirm log channel** posts to `1474946201450184726`

---

## PHASE 2: License Tracker + Recruiter Nudge System
<a name="phase-2"></a>

### 2A. License Tracker

**Goal:** Track licensing progress: "200/250 agents are licensed". Show overall, per-agency, per-recruiter, per-individual drill-down.

#### File: `db/tracker.go` — NEW

```go
package db

import "time"

type TrackerStats struct {
    TotalAgents    int
    LicensedAgents int
    Percentage     float64
}

type AgencyStats struct {
    Agency         string
    TotalAgents    int
    LicensedAgents int
    Percentage     float64
}

type RecruiterStats struct {
    RecruiterName     string
    RecruiterDiscordID string
    TotalRecruits     int
    LicensedRecruits  int
    Percentage        float64
    UnlicensedNames   []string // for drill-down
}

// GetOverallTrackerStats returns total vs licensed across all agents.
func (d *DB) GetOverallTrackerStats() (*TrackerStats, error) {
    // SELECT COUNT(*) as total,
    //        SUM(CASE WHEN verified = true THEN 1 ELSE 0 END) as licensed
    // FROM onboarding_agents
    // WHERE kicked_at IS NULL
}

// GetAgencyTrackerStats returns licensed counts per agency.
func (d *DB) GetAgencyTrackerStats() ([]AgencyStats, error) {
    // SELECT agency, COUNT(*), SUM(CASE WHEN verified THEN 1 ELSE 0 END)
    // FROM onboarding_agents
    // WHERE kicked_at IS NULL AND agency != ''
    // GROUP BY agency ORDER BY agency
}

// GetRecruiterTrackerStats returns per-recruiter licensing stats.
func (d *DB) GetRecruiterTrackerStats(agency string) ([]RecruiterStats, error) {
    // SELECT upline_manager, COUNT(*), SUM(verified)
    // FROM onboarding_agents
    // WHERE kicked_at IS NULL AND agency = $1
    // GROUP BY upline_manager
}

// GetUnlicensedByRecruiter returns unlicensed agents under a specific recruiter.
func (d *DB) GetUnlicensedByRecruiter(recruiterDiscordID string) ([]Agent, error) {
    // SELECT * FROM onboarding_agents
    // WHERE upline_manager_discord_id = $1 AND verified = false AND kicked_at IS NULL
}
```

#### File: `bot/tracker.go` — NEW

```go
package bot

import "github.com/bwmarrin/discordgo"

// handleTrackerCommand handles /tracker slash command.
// Subcommands: overview, agency, recruiter, individual
func (b *Bot) handleTrackerCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Route to subcommand handler
}

// handleTrackerOverview shows overall stats embed.
func (b *Bot) handleTrackerOverview(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Query GetOverallTrackerStats()
    // Build embed: "Licensed Agents: 200/250 (80%)"
    // Progress bar visualization: ████████░░ 80%
    // Show buttons: "By Agency" "By Recruiter"
}

// handleTrackerByAgency shows per-agency breakdown.
func (b *Bot) handleTrackerByAgency(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Query GetAgencyTrackerStats()
    // Build embed with each agency line:
    //   TFC: 50/60 (83%) ████████░░
    //   Radiant: 30/40 (75%) ███████░░░
    //   etc.
}

// handleTrackerByRecruiter shows per-recruiter under an agency.
func (b *Bot) handleTrackerByRecruiter(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Requires agency parameter
    // Query GetRecruiterTrackerStats(agency)
    // Show each recruiter's licensed/total
}

// postTrackerEmbed posts auto-updating tracker to a specific channel.
func (b *Bot) postTrackerEmbed(s *discordgo.Session, channelID string) {
    // Called by scheduler daily
    // Posts/updates a persistent embed with current stats
}
```

#### Slash Command Registration — `/tracker`

```go
{
    Name:        "tracker",
    Description: "License progress tracker",
    Options: []*discordgo.ApplicationCommandOption{
        {
            Type:        discordgo.ApplicationCommandOptionSubCommand,
            Name:        "overview",
            Description: "Overall licensing progress",
        },
        {
            Type:        discordgo.ApplicationCommandOptionSubCommand,
            Name:        "agency",
            Description: "Per-agency breakdown",
            Options: []*discordgo.ApplicationCommandOption{
                {
                    Type:        discordgo.ApplicationCommandOptionString,
                    Name:        "name",
                    Description: "Agency name (optional, shows all if empty)",
                },
            },
        },
        {
            Type:        discordgo.ApplicationCommandOptionSubCommand,
            Name:        "recruiter",
            Description: "Per-recruiter breakdown",
            Options: []*discordgo.ApplicationCommandOption{
                {
                    Type:        discordgo.ApplicationCommandOptionString,
                    Name:        "agency",
                    Description: "Agency to drill into",
                    Required:    true,
                },
            },
        },
    },
}
```

#### API Endpoint: `/api/v1/tracker`

```go
// File: api/tracker.go — NEW

// GET /api/v1/tracker/overview     → TrackerStats
// GET /api/v1/tracker/agencies     → []AgencyStats
// GET /api/v1/tracker/recruiters?agency=TFC → []RecruiterStats
// GET /api/v1/tracker/unlicensed?recruiter={discord_id} → []Agent
```

### 2B. Recruiter Nudge System (30-Day)

**Goal:** After 30 days, if a recruit isn't licensed, DM their upline manager.

#### File: `bot/scheduler.go` — MODIFY

Add new scheduled job:

```go
// Add to runSchedulerCycle():
b.sendRecruiterNudges()

// sendRecruiterNudges checks agents past 30 days without verification.
func (b *Bot) sendRecruiterNudges() {
    // 1. Query agents where:
    //    - stage < StageVerified (not yet licensed)
    //    - joined_at < 30 days ago
    //    - kicked_at IS NULL
    //    - last_nudge_sent_at IS NULL OR last_nudge_sent_at < 7 days ago
    //
    // 2. Group by upline_manager
    //
    // 3. For each recruiter:
    //    a. DM the recruiter via Discord
    //    b. List their unlicensed recruits with days since join
    //    c. Ask "Are they still pursuing licensing?"
    //    d. Update last_nudge_sent_at
    //
    // 4. If email opted in, also send via Resend
}
```

#### Database Change: `onboarding_agents` — ADD COLUMN

```sql
ALTER TABLE onboarding_agents ADD COLUMN last_nudge_sent_at TIMESTAMP;
ALTER TABLE onboarding_agents ADD COLUMN upline_manager_discord_id TEXT;
```

**Note:** `upline_manager_discord_id` is needed to DM the recruiter. Currently only stores name. Need to resolve Discord username → ID.

#### Recruiter Resolution Strategy

The `upline_manager` field stores a text name. To DM them, we need their Discord ID. Options:

1. **During onboarding** — When a new agent fills out the form, try to resolve their upline's Discord ID by searching guild members. Store in `upline_manager_discord_id`.
2. **Manual assignment** — Admin can use `/agent assign-manager @user agent_id` to link.
3. **Fallback** — If no Discord ID found, post to admin channel instead of DM.

```go
// In bot/intake.go, after form submission:
func (b *Bot) resolveUplineDiscordID(s *discordgo.Session, guildID, uplineName string) string {
    members, _ := s.GuildMembersSearch(guildID, uplineName, 5)
    for _, m := range members {
        if strings.EqualFold(m.User.Username, uplineName) ||
           strings.EqualFold(m.Nick, uplineName) {
            return m.User.ID
        }
    }
    return "" // not found
}
```

---

## PHASE 3: Agency Owner Approval + Manager Assignment
<a name="phase-3"></a>

### 3A. Approval System

**Goal:** When a user joins under a specific agency, the agency owner must approve them before they get sorted into that agency.

#### New Database Table: `approval_requests`

```sql
CREATE TABLE IF NOT EXISTS approval_requests (
    id              SERIAL PRIMARY KEY,
    agent_discord_id TEXT NOT NULL,
    guild_id        TEXT NOT NULL,
    agency          TEXT NOT NULL,
    owner_discord_id TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending, approved, denied
    denial_reason   TEXT,
    requested_at    TIMESTAMP DEFAULT NOW(),
    responded_at    TIMESTAMP,
    dm_message_id   TEXT,                              -- to edit the DM after response
    UNIQUE(agent_discord_id, guild_id)
);
```

#### File: `db/approval.go` — NEW

```go
package db

import "time"

type ApprovalRequest struct {
    ID              int
    AgentDiscordID  string
    GuildID         string
    Agency          string
    OwnerDiscordID  string
    Status          string
    DenialReason    string
    RequestedAt     time.Time
    RespondedAt     *time.Time
    DMMessageID     string
}

func (d *DB) CreateApprovalRequest(req *ApprovalRequest) error { /* INSERT */ }
func (d *DB) GetPendingApproval(agentDiscordID, guildID string) (*ApprovalRequest, error) { /* SELECT */ }
func (d *DB) ApproveAgent(id int) error { /* UPDATE status='approved', responded_at=NOW() */ }
func (d *DB) DenyAgent(id int, reason string) error { /* UPDATE status='denied' */ }
func (d *DB) GetPendingByOwner(ownerDiscordID string) ([]ApprovalRequest, error) { /* SELECT */ }
```

#### File: `config/config.go` — ADD

```go
// Agency owner Discord IDs — map agency name to owner Discord ID
// Store as env vars or in a new DB table

type AgencyOwner struct {
    Agency    string
    OwnerID   string // Discord user ID
    OwnerName string
    Phone     string // for Twilio SMS (Phase future)
}

// Add to Config:
AgencyOwners map[string]string // agency name → owner Discord ID

// Load from env:
// AGENCY_OWNER_TFC=123456789
// AGENCY_OWNER_RADIANT=987654321
// etc.
```

#### File: `bot/approval.go` — NEW

```go
package bot

import (
    "fmt"
    "github.com/bwmarrin/discordgo"
    "license-bot-go/db"
)

// triggerApprovalFlow is called during onboarding after Step 1 form.
// Instead of immediately assigning agency role, puts agent in "pending" state.
func (b *Bot) triggerApprovalFlow(s *discordgo.Session, agentDiscordID, guildID, agency, agentName string) error {
    ownerID := b.cfg.AgencyOwners[agency]
    if ownerID == "" {
        // No owner configured — auto-approve
        return nil
    }

    // 1. Create approval request in DB
    req := &db.ApprovalRequest{
        AgentDiscordID: agentDiscordID,
        GuildID:        guildID,
        Agency:         agency,
        OwnerDiscordID: ownerID,
        Status:         "pending",
    }
    if err := b.db.CreateApprovalRequest(req); err != nil {
        return err
    }

    // 2. DM the agency owner
    ch, err := s.UserChannelCreate(ownerID)
    if err != nil {
        return fmt.Errorf("cannot DM agency owner: %w", err)
    }

    embed := &discordgo.MessageEmbed{
        Title:       "🆕 New Agent Awaiting Approval",
        Description: fmt.Sprintf("**%s** wants to join **%s**.", agentName, agency),
        Color:       0xFFA500, // orange
        Fields: []*discordgo.MessageEmbedField{
            {Name: "Agent", Value: fmt.Sprintf("<@%s>", agentDiscordID), Inline: true},
            {Name: "Agency", Value: agency, Inline: true},
        },
    }

    msg, err := s.ChannelMessageSendComplex(ch.ID, &discordgo.MessageSend{
        Embeds: []*discordgo.MessageEmbed{embed},
        Components: []discordgo.MessageComponent{
            discordgo.ActionsRow{
                Components: []discordgo.MessageComponent{
                    discordgo.Button{
                        Label:    "✅ Approve",
                        Style:    discordgo.SuccessButton,
                        CustomID: fmt.Sprintf("vipa:approve:%s", agentDiscordID),
                    },
                    discordgo.Button{
                        Label:    "❌ Deny",
                        Style:    discordgo.DangerButton,
                        CustomID: fmt.Sprintf("vipa:deny:%s", agentDiscordID),
                    },
                },
            },
        },
    })
    if err != nil {
        return err
    }

    // 3. Store DM message ID so we can edit it later
    b.db.UpdateApprovalDMMessageID(req.ID, msg.ID)

    // 4. Assign "Pending" role to agent (optional)
    // s.GuildMemberRoleAdd(guildID, agentDiscordID, b.cfg.PendingRoleID)

    return nil
}

// handleApprovalResponse handles Approve/Deny button clicks.
func (b *Bot) handleApprovalResponse(s *discordgo.Session, i *discordgo.InteractionCreate) {
    customID := i.MessageComponentData().CustomID
    // Parse: "vipa:approve:{discord_id}" or "vipa:deny:{discord_id}"

    parts := strings.Split(customID, ":")
    action := parts[1]  // "approve" or "deny"
    agentID := parts[2] // discord ID

    switch action {
    case "approve":
        b.handleApprove(s, i, agentID)
    case "deny":
        // Show modal asking for optional reason
        b.showDenialReasonModal(s, i, agentID)
    }
}

// handleApprove completes the approval.
func (b *Bot) handleApprove(s *discordgo.Session, i *discordgo.InteractionCreate, agentDiscordID string) {
    // 1. Update DB: status = approved
    // 2. Assign agency role to agent
    // 3. DM agent: "You've been approved to join {agency}!"
    // 4. Edit owner's DM: "✅ Approved — {agent name}"
    // 5. Remove Pending role if assigned
    // 6. Continue onboarding flow
}

// handleDenialSubmit processes denial with optional reason.
func (b *Bot) handleDenialSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // 1. Extract reason from modal
    // 2. Update DB: status = denied, denial_reason
    // 3. DM agent: "Your request to join {agency} was not approved."
    //    If reason provided: "Reason: {reason}"
    // 4. Edit owner's DM: "❌ Denied — {agent name}"
    // 5. Log to admin channel
}
```

#### File: `bot/bot.go` — MODIFY

Add approval button handlers to `handleComponent()`:

```go
case strings.HasPrefix(customID, "vipa:approve:") || strings.HasPrefix(customID, "vipa:deny:"):
    b.handleApprovalResponse(s, i)
```

Add denial modal handler to `handleModalSubmit()`:

```go
case strings.HasPrefix(customID, "vipa:modal_denial:"):
    b.handleDenialSubmit(s, i)
```

#### File: `bot/intake.go` — MODIFY

In the form submission handler (after Step 1 or Step 2), before assigning the agency role:

```go
// After normalizing agency and before assigning role:
if b.cfg.AgencyOwners[agency] != "" {
    // Trigger approval flow instead of immediate assignment
    if err := b.triggerApprovalFlow(s, userID, guildID, agency, fullName); err != nil {
        log.Printf("Approval flow error: %v", err)
    }
    // Don't assign agency role yet — wait for approval
    return
}
// If no owner configured, assign role immediately (existing behavior)
```

### 3B. Direct Manager Assignment

#### Database Change: `onboarding_agents` — ADD COLUMN

```sql
ALTER TABLE onboarding_agents ADD COLUMN direct_manager_discord_id TEXT;
ALTER TABLE onboarding_agents ADD COLUMN direct_manager_name TEXT;
```

#### Slash Command: `/agent assign-manager`

```go
// Add subcommand to /agent:
{
    Type:        discordgo.ApplicationCommandOptionSubCommand,
    Name:        "assign-manager",
    Description: "Assign a direct manager to an agent",
    Options: []*discordgo.ApplicationCommandOption{
        {
            Type:        discordgo.ApplicationCommandOptionUser,
            Name:        "agent",
            Description: "The agent to assign",
            Required:    true,
        },
        {
            Type:        discordgo.ApplicationCommandOptionUser,
            Name:        "manager",
            Description: "The manager to assign",
            Required:    true,
        },
    },
}
```

```go
func (b *Bot) handleAssignManager(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // 1. Check if caller is staff
    // 2. Get agent and manager from options
    // 3. Update onboarding_agents SET direct_manager_discord_id, direct_manager_name
    // 4. Log activity
    // 5. Respond with confirmation embed
}
```

### 3C. Twilio SMS (Future — Placeholder)

#### File: `sms/twilio.go` — NEW

```go
package sms

import (
    "fmt"
    "net/http"
    "net/url"
    "strings"
)

type TwilioClient struct {
    accountSID string
    authToken  string
    fromNumber string
}

func NewTwilioClient(sid, token, from string) *TwilioClient {
    if sid == "" || token == "" || from == "" {
        return nil
    }
    return &TwilioClient{accountSID: sid, authToken: token, fromNumber: from}
}

func (t *TwilioClient) SendSMS(to, body string) error {
    apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)

    data := url.Values{}
    data.Set("To", to)
    data.Set("From", t.fromNumber)
    data.Set("Body", body)

    req, _ := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
    req.SetBasicAuth(t.accountSID, t.authToken)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 400 {
        return fmt.Errorf("twilio: HTTP %d", resp.StatusCode)
    }
    return nil
}
```

#### Config additions:

```go
TwilioAccountSID string // TWILIO_ACCOUNT_SID
TwilioAuthToken  string // TWILIO_AUTH_TOKEN
TwilioFromNumber string // TWILIO_FROM_NUMBER
```

---

## PHASE 4: GoHighLevel CRM Integration
<a name="phase-4"></a>

### 4A. GHL API Client

**Research with Perplexity MCP FIRST:**
```
"GoHighLevel API v2 documentation" contacts create update pipeline
"GoHighLevel CRM API authentication" bearer token
```

#### File: `ghl/client.go` — NEW

```go
package ghl

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

const baseURL = "https://services.leadconnectorhq.com"

type Client struct {
    apiKey     string
    locationID string
    httpClient *http.Client
}

func NewClient(apiKey, locationID string) *Client {
    if apiKey == "" {
        return nil
    }
    return &Client{
        apiKey:     apiKey,
        locationID: locationID,
        httpClient: &http.Client{Timeout: 30 * time.Second},
    }
}

func (c *Client) do(method, path string, body interface{}) ([]byte, error) {
    var reqBody io.Reader
    if body != nil {
        data, err := json.Marshal(body)
        if err != nil {
            return nil, err
        }
        reqBody = bytes.NewReader(data)
    }

    req, err := http.NewRequest(method, baseURL+path, reqBody)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Authorization", "Bearer "+c.apiKey)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Version", "2021-07-28")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respBody, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("GHL API %d: %s", resp.StatusCode, string(respBody))
    }
    return respBody, nil
}
```

#### File: `ghl/contacts.go` — NEW

```go
package ghl

import "encoding/json"

type Contact struct {
    ID        string `json:"id,omitempty"`
    FirstName string `json:"firstName"`
    LastName  string `json:"lastName"`
    Email     string `json:"email,omitempty"`
    Phone     string `json:"phone,omitempty"`
    Tags      []string `json:"tags,omitempty"`
    CustomFields []CustomField `json:"customField,omitempty"`
}

type CustomField struct {
    ID    string `json:"id"`
    Value string `json:"value"`
}

// CreateOrUpdateContact upserts a contact in GHL.
func (c *Client) CreateOrUpdateContact(contact *Contact) (*Contact, error) {
    data, err := c.do("POST", "/contacts/upsert", map[string]interface{}{
        "firstName":   contact.FirstName,
        "lastName":    contact.LastName,
        "email":       contact.Email,
        "phone":       contact.Phone,
        "tags":        contact.Tags,
        "locationId":  c.locationID,
        "customField": contact.CustomFields,
    })
    if err != nil {
        return nil, err
    }
    var resp struct {
        Contact Contact `json:"contact"`
    }
    json.Unmarshal(data, &resp)
    return &resp.Contact, nil
}

// UpdateContactStage updates the pipeline stage for a contact.
func (c *Client) UpdateContactStage(contactID, pipelineID, stageID string) error {
    _, err := c.do("PUT", "/opportunities/upsert", map[string]interface{}{
        "pipelineId":    pipelineID,
        "stageId":       stageID,
        "contactId":     contactID,
        "locationId":    c.locationID,
        "status":        "open",
    })
    return err
}
```

#### File: `ghl/pipeline.go` — NEW

```go
package ghl

// Maps bot stages to GHL pipeline stages.
// These IDs come from GHL pipeline configuration.

type PipelineMapping struct {
    PipelineID string
    Stages     map[int]string // bot stage → GHL stage ID
}

// GetPipelineMapping returns the stage mapping.
// GHL pipeline/stage IDs must be configured via env vars.
func GetPipelineMapping(cfg map[string]string) *PipelineMapping {
    return &PipelineMapping{
        PipelineID: cfg["GHL_PIPELINE_ID"],
        Stages: map[int]string{
            1: cfg["GHL_STAGE_WELCOME"],
            2: cfg["GHL_STAGE_FORM"],
            3: cfg["GHL_STAGE_SORTED"],
            4: cfg["GHL_STAGE_STUDENT"],
            5: cfg["GHL_STAGE_LICENSED"],
            6: cfg["GHL_STAGE_CONTRACTING"],
            7: cfg["GHL_STAGE_SETUP"],
            8: cfg["GHL_STAGE_ACTIVE"],
        },
    }
}
```

### 4B. GHL Sync Integration Points

Add GHL sync calls to existing handlers:

| Event | File | Function | GHL Action |
|-------|------|----------|------------|
| Agent onboarded | `bot/intake.go` | After Step 2 submit | `CreateOrUpdateContact()` |
| License verified | `bot/verify.go` | After successful verify | `UpdateContactStage(5)` + add "Licensed" tag |
| Stage change | `db/db.go` | `UpdateAgentStage()` | `UpdateContactStage(newStage)` |
| Agent activated | `bot/activation.go` | `handleActivate()` | `UpdateContactStage(8)` + add "Active" tag |
| Agent kicked | `bot/admin.go` | `handleKick()` | Mark opportunity as "lost" |

#### File: `bot/bot.go` — ADD GHL client

```go
type Bot struct {
    cfg        *config.Config
    db         *db.DB
    session    *discordgo.Session
    registry   *scrapers.Registry
    mailer     *email.Client
    ghl        *ghl.Client       // NEW
    sms        *sms.TwilioClient  // NEW
    modalState sync.Map
}
```

Initialize in `New()`:

```go
ghlClient := ghl.NewClient(cfg.GHLAPIKey, cfg.GHLLocationID)
if ghlClient != nil {
    log.Println("GoHighLevel CRM client configured")
}

smsClient := sms.NewTwilioClient(cfg.TwilioAccountSID, cfg.TwilioAuthToken, cfg.TwilioFromNumber)
if smsClient != nil {
    log.Println("Twilio SMS client configured")
}
```

### 4C. GHL Webhook Handler

#### File: `ghl/webhooks.go` — NEW

```go
package ghl

// Handle incoming webhooks from GHL (e.g., contact updated in CRM)
// Register webhook at: POST /api/v1/webhooks/ghl

type WebhookPayload struct {
    Type      string `json:"type"`
    ContactID string `json:"contact_id"`
    // ... more fields based on GHL webhook format
}
```

#### File: `api/server.go` — ADD ROUTE

```go
mux.HandleFunc("POST /api/v1/webhooks/ghl", s.handleGHLWebhook)
```

---

## PHASE 5: Leaderboard + Activity Tracking
<a name="phase-5"></a>

### 5A. Activity Logging Commands

**Goal:** Agents and managers can log activities (calls made, appointments set, policies written). Like the JW Sales Bot reference.

#### New Database Table: `activity_entries`

```sql
CREATE TABLE IF NOT EXISTS activity_entries (
    id              SERIAL PRIMARY KEY,
    discord_id      TEXT NOT NULL,
    guild_id        TEXT NOT NULL,
    activity_type   TEXT NOT NULL,    -- 'calls', 'appointments', 'presentations', 'policies', 'recruits'
    count           INT NOT NULL DEFAULT 1,
    notes           TEXT,
    logged_at       TIMESTAMP DEFAULT NOW(),
    week_start      DATE NOT NULL     -- Monday of the week (for weekly aggregation)
);

CREATE INDEX idx_activity_week ON activity_entries(discord_id, week_start);
CREATE INDEX idx_activity_type ON activity_entries(activity_type, week_start);
```

#### File: `db/activity.go` — NEW (rename from existing db/agents.go LogActivity)

```go
package db

import "time"

type ActivityEntry struct {
    ID           int
    DiscordID    string
    ActivityType string
    Count        int
    Notes        string
    LoggedAt     time.Time
    WeekStart    time.Time
}

type LeaderboardEntry struct {
    DiscordID    string
    DisplayName  string
    TotalCount   int
    Rank         int
}

// LogActivityEntry records a new activity.
func (d *DB) LogActivityEntry(entry *ActivityEntry) error {
    // INSERT INTO activity_entries (discord_id, guild_id, activity_type, count, notes, week_start)
    // VALUES ($1, $2, $3, $4, $5, date_trunc('week', NOW()))
}

// GetWeeklyLeaderboard returns top agents by activity type for current week.
func (d *DB) GetWeeklyLeaderboard(activityType string, limit int) ([]LeaderboardEntry, error) {
    // SELECT discord_id, SUM(count) as total
    // FROM activity_entries
    // WHERE activity_type = $1 AND week_start = date_trunc('week', NOW())
    // GROUP BY discord_id ORDER BY total DESC LIMIT $2
}

// GetMonthlyLeaderboard returns top agents by activity type for current month.
func (d *DB) GetMonthlyLeaderboard(activityType string, limit int) ([]LeaderboardEntry, error) {
    // Similar but WHERE logged_at >= date_trunc('month', NOW())
}

// GetAgentWeeklyActivity returns an agent's activities for the current week.
func (d *DB) GetAgentWeeklyActivity(discordID string) ([]ActivityEntry, error) {
    // SELECT * FROM activity_entries
    // WHERE discord_id = $1 AND week_start = date_trunc('week', NOW())
}

// GetDailyRecap returns aggregate counts for today.
func (d *DB) GetDailyRecap() (map[string]int, error) {
    // SELECT activity_type, SUM(count)
    // FROM activity_entries
    // WHERE logged_at::date = CURRENT_DATE
    // GROUP BY activity_type
}
```

#### File: `bot/activity_log.go` — NEW

```go
package bot

// /log command — agents log their daily activities

// Slash command: /log
// Options:
//   calls: int (number of calls made)
//   appointments: int (appointments set)
//   presentations: int (presentations given)
//   policies: int (policies written)
//   recruits: int (people recruited)
//   notes: string (optional notes)

func (b *Bot) handleLogCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // 1. Parse all option values
    // 2. For each non-zero value, create ActivityEntry
    // 3. Save to DB
    // 4. Respond with summary embed:
    //    "📊 Activity Logged!
    //     Calls: 25 | Appointments: 3 | Policies: 1
    //     Weekly Total: Calls: 150 | Appointments: 12 | Policies: 5"
    // 5. If GHL configured, sync activity to GHL contact
}
```

#### Slash Command Registration — `/log`

```go
{
    Name:        "log",
    Description: "Log your daily activity",
    Options: []*discordgo.ApplicationCommandOption{
        {Type: discordgo.ApplicationCommandOptionInteger, Name: "calls", Description: "Number of calls made"},
        {Type: discordgo.ApplicationCommandOptionInteger, Name: "appointments", Description: "Appointments set"},
        {Type: discordgo.ApplicationCommandOptionInteger, Name: "presentations", Description: "Presentations given"},
        {Type: discordgo.ApplicationCommandOptionInteger, Name: "policies", Description: "Policies written"},
        {Type: discordgo.ApplicationCommandOptionInteger, Name: "recruits", Description: "People recruited"},
        {Type: discordgo.ApplicationCommandOptionString, Name: "notes", Description: "Optional notes"},
    },
}
```

### 5B. Leaderboard Commands

#### File: `bot/leaderboard.go` — NEW

```go
package bot

// /leaderboard command — shows top performers

func (b *Bot) handleLeaderboard(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Subcommands: weekly, monthly, alltime
    // Options: type (calls, appointments, policies, recruits, all)
}

func (b *Bot) buildLeaderboardEmbed(entries []db.LeaderboardEntry, title, period string) *discordgo.MessageEmbed {
    // 🏆 Weekly Leaderboard — Calls
    //
    // 🥇 @Agent1 — 250 calls
    // 🥈 @Agent2 — 200 calls
    // 🥉 @Agent3 — 175 calls
    // 4. @Agent4 — 150 calls
    // 5. @Agent5 — 120 calls
    // ...
}

// postDailyRecap posts end-of-day activity summary.
// Called by scheduler at configured time.
func (b *Bot) postDailyRecap(s *discordgo.Session) {
    // 1. Get today's aggregate counts
    // 2. Build embed:
    //    "📊 Daily Recap — Feb 22, 2026
    //     Total Calls: 450 | Appointments: 35 | Policies: 12
    //     Top Performer: @Agent1 (250 calls)"
    // 3. Post to configured channel
}

// postWeeklyLeaderboard posts end-of-week summary.
func (b *Bot) postWeeklyLeaderboard(s *discordgo.Session) {
    // Full leaderboard across all categories
    // Post to leaderboard channel
}
```

#### Slash Command Registration — `/leaderboard`

```go
{
    Name:        "leaderboard",
    Description: "View activity leaderboard",
    Options: []*discordgo.ApplicationCommandOption{
        {
            Type:        discordgo.ApplicationCommandOptionSubCommand,
            Name:        "weekly",
            Description: "This week's leaderboard",
            Options: []*discordgo.ApplicationCommandOption{
                {Type: discordgo.ApplicationCommandOptionString, Name: "type",
                 Description: "Activity type", Required: false,
                 Choices: []*discordgo.ApplicationCommandOptionChoice{
                     {Name: "Calls", Value: "calls"},
                     {Name: "Appointments", Value: "appointments"},
                     {Name: "Policies", Value: "policies"},
                     {Name: "Recruits", Value: "recruits"},
                     {Name: "All", Value: "all"},
                 }},
            },
        },
        {
            Type:        discordgo.ApplicationCommandOptionSubCommand,
            Name:        "monthly",
            Description: "This month's leaderboard",
        },
    },
}
```

#### API Endpoints

```go
// File: api/leaderboard.go — NEW

// GET /api/v1/leaderboard/weekly?type=calls    → []LeaderboardEntry
// GET /api/v1/leaderboard/monthly?type=calls   → []LeaderboardEntry
// GET /api/v1/leaderboard/daily-recap          → DailyRecapResponse
```

#### Scheduler Integration

```go
// In bot/scheduler.go, add:

// Daily at 9 PM ET:
b.postDailyRecap(s)

// Weekly on Sunday at 9 PM ET:
b.postWeeklyLeaderboard(s)
```

---

## PHASE 6: Mobile UX + Zoom Verticals + Role Cleanup
<a name="phase-6"></a>

### 6A. Mobile UX Improvements

Discord embeds are already mobile-friendly. Focus on:

1. **Shorter embed descriptions** — trim long text for mobile readability
2. **Button-first interactions** — prefer buttons over typing commands
3. **Compact leaderboard** — mobile-friendly formatting with fewer columns
4. **Quick-log buttons** — persistent message with +1 buttons for common activities

#### File: `bot/embeds.go` — MODIFY

```go
// Add mobile-friendly embed variants:
// - Shorter field values (truncate to 100 chars)
// - Fewer inline fields (max 2 per row)
// - Use emojis for visual hierarchy instead of long text
```

#### Quick-Log Persistent Message

```go
// Post a persistent message in #activity channel with buttons:
// [📞 +1 Call] [📅 +1 Appt] [📋 +1 Presentation] [📝 +1 Policy]
// Each button logs +1 for that activity type for the clicking user.

func (b *Bot) postQuickLogMessage(s *discordgo.Session, channelID string) {
    s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
        Embeds: []*discordgo.MessageEmbed{{
            Title:       "⚡ Quick Activity Log",
            Description: "Tap a button to log +1 for that activity.",
            Color:       0x00D166,
        }},
        Components: []discordgo.MessageComponent{
            discordgo.ActionsRow{Components: []discordgo.MessageComponent{
                discordgo.Button{Label: "📞 Call", CustomID: "vipa:quicklog:calls", Style: discordgo.SecondaryButton},
                discordgo.Button{Label: "📅 Appt", CustomID: "vipa:quicklog:appointments", Style: discordgo.SecondaryButton},
                discordgo.Button{Label: "📋 Pres", CustomID: "vipa:quicklog:presentations", Style: discordgo.SecondaryButton},
                discordgo.Button{Label: "📝 Policy", CustomID: "vipa:quicklog:policies", Style: discordgo.SecondaryButton},
            }},
        },
    })
}
```

### 6B. Zoom Verticals

**Goal:** Break Zoom rooms into verticals (Union Leads, Aged MP, different lead types).

#### New Database Table: `zoom_verticals`

```sql
CREATE TABLE IF NOT EXISTS zoom_verticals (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,           -- 'Union Leads', 'Aged MP', 'Fresh Leads'
    description TEXT,
    zoom_link   TEXT,
    schedule    TEXT,                     -- 'Mon/Wed/Fri 10am ET'
    lead_type   TEXT,                    -- category tag
    active      BOOLEAN DEFAULT true,
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS zoom_assignments (
    id           SERIAL PRIMARY KEY,
    discord_id   TEXT NOT NULL,
    vertical_id  INT REFERENCES zoom_verticals(id),
    role         TEXT DEFAULT 'member',   -- 'lead', 'member'
    assigned_at  TIMESTAMP DEFAULT NOW()
);
```

#### File: `bot/zoom.go` — NEW

```go
package bot

// /zoom command — manage zoom verticals

// Subcommands:
//   /zoom list          — show all verticals with links
//   /zoom join {name}   — join a vertical
//   /zoom leave {name}  — leave a vertical
//   /zoom schedule      — show this week's zoom schedule
//   /zoom create {name} {link} {schedule}  — admin: create vertical
//   /zoom delete {name} — admin: remove vertical

func (b *Bot) handleZoomCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Route subcommands
}

func (b *Bot) handleZoomList(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // Query active verticals
    // Build embed:
    //   📹 Zoom Verticals
    //   🔵 Union Leads — Mon/Wed/Fri 10am ET [Join Link]
    //   🟢 Aged MP — Tue/Thu 2pm ET [Join Link]
    //   🟡 Fresh Leads — Daily 9am ET [Join Link]
}
```

### 6C. Role Cleanup

#### File: `bot/admin.go` — ADD

```go
// /role-audit command — find users with conflicting or missing roles

func (b *Bot) handleRoleAudit(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // 1. Get all guild members
    // 2. For each member:
    //    a. Check if they have both Student AND Licensed roles (conflict)
    //    b. Check if they're in DB but missing expected role for their stage
    //    c. Check if they have an agency role but aren't in the DB
    // 3. Build report embed
    // 4. Offer "Fix All" button to auto-correct
}

// /role-sync command — sync all roles to match DB state
func (b *Bot) handleRoleSync(s *discordgo.Session, i *discordgo.InteractionCreate) {
    // For each agent in DB:
    //   Set roles based on current stage
    //   Remove conflicting roles
}
```

---

## DATABASE SCHEMA (Complete)
<a name="database-schema"></a>

### All Tables After All Phases

```sql
-- ═══════════════════════════════════════════════════════════════
-- EXISTING TABLES (already created in db/db.go migrations)
-- ═══════════════════════════════════════════════════════════════

-- 1. onboarding_agents (main agent table)
-- 2. license_checks (verification history)
-- 3. verification_deadlines (30-day deadline tracking)
-- 4. agent_activity_log (generic activity events)
-- 5. agent_weekly_checkins (weekly check-in records)
-- 6. contracting_managers (manager info)
-- 7. agent_setup_progress (setup checklist)

-- ═══════════════════════════════════════════════════════════════
-- NEW TABLES
-- ═══════════════════════════════════════════════════════════════

-- 8. Phase 2: No new tables (uses existing onboarding_agents + new columns)

-- 9. Phase 3: Approval system
CREATE TABLE IF NOT EXISTS approval_requests (
    id              SERIAL PRIMARY KEY,
    agent_discord_id TEXT NOT NULL,
    guild_id        TEXT NOT NULL,
    agency          TEXT NOT NULL,
    owner_discord_id TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    denial_reason   TEXT,
    requested_at    TIMESTAMP DEFAULT NOW(),
    responded_at    TIMESTAMP,
    dm_message_id   TEXT,
    UNIQUE(agent_discord_id, guild_id)
);

-- 10. Phase 5: Activity tracking
CREATE TABLE IF NOT EXISTS activity_entries (
    id              SERIAL PRIMARY KEY,
    discord_id      TEXT NOT NULL,
    guild_id        TEXT NOT NULL,
    activity_type   TEXT NOT NULL,
    count           INT NOT NULL DEFAULT 1,
    notes           TEXT,
    logged_at       TIMESTAMP DEFAULT NOW(),
    week_start      DATE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_activity_week ON activity_entries(discord_id, week_start);
CREATE INDEX IF NOT EXISTS idx_activity_type ON activity_entries(activity_type, week_start);

-- 11. Phase 6: Zoom verticals
CREATE TABLE IF NOT EXISTS zoom_verticals (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    zoom_link   TEXT,
    schedule    TEXT,
    lead_type   TEXT,
    active      BOOLEAN DEFAULT true,
    created_at  TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS zoom_assignments (
    id           SERIAL PRIMARY KEY,
    discord_id   TEXT NOT NULL,
    vertical_id  INT REFERENCES zoom_verticals(id),
    role         TEXT DEFAULT 'member',
    assigned_at  TIMESTAMP DEFAULT NOW(),
    UNIQUE(discord_id, vertical_id)
);

-- ═══════════════════════════════════════════════════════════════
-- COLUMN ADDITIONS TO EXISTING TABLES
-- ═══════════════════════════════════════════════════════════════

-- Phase 2: Add to onboarding_agents
ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS last_nudge_sent_at TIMESTAMP;
ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS upline_manager_discord_id TEXT;

-- Phase 3: Add to onboarding_agents
ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS direct_manager_discord_id TEXT;
ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS direct_manager_name TEXT;
ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS approval_status TEXT DEFAULT 'none';

-- Phase 4: Add to onboarding_agents
ALTER TABLE onboarding_agents ADD COLUMN IF NOT EXISTS ghl_contact_id TEXT;
```

---

## API ENDPOINTS (Complete)
<a name="api-endpoints"></a>

### All Endpoints After All Phases

| Method | Path | Auth | Phase | Purpose |
|--------|------|------|-------|---------|
| GET | `/api/v1/health` | No | Existing | Health check |
| GET | `/api/v1/agents` | Yes | Existing | List agents |
| GET | `/api/v1/agents/{id}` | Yes | Existing | Agent detail |
| GET | `/api/v1/dashboard/funnel` | Yes | Existing | Stage counts |
| GET | `/api/v1/dashboard/summary` | Yes | Existing | Summary metrics |
| GET | `/api/v1/dashboard/at-risk` | Yes | Existing | At-risk agents |
| GET | `/api/v1/tracker/overview` | Yes | Phase 2 | Overall licensed/total |
| GET | `/api/v1/tracker/agencies` | Yes | Phase 2 | Per-agency stats |
| GET | `/api/v1/tracker/recruiters` | Yes | Phase 2 | Per-recruiter stats |
| GET | `/api/v1/tracker/unlicensed` | Yes | Phase 2 | Unlicensed agent list |
| GET | `/api/v1/approvals/pending` | Yes | Phase 3 | Pending approvals |
| POST | `/api/v1/approvals/{id}/approve` | Yes | Phase 3 | Approve agent |
| POST | `/api/v1/approvals/{id}/deny` | Yes | Phase 3 | Deny agent |
| POST | `/api/v1/webhooks/ghl` | Token | Phase 4 | GHL webhook receiver |
| GET | `/api/v1/leaderboard/weekly` | Yes | Phase 5 | Weekly leaderboard |
| GET | `/api/v1/leaderboard/monthly` | Yes | Phase 5 | Monthly leaderboard |
| GET | `/api/v1/leaderboard/daily-recap` | Yes | Phase 5 | Today's recap |
| GET | `/api/v1/zoom/verticals` | Yes | Phase 6 | List verticals |

---

## SLASH COMMANDS (Complete)
<a name="slash-commands"></a>

### All Commands After All Phases

| Command | Phase | Handler | Purpose |
|---------|-------|---------|---------|
| `/verify` | Existing | `handleVerify` | License verification |
| `/npn` | Existing | `handleNPNLookup` | NPN lookup |
| `/license-history` | Existing | `handleHistory` | Check history |
| `/contract` | Existing | `handleContract` | Book contracting |
| `/email-optin` | Existing | `handleEmailOptIn` | Opt in to email |
| `/email-optout` | Existing | `handleEmailOptOut` | Opt out of email |
| `/setup` | Existing | `handleSetup` | Setup checklist |
| `/agent` | Existing | `handleAgentCommand` | Admin agent mgmt |
| `/contracting` | Existing | `handleContractingCommand` | Manage managers |
| `/restart` | Existing | `handleRestart` | Restart onboarding |
| `/onboarding-setup` | Existing | `handleOnboardingSetup` | Post onboarding msg |
| `/setup-rules` | Existing | `handleSetupRules` | Post rules embed |
| `/tracker` | Phase 2 | `handleTrackerCommand` | License progress |
| `/agent assign-manager` | Phase 3 | `handleAssignManager` | Assign manager |
| `/log` | Phase 5 | `handleLogCommand` | Log daily activity |
| `/leaderboard` | Phase 5 | `handleLeaderboard` | View leaderboard |
| `/zoom` | Phase 6 | `handleZoomCommand` | Zoom verticals |
| `/role-audit` | Phase 6 | `handleRoleAudit` | Find role issues |
| `/role-sync` | Phase 6 | `handleRoleSync` | Fix all roles |
| `/quicklog-setup` | Phase 6 | `postQuickLogMessage` | Post quick-log buttons |

---

## ENVIRONMENT VARIABLES
<a name="environment-variables"></a>

### All Variables After All Phases

```bash
# ═══════════════════════════════════════
# EXISTING (already configured)
# ═══════════════════════════════════════
DISCORD_TOKEN=
GUILD_ID=
DATABASE_URL=
CAPSOLVER_API_KEY=
RESEND_API_KEY=
EMAIL_FROM=
EMAIL_FROM_NAME=
API_TOKEN=
API_PORT=8080

# Channel IDs
HIRING_LOG_CHANNEL_ID=
LICENSE_CHECK_CHANNEL_ID=
ADMIN_NOTIFY_CHANNEL_ID=
GREETINGS_CHANNEL_ID=
START_HERE_CHANNEL_ID=
RULES_CHANNEL_ID=

# Role IDs
STUDENT_ROLE_ID=
LICENSED_AGENT_ROLE_ID=
ACTIVE_AGENT_ROLE_ID=
STAFF_ROLE_IDS=
TFC_ROLE_ID=
RADIANT_ROLE_ID=
GBU_ROLE_ID=
TRULIGHT_ROLE_ID=
THRIVE_ROLE_ID=
THE_POINT_ROLE_ID=
SYNERGY_ROLE_ID=
ILLUMINATE_ROLE_ID=
ELITE_ONE_ROLE_ID=

# Scheduler
CHECKIN_DAY=Monday
DEADLINE_DAYS=30

# ═══════════════════════════════════════
# NEW — Phase 1
# ═══════════════════════════════════════
LICENSE_VERIFY_LOG_CHANNEL_ID=1474946201450184726
NIPR_API_KEY=                           # Optional: NIPR PDB API key

# ═══════════════════════════════════════
# NEW — Phase 2
# ═══════════════════════════════════════
TRACKER_CHANNEL_ID=                     # Channel for auto-posting tracker
NUDGE_AFTER_DAYS=30                     # Days before sending recruiter nudge

# ═══════════════════════════════════════
# NEW — Phase 3
# ═══════════════════════════════════════
PENDING_ROLE_ID=                        # Role for pending-approval agents
AGENCY_OWNER_TFC=                       # Discord ID of TFC owner
AGENCY_OWNER_RADIANT=                   # Discord ID of Radiant owner
AGENCY_OWNER_GBU=                       # Discord ID of GBU owner
AGENCY_OWNER_TRULIGHT=                  # Discord ID of TruLight owner
AGENCY_OWNER_THRIVE=                    # Discord ID of Thrive owner
AGENCY_OWNER_THE_POINT=                 # Discord ID of The Point owner
AGENCY_OWNER_SYNERGY=                   # Discord ID of Synergy owner
AGENCY_OWNER_ILLUMINATE=                # Discord ID of Illuminate owner
AGENCY_OWNER_ELITE_ONE=                 # Discord ID of Elite One owner

# ═══════════════════════════════════════
# NEW — Phase 4
# ═══════════════════════════════════════
GHL_API_KEY=                            # GoHighLevel API key
GHL_LOCATION_ID=                        # GHL sub-account location ID
GHL_PIPELINE_ID=                        # GHL pipeline for onboarding
GHL_STAGE_WELCOME=                      # GHL stage IDs (1-8)
GHL_STAGE_FORM=
GHL_STAGE_SORTED=
GHL_STAGE_STUDENT=
GHL_STAGE_LICENSED=
GHL_STAGE_CONTRACTING=
GHL_STAGE_SETUP=
GHL_STAGE_ACTIVE=
GHL_WEBHOOK_SECRET=                     # For validating inbound GHL webhooks

# ═══════════════════════════════════════
# NEW — Phase 5
# ═══════════════════════════════════════
LEADERBOARD_CHANNEL_ID=                 # Channel for leaderboard posts
ACTIVITY_CHANNEL_ID=                    # Channel for quick-log buttons
DAILY_RECAP_HOUR=21                     # Hour (ET) for daily recap

# ═══════════════════════════════════════
# NEW — Phase 6 (Twilio + Zoom)
# ═══════════════════════════════════════
TWILIO_ACCOUNT_SID=
TWILIO_AUTH_TOKEN=
TWILIO_FROM_NUMBER=
```

---

## IMPLEMENTATION ORDER
<a name="implementation-order"></a>

### Step-by-Step Build Sequence

```
PHASE 1A: License Verify Log Channel (1-2 hours)
  1. Add LICENSE_VERIFY_LOG_CHANNEL_ID to config/config.go
  2. Create postVerifyToLogChannel() in bot/verify.go
  3. Create buildVerifyLogEmbed() in bot/embeds.go
  4. Update handleVerify() to use new function
  5. Test: /verify → check channel 1474946201450184726
  6. Commit: "feat: separate license verify log channel"

PHASE 1B: SIRCON Scraper (4-6 hours)
  7. Research SIRCON form interface with Perplexity MCP
  8. Create scrapers/sircon.go with full implementation
  9. Test each of 8 SIRCON states with known agent names
  10. Update scrapers/registry.go — add SIRCON states
  11. Remove those 8 states from ManualLookupURLs
  12. Commit: "feat: SIRCON scraper for CO, GA, IN, KY, LA, MI, NV, OH"

PHASE 1C: State DOI Scrapers (8-12 hours)
  13. Research each state DOI website with Perplexity MCP
  14. Create scrapers/{state}.go for each of 9 states
  15. Test each scraper individually
  16. Update registry.go — add all state scrapers
  17. Commit per state or batch: "feat: {state} DOI scraper"

PHASE 1D: Territories + NIPR (2-3 hours)
  18. Create scrapers/territories.go
  19. Research NIPR PDB API with Perplexity
  20. Create scrapers/nipr.go (if API accessible)
  21. Update registry.go — add territories
  22. Remove ManualLookupURLs map and ManualScraper
  23. Commit: "feat: territory support + NIPR fallback"

PHASE 2A: License Tracker (4-6 hours)
  24. Create db/tracker.go with all query methods
  25. Create bot/tracker.go with command handlers
  26. Register /tracker slash command in bot/commands.go
  27. Add tracker routes in api/tracker.go
  28. Add scheduler job for daily tracker post
  29. Test: /tracker overview, /tracker agency, /tracker recruiter
  30. Commit: "feat: license progress tracker"

PHASE 2B: Recruiter Nudge (3-4 hours)
  31. Add columns to onboarding_agents (migration in db/db.go)
  32. Add resolveUplineDiscordID() to bot/intake.go
  33. Add sendRecruiterNudges() to bot/scheduler.go
  34. Test: manually set agent joined_at to 31 days ago → verify nudge
  35. Commit: "feat: 30-day recruiter nudge system"

PHASE 3A: Agency Owner Approval (6-8 hours)
  36. Create approval_requests table (migration)
  37. Create db/approval.go
  38. Create bot/approval.go
  39. Add agency owner env vars to config/config.go
  40. Modify bot/intake.go to trigger approval flow
  41. Add button/modal handlers to bot/bot.go
  42. Create api/approval.go for API endpoints
  43. Test: full approval + denial flow
  44. Commit: "feat: agency owner approval system"

PHASE 3B: Manager Assignment (2 hours)
  45. Add columns to onboarding_agents
  46. Add /agent assign-manager subcommand
  47. Create handleAssignManager()
  48. Test: assign and verify
  49. Commit: "feat: direct manager assignment"

PHASE 4: GHL Integration (8-10 hours)
  50. Research GHL API v2 with Perplexity MCP
  51. Create ghl/client.go
  52. Create ghl/contacts.go
  53. Create ghl/pipeline.go
  54. Create ghl/webhooks.go
  55. Add GHL env vars to config
  56. Add GHL client to Bot struct
  57. Add sync calls to intake, verify, activation handlers
  58. Add webhook route to API
  59. Test: verify contact appears in GHL after onboarding
  60. Commit: "feat: GoHighLevel CRM integration"

PHASE 5A: Activity Logging (4-6 hours)
  61. Create activity_entries table (migration)
  62. Create db/activity.go
  63. Create bot/activity_log.go with /log command
  64. Register command
  65. Test: /log calls:25 appointments:3
  66. Commit: "feat: activity logging"

PHASE 5B: Leaderboard (4-6 hours)
  67. Create bot/leaderboard.go
  68. Create api/leaderboard.go
  69. Register /leaderboard command
  70. Add scheduler jobs for daily recap + weekly leaderboard
  71. Test: /leaderboard weekly type:calls
  72. Commit: "feat: leaderboard + daily recap"

PHASE 6A: Quick-Log Buttons (2 hours)
  73. Add quick-log button handlers to bot/bot.go
  74. Create postQuickLogMessage() in bot/activity_log.go
  75. Test: tap button → activity logged
  76. Commit: "feat: mobile quick-log buttons"

PHASE 6B: Zoom Verticals (4-6 hours)
  77. Create zoom tables (migration)
  78. Create bot/zoom.go with all subcommands
  79. Register /zoom command
  80. Test: /zoom list, /zoom join, /zoom create
  81. Commit: "feat: zoom vertical management"

PHASE 6C: Role Cleanup (3-4 hours)
  82. Create /role-audit and /role-sync commands
  83. Implement audit logic
  84. Test: run audit → fix → verify
  85. Commit: "feat: role audit + sync tools"

PHASE 6D: Twilio SMS (2-3 hours)
  86. Create sms/twilio.go
  87. Add Twilio config vars
  88. Integrate into approval flow (send SMS to owner alongside DM)
  89. Test: verify SMS received
  90. Commit: "feat: Twilio SMS for approval notifications"
```

---

## VERIFICATION CHECKPOINTS
<a name="verification-checkpoints"></a>

### After Each Phase

#### Phase 1 Verification
- [ ] `/verify state:FL name:Smith` → result posts to channel `1474946201450184726`
- [ ] `/verify state:CO name:Smith` → SIRCON scraper returns results (not manual URL)
- [ ] `/verify state:NY name:Smith` → NY DFS scraper returns results
- [ ] `/verify state:PR name:Smith` → Returns NIPR fallback message
- [ ] All 56 state codes resolve in registry (no panics, no unhandled states)
- [ ] `ManualLookupURLs` map is removed from registry.go
- [ ] `/npn npn:12345678` → still works for multi-state search

#### Phase 2 Verification
- [ ] `/tracker overview` → shows "X/Y agents licensed (Z%)"
- [ ] `/tracker agency` → shows per-agency breakdown
- [ ] `/tracker recruiter agency:TFC` → shows per-recruiter stats
- [ ] Scheduler posts daily tracker embed to configured channel
- [ ] Agents 30+ days without license → recruiter gets DM
- [ ] Recruiter nudge doesn't re-send within 7 days

#### Phase 3 Verification
- [ ] New agent joins under TFC → TFC owner gets DM with Approve/Deny
- [ ] Owner clicks Approve → agent gets agency role + DM
- [ ] Owner clicks Deny → modal for reason → agent gets DM with reason
- [ ] No owner configured for agency → auto-approve (existing behavior)
- [ ] `/agent assign-manager @agent @manager` → DB updated
- [ ] API `/api/v1/approvals/pending` returns pending list

#### Phase 4 Verification
- [ ] New agent completes onboarding → GHL contact created
- [ ] Agent license verified → GHL opportunity moves to "Licensed" stage
- [ ] Agent activated → GHL opportunity moves to "Active" stage
- [ ] Agent kicked → GHL opportunity marked "lost"
- [ ] GHL webhook received → bot processes correctly

#### Phase 5 Verification
- [ ] `/log calls:25 appointments:3` → activity saved, embed shown
- [ ] `/leaderboard weekly type:calls` → shows ranked list
- [ ] Daily recap posts at configured time
- [ ] Weekly leaderboard posts on Sunday
- [ ] API leaderboard endpoints return correct data

#### Phase 6 Verification
- [ ] Quick-log buttons → tap → +1 logged → ephemeral confirmation
- [ ] `/zoom list` → shows active verticals
- [ ] `/zoom create` → creates new vertical (admin only)
- [ ] `/role-audit` → identifies conflicts
- [ ] `/role-sync` → fixes mismatched roles
- [ ] Twilio SMS sends on approval request (if configured)

### Perplexity MCP Research Checkpoints

Use Perplexity MCP before implementing each scraper:

```
For SIRCON:
  perplexity_ask("Vertafore SIRCON insurance producer license lookup form fields URL 2025")

For each state DOI (9 states):
  perplexity_ask("{State} department of insurance producer license search URL 2025")

For NIPR:
  perplexity_ask("NIPR producer database API documentation 2025")

For GoHighLevel:
  perplexity_ask("GoHighLevel API v2 contact create pipeline opportunity endpoint documentation 2025")

For Twilio:
  perplexity_ask("Twilio SMS API Go golang send message example 2025")
```

---

## DEPENDENCIES TO ADD

```
# go.mod additions:

# Phase 4 — GHL (standard HTTP, no extra deps needed)
# Phase 6 — Twilio (standard HTTP, no extra deps needed)

# No new Go dependencies required — everything uses stdlib + existing deps
# (net/http, encoding/json, github.com/PuerkitoBio/goquery, etc.)
```

---

## SUMMARY

| Phase | New Files | Modified Files | New DB Tables | New Commands | New API Endpoints | Est. Hours |
|-------|-----------|----------------|---------------|--------------|-------------------|------------|
| 1 | 20 scrapers + territories + nipr | config, registry, verify, embeds | 0 | 0 | 0 | 16-22 |
| 2 | db/tracker, bot/tracker, api/tracker | scheduler, intake, db/db (migration) | 0 (+2 columns) | /tracker | 4 | 7-10 |
| 3 | db/approval, bot/approval, api/approval, sms/twilio | config, bot, intake, admin | 1 (+3 columns) | /agent assign-manager | 3 | 8-10 |
| 4 | ghl/client, contacts, pipeline, webhooks | config, bot, intake, verify, activation | 0 (+1 column) | 0 | 1 | 8-10 |
| 5 | db/activity, bot/activity_log, bot/leaderboard, api/leaderboard | scheduler, commands, db (migration) | 1 | /log, /leaderboard | 3 | 8-12 |
| 6 | bot/zoom | embeds, admin, activity_log, db (migration) | 2 | /zoom, /role-audit, /role-sync, /quicklog-setup | 1 | 11-15 |
| **TOTAL** | **~35 new files** | **~15 modified files** | **4 new tables** | **8 new commands** | **12 new endpoints** | **58-79 hrs** |
