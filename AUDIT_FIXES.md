# AUDIT FIX INSTRUCTIONS

Fix all 19 issues found in the code audit. Do them in order (critical first, then high, medium, low). After each fix, do NOT commit — fix everything, then run `go build ./...` and `go vet ./...` at the end to confirm zero errors. Then commit once as a single commit.

---

## CRITICAL (3) — Must fix, will cause crashes or data corruption

### Fix 1: db/zoom.go — ON CONFLICT missing constraint columns

In the `JoinZoomVertical` function, the INSERT statement uses `ON CONFLICT DO NOTHING` without specifying which columns define the conflict. Postgres silently ignores it, allowing duplicate zoom assignments.

Find:
```
ON CONFLICT DO NOTHING
```

Replace with:
```
ON CONFLICT (discord_id, vertical_id) DO NOTHING
```

### Fix 2: bot/zoom.go — Nil pointer panic when command used in DM

The `handleZoomCommand` function accesses `i.Member.Roles` without checking if `i.Member` is nil. When a slash command is invoked from a DM, `i.Member` is nil and this panics the entire bot.

Add a nil guard at the very top of `handleZoomCommand`, before any other logic:
```go
if i.Member == nil {
    s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
        Type: discordgo.InteractionResponseChannelMessageWithSource,
        Data: &discordgo.InteractionResponseData{
            Content: "This command can only be used in a server.",
            Flags:   discordgo.MessageFlagsEphemeral,
        },
    })
    return
}
```

Do the exact same nil guard at the top of `handleRoleAudit`.

### Fix 3: bot/approval.go — DM message ID stored even when DM fails

In `triggerApprovalFlow`, `UpdateApprovalDMMessageID` is called even if the DM to the agency owner failed (e.g., owner has DMs disabled). This stores an empty message ID, which causes Discord API errors later when trying to edit the message on approve/deny.

Find the call to `b.db.UpdateApprovalDMMessageID` and wrap it:
```go
if err == nil && msg != nil {
    b.db.UpdateApprovalDMMessageID(req.ID, msg.ID)
}
```

Make sure the `err` being checked is the error from the `ChannelMessageSendComplex` call, not a different error variable.

---

## HIGH (4) — Should fix, causes silent failures or panics under edge conditions

### Fix 4: bot/activity_log.go — Silently discarded DB error

In the `/log` command handler, the call to `GetAgentWeeklyActivity` has its error assigned to `_`. If the database is down, weekly totals show as zero with no indication anything is wrong.

Change:
```go
weeklyActivity, _ := b.db.GetAgentWeeklyActivity(ctx, userIDInt)
```
To:
```go
weeklyActivity, err := b.db.GetAgentWeeklyActivity(ctx, userIDInt)
if err != nil {
    log.Printf("Failed to get weekly activity for %d: %v", userIDInt, err)
}
```

### Fix 5: bot/zoom.go — parseDiscordID errors silently ignored (3 places)

There are 3 places in bot/zoom.go where `strconv.ParseInt` (or a helper like `parseDiscordID`) is called and the error is assigned to `_`. Invalid Discord IDs pass through as 0.

Find every instance where a Discord ID is parsed with the error ignored. For each one, add error checking:
```go
idInt, err := strconv.ParseInt(idStr, 10, 64)
if err != nil {
    b.followUp(s, i, "Invalid ID format.")
    return
}
```

### Fix 6: bot/verify.go — results[0] accessed without length check

In `handleVerify`, after the scraper returns results, the code accesses `results[0].Error` without checking if `results` is empty. If a scraper returns an empty slice, this panics.

Find the block that accesses `results[0]` after the match-finding loops. Add a guard before it:
```go
if len(results) == 0 {
    b.followUp(s, i, "No results returned from license lookup.")
    return
}
```

### Fix 7: bot/intake.go — Approval goroutine needs panic recovery

The `triggerApprovalFlow` goroutine is launched fire-and-forget. If it panics, the agent is stuck with no role and no error message.

Find:
```go
go b.triggerApprovalFlow(...)
```

Replace with:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("PANIC in triggerApprovalFlow for %s: %v", userID, r)
            // Fallback: assign roles directly so agent isn't stuck
            b.sortAndAssignRoles(s, userID, guildID, agency, licenseStatus)
        }
    }()
    b.triggerApprovalFlow(...)
}()
```

Make sure the variable names match what's actually in scope at that call site.

---

## MEDIUM (8) — Should fix for reliability and correctness

### Fix 8: bot/activity_log.go — context.Background() instead of ctx

In the `/log` handler, `UpsertAgent` is called with `context.Background()` when there's already a `ctx` variable in scope.

Change `context.Background()` to `ctx` in that specific `UpsertAgent` call.

### Fix 9: api/tracker.go — Loop variable shadows parameter

In the recruiter stats handler, the loop variable `r` shadows the `*http.Request` parameter `r`.

Rename the loop variable from `r` to `rec` or `recruiter`.

### Fix 10: db/db.go — Replace joinStrings with strings.Join

The custom `joinStrings()` function is less efficient than the stdlib.

Replace the entire `joinStrings` function body with:
```go
func joinStrings(ss []string, sep string) string {
    return strings.Join(ss, sep)
}
```

Or better yet, delete the function entirely and replace all calls to `joinStrings(x, y)` with `strings.Join(x, y)`.

### Fix 11: scrapers/types.go — IsLifeLicensed false positives

`strings.Contains(lower, "life")` matches "nightlife", "lifespan", etc.

Replace the contains check with a more targeted approach:
```go
func (r LicenseResult) IsLifeLicensed() bool {
    if !r.Active {
        return false
    }
    lower := strings.ToLower(r.LicenseType) + " " + strings.ToLower(r.LOAs)
    // Check for known life insurance license patterns
    lifePatterns := []string{"life", "life insurance", "life & health", "life, accident", "life/health"}
    for _, p := range lifePatterns {
        if strings.Contains(lower, p) {
            return true
        }
    }
    return false
}
```

Actually — the original `strings.Contains(lower, "life")` already covers all these patterns since they all contain "life". The real risk is false positives from words containing "life" as a substring. In practice, insurance license types won't contain "nightlife" or "lifespan", so the current implementation is acceptable for this domain. Leave it as-is but add a comment explaining why:

```go
// Note: "life" substring matching is sufficient for insurance license types.
// False positives from non-insurance words (e.g., "nightlife") are not realistic
// in DOI/NAIC license type and LOA fields.
```

### Fix 12: ghl/contacts.go — JSON tag may not match API spec

The `CustomFields` field has JSON tag `"customField"` (singular). The GHL API v2 may expect `"customFields"` (plural).

Use Perplexity MCP to verify:
```
perplexity_ask("GoHighLevel API v2 contacts upsert endpoint custom fields JSON field name 2025")
```

If the API expects plural, change the JSON tag from `"customField"` to `"customFields"`.

### Fix 13: scrapers/california.go — Hardcoded Turnstile site key

The Turnstile site key is a const. If CA DOI rotates it, the scraper breaks silently.

This is a known fragility but not urgent. Add a comment:
```go
// TODO: Extract Turnstile site key from page HTML dynamically instead of hardcoding.
// If CA DOI rotates the key, this will need updating.
const caTurnstileSiteKey = "0x4AAAAAAAeV7o-X_350Kljk"
```

### Fix 14: bot/bot.go — Missing IntentsDirectMessages

Add `discordgo.IntentsDirectMessages` to the intents bitmask. Find the line that sets intents:
```go
session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers
```

Change to:
```go
session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsDirectMessages
```

### Fix 15: scrapers/captcha/capsolver.go — Ignored errors (4 places)

Find every `json.Marshal` and `io.ReadAll` call where the error is assigned to `_`. Change each one to check the error:

```go
// Instead of:
taskJSON, _ := json.Marshal(taskPayload)
// Use:
taskJSON, err := json.Marshal(taskPayload)
if err != nil {
    return "", fmt.Errorf("capsolver: failed to marshal task: %w", err)
}
```

```go
// Instead of:
body, _ := io.ReadAll(resp.Body)
// Use:
body, err := io.ReadAll(resp.Body)
if err != nil {
    return "", fmt.Errorf("capsolver: failed to read response: %w", err)
}
```

Do this for ALL 4 instances in the file.

### Fix 16: bot/verify.go — Phone number accepts invalid formats

In `cleanPhoneNumber`, add a minimum length check. After stripping non-digits, require at least 10 digits:

```go
func cleanPhoneNumber(phone string) string {
    digits := strings.Map(func(r rune) rune {
        if r >= '0' && r <= '9' {
            return r
        }
        return -1
    }, phone)
    // Remove leading 1 for US numbers
    if len(digits) == 11 && digits[0] == '1' {
        digits = digits[1:]
    }
    // Require exactly 10 digits for a valid US phone number
    if len(digits) != 10 {
        return ""
    }
    return digits
}
```

---

## LOW (4) — Code quality, address when convenient

### Fix 17: bot/zoom.go — Duplicated roleInList helper

If `roleInList` (or similar) exists elsewhere in the codebase, remove the duplicate and use the existing one. If it only exists in zoom.go, leave it but add a comment: `// TODO: Extract to shared helpers package`.

### Fix 18: bot/activity_log.go — capitalize() panics on empty string

Find the `capitalize` function and add an empty string guard:
```go
func capitalize(s string) string {
    if s == "" {
        return ""
    }
    return strings.ToUpper(s[:1]) + s[1:]
}
```

### Fix 19: ghl/client.go — Raw body in error logs may contain sensitive data

In the `do()` method, truncate the response body in error messages:
```go
if resp.StatusCode >= 400 {
    bodyStr := string(respBody)
    if len(bodyStr) > 200 {
        bodyStr = bodyStr[:200] + "..."
    }
    return nil, fmt.Errorf("GHL API %d: %s", resp.StatusCode, bodyStr)
}
```

---

## After All Fixes

Run:
```bash
go build ./...
go vet ./...
```

Both must pass with zero output. Then commit:
```
git add -A
git commit -m "fix: address all 19 code audit issues (3 critical, 4 high, 8 medium, 4 low)"
```
