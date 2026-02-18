# VIPA License Bot — Claude Code Project Guide

## What This Project Is

A Discord bot written in **Go 1.24** that verifies insurance agent licenses by scraping state DOI (Department of Insurance) databases. It runs on **Railway** with **PostgreSQL**. It's one half of a two-bot system — the other half is a Python onboarding bot at `../onboarding-bot/`.

## Architecture

```
license-bot/
├── main.go                  # Entry point: loads config, connects DB, starts bot
├── config/config.go         # Env var loading (DISCORD_TOKEN, DATABASE_URL, etc.)
├── db/db.go                 # PostgreSQL: migrations, CRUD for agents/checks/deadlines
├── bot/
│   ├── bot.go               # Bot struct, Run(), slash command registration, handler routing
│   ├── verify.go            # /verify command handler + performVerification() shared logic
│   ├── history.go           # /license-history command handler
│   ├── auto_verify.go       # GuildMemberUpdate handler — auto-verifies when roles change
│   ├── scheduler.go         # 24h background loop: retries verification, sends reminders
│   └── email_prefs.go       # /email-optin and /email-optout command handlers
├── scrapers/
│   ├── types.go             # LicenseResult struct + Scraper interface
│   ├── registry.go          # Routes state codes → correct scraper (NAIC/FL/CA/TX/Manual)
│   ├── naic.go              # NAIC SBS API scraper (covers 31 states)
│   ├── florida.go           # Florida DOI HTML scraper
│   ├── california.go        # California CDI scraper (needs CapSolver for Turnstile)
│   ├── texas.go             # Texas TDI scraper (needs CapSolver for Turnstile)
│   └── captcha/capsolver.go # CapSolver API client for Cloudflare Turnstile
├── email/sendgrid.go        # SendGrid v3 API client for transactional emails
├── sms/twilio.go            # DEPRECATED — being replaced by email/sendgrid.go
├── tlsclient/client.go      # bogdanfinn/tls-client wrapper (anti-bot TLS fingerprinting)
├── Dockerfile               # Multi-stage build: golang:1.24-alpine → alpine:3.19
├── railway.toml             # Railway deployment config
└── .gitignore               # Uses /bot (not bot) to only ignore the compiled binary
```

## How The Two Bots Work Together

1. **Onboarding Bot (Python)** — handles the 8-stage welcome flow, collects user info (name, state, phone, email), assigns @Licensed-Agent or @Student roles
2. **License Bot (Go, this repo)** — watches for role changes via `GuildMemberUpdate`, auto-verifies licenses, manages 30-day deadlines, sends reminders via Discord DM + email

The Go bot detects when the Python bot assigns roles and triggers verification automatically.

## Database Tables (PostgreSQL)

- `onboarding_agents` — agent profiles (discord_id, name, email, email_opt_in, state, license_verified, etc.)
- `license_checks` — history of every verification attempt
- `verification_deadlines` — 30-day deadline tracking for unverified agents

## Key Env Vars

```
DISCORD_TOKEN=            # Required
GUILD_ID=                 # Required
DATABASE_URL=             # Required (PostgreSQL connection string)
CAPSOLVER_API_KEY=        # Optional (needed for CA/TX scrapers)
LICENSE_CHECK_CHANNEL_ID= # Channel to post verification results
HIRING_LOG_CHANNEL_ID=    # Fallback channel for logs
STUDENT_ROLE_ID=          # Discord role ID for @Student
LICENSED_AGENT_ROLE_ID=   # Discord role ID for @Licensed-Agent
SENDGRID_API_KEY=         # Optional (enables email notifications)
EMAIL_FROM=               # SendGrid verified sender email
EMAIL_FROM_NAME=          # Display name (defaults to "VIPA Insurance")
ADMIN_NOTIFICATION_CHANNEL_ID= # Channel for admin alerts on expired deadlines
```

## Build & Run

```bash
go build ./...        # Compile
go vet ./...          # Static analysis
go build -o bot .     # Build binary
./bot                 # Run (needs env vars or .env file)
```

## Deployment

Deployed on Railway. Push to `main` branch triggers auto-deploy. The Dockerfile handles everything.

## Current State & What Needs To Be Done

### COMPLETED:
- All scrapers working (NAIC 31 states, FL, CA, TX, 17 manual states)
- /verify slash command with full license detail DM embed
- /license-history command
- Auto-verify via GuildMemberUpdate (bot/auto_verify.go)
- 30-day deadline system with verification_deadlines table
- Background scheduler (bot/scheduler.go) — 24h loop, retries verification, sends reminders at day 7/14/21
- SendGrid email client (email/sendgrid.go) — sends reminders, deadline expired, verification success emails
- Email opt-in/opt-out slash commands (bot/email_prefs.go)
- email_opt_in field added to onboarding_agents table + Agent struct + AgentUpdate
- Config updated with SendGrid + admin channel env vars
- bot.go wired up: email client init, GuildMemberUpdate handler registered, scheduler started

### NEEDS TO BE DONE:
1. **Delete sms/twilio.go** — It's the old Twilio SMS integration, fully replaced by email/sendgrid.go. The file couldn't be deleted due to filesystem permissions in the dev environment. Just `rm sms/twilio.go && rmdir sms/` and remove any remaining import references.
2. **Build & verify** — Run `go build ./...` and `go vet ./...` to make sure everything compiles clean after removing the sms package. If there are any leftover `license-bot-go/sms` imports, remove them.
3. **Commit & push** — Stage all changes and push to main:
   ```bash
   git add -A
   git commit -m "Replace Twilio SMS with SendGrid email + add email opt-in/opt-out commands"
   git push origin main
   ```
4. **Railway env vars** — After pushing, set these on Railway:
   - `SENDGRID_API_KEY` — Get from sendgrid.com (free tier = 100 emails/day)
   - `EMAIL_FROM` — A verified sender email in SendGrid
   - `EMAIL_FROM_NAME` — "VIPA Insurance" or whatever you want
   - `ADMIN_NOTIFICATION_CHANNEL_ID` — Discord channel ID for admin alerts

## Code Patterns & Conventions

- **Error handling:** Log and continue, don't crash. Every goroutine handler has `defer recover()`.
- **DB queries:** Use `COALESCE()` for nullable fields, `ON CONFLICT DO UPDATE` for upserts.
- **Discord interactions:** Always defer first (`InteractionResponseDeferredChannelMessageWithSource`), then follow up.
- **Scrapers:** All implement the `Scraper` interface in `scrapers/types.go`. Return `[]LicenseResult`.
- **Email notifications:** Only sent if `agent.EmailOptIn == true` and `agent.Email != ""`.
- **nvl() helper:** In `verify.go`, returns fallback if string is empty. Used throughout embed builders.

## Important Gotchas

- `.gitignore` has `/bot` (with leading slash) — this only ignores the compiled binary at root, NOT the `bot/` source directory. If someone changes this to just `bot`, the entire bot/ package will be git-ignored and the build will fail on Railway.
- The `scrapers/captcha/capsolver.go` requires a paid CapSolver API key for CA and TX states. Without it, those states fall back to manual lookup URLs.
- The Go bot needs `IntentsGuilds | IntentsGuildMembers` Discord gateway intents (already set in bot.go). The bot must also have the "Server Members Intent" enabled in the Discord Developer Portal.
- PostgreSQL is required (not SQLite). The `DATABASE_URL` env var should be a full postgres:// connection string.
