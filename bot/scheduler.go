package bot

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
	"license-bot-go/email"
	"license-bot-go/scrapers"
)

// StartScheduler runs a background loop that checks deadlines every 24 hours.
// It sends reminders at day 7, 14, and 21, and notifies admin when deadlines expire.
func (b *Bot) StartScheduler(ctx context.Context, mailer *email.Client) {
	log.Println("Scheduler started: checking deadlines every 24 hours")

	// Run immediately on startup, then every 24h
	b.runSchedulerCycle(mailer)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler shutting down")
			return
		case <-ticker.C:
			b.runSchedulerCycle(mailer)
		}
	}
}

func (b *Bot) runSchedulerCycle(mailer *email.Client) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Scheduler panic: %v", r)
		}
	}()

	log.Println("Scheduler: running deadline check cycle")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. Re-attempt verification for pending deadlines
	b.retryVerifications(ctx, mailer)

	// 2. Send reminders for approaching deadlines (every 7 days)
	b.sendReminders(ctx, mailer)

	// 3. Handle expired deadlines — notify admin
	b.handleExpiredDeadlines(ctx, mailer)

	// 4. Weekly check-ins (only on configured day)
	loc, _ := time.LoadLocation("America/New_York")
	nowET := time.Now().In(loc)
	if int(nowET.Weekday()) == b.cfg.CheckinDay {
		b.sendWeeklyCheckins(ctx)
	}

	// 5. Inactivity check (daily)
	b.checkInactivity(ctx)

	// 6. Recruiter nudges for unlicensed agents past threshold
	b.sendRecruiterNudges(ctx)

	// 7. Unlicensed agent 60-day kick check (daily)
	b.checkUnlicensedKicks(ctx)

	// 8. Daily tracker auto-post (only once per day, early afternoon ET)
	if nowET.Hour() >= 13 && nowET.Hour() < 14 {
		b.postDailyTracker(ctx)
	}
}

// retryVerifications attempts auto-verify for all pending deadlines.
func (b *Bot) retryVerifications(ctx context.Context, mailer *email.Client) {
	deadlines, err := b.db.GetPendingDeadlines(ctx, 0) // Get all pending
	if err != nil {
		log.Printf("Scheduler: failed to get pending deadlines: %v", err)
		return
	}

	for _, dl := range deadlines {
		if dl.FirstName == "" || dl.LastName == "" || dl.HomeState == "" {
			continue
		}

		verifyCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
		result := b.performVerification(verifyCtx, dl.FirstName, dl.LastName, dl.HomeState, dl.DiscordID, dl.GuildID)
		cancel()

		if result.Found && result.Match != nil {
			log.Printf("Scheduler: auto-verified %s %s (%d)", dl.FirstName, dl.LastName, dl.DiscordID)
			b.db.MarkDeadlineVerified(ctx, dl.DiscordID)

			// Assign role and notify
			userID := strconv.FormatInt(dl.DiscordID, 10)
			guildID := strconv.FormatInt(dl.GuildID, 10)

			if b.cfg.LicensedAgentRoleID != "" {
				b.session.GuildMemberRoleAdd(guildID, userID, b.cfg.LicensedAgentRoleID)
			}
			if b.cfg.StudentRoleID != "" {
				b.session.GuildMemberRoleRemove(guildID, userID, b.cfg.StudentRoleID)
			}

			// Discord DM
			b.dmUser(b.session, userID, fmt.Sprintf(
				"**Great news!** Your license has been verified for **%s %s** in **%s**!\n\n"+
					"You've been promoted to **Licensed Agent**. Use `/contract` to book your contracting appointment.",
				dl.FirstName, dl.LastName, dl.HomeState))

			// Email notification (if opted in)
			if mailer != nil {
				agent, err := b.db.GetAgent(ctx, dl.DiscordID)
				if err == nil && agent != nil && agent.Email != "" && agent.EmailOptIn {
					licNum := "N/A"
					if result.Match.LicenseNumber != "" {
						licNum = result.Match.LicenseNumber
					}
					if err := mailer.SendVerificationSuccess(agent.Email, dl.FirstName+" "+dl.LastName, dl.HomeState, licNum); err != nil {
						log.Printf("Scheduler: email failed for %d: %v", dl.DiscordID, err)
					}
				}
			}

			// Post to channel
			b.postSchedulerVerifyToChannel(result.Match, dl.HomeState, userID)

			// GHL sync
			go b.syncGHLStage(dl.DiscordID, db.StageVerified)
		}
	}
}

// sendReminders sends DM + email reminders for deadlines approaching.
// Reminders are sent at roughly day 7, 14, and 21 (every 7 days).
func (b *Bot) sendReminders(ctx context.Context, mailer *email.Client) {
	// Get deadlines that haven't had a reminder in the last 6 days
	deadlines, err := b.db.GetPendingDeadlines(ctx, 6*24*time.Hour)
	if err != nil {
		log.Printf("Scheduler: failed to get reminder deadlines: %v", err)
		return
	}

	for _, dl := range deadlines {
		daysLeft := int(math.Ceil(time.Until(dl.DeadlineAt).Hours() / 24))
		if daysLeft > 25 {
			continue // Too early for reminders (first week)
		}

		userID := strconv.FormatInt(dl.DiscordID, 10)

		// Discord DM reminder
		var urgency string
		switch {
		case daysLeft <= 7:
			urgency = "**URGENT:** "
		case daysLeft <= 14:
			urgency = "**Reminder:** "
		default:
			urgency = ""
		}

		msg := fmt.Sprintf(
			"%sYou have **%d days** left to get your insurance license verified.\n\n"+
				"We're checking state records automatically every day. "+
				"As soon as your license shows up, you'll be promoted to **Licensed Producer** automatically.\n\n"+
				"Want to check now? Use `/verify first_name:YourFirst last_name:YourLast state:XX`\n"+
				"Need help? Contact your upline.",
			urgency, daysLeft)

		b.dmUser(b.session, userID, msg)

		// Email reminder (if opted in)
		if mailer != nil {
			agent, err := b.db.GetAgent(ctx, dl.DiscordID)
			if err == nil && agent != nil && agent.Email != "" && agent.EmailOptIn {
				if err := mailer.SendReminder(agent.Email, dl.FirstName+" "+dl.LastName, daysLeft); err != nil {
					log.Printf("Scheduler: email failed for %d: %v", dl.DiscordID, err)
				}
			}
		}

		// Mark reminder sent
		b.db.UpdateReminderSent(ctx, dl.DiscordID)
		log.Printf("Scheduler: sent %d-day reminder to %d (%s %s)", daysLeft, dl.DiscordID, dl.FirstName, dl.LastName)
	}
}

// handleExpiredDeadlines notifies admin about users who missed their deadline.
func (b *Bot) handleExpiredDeadlines(ctx context.Context, mailer *email.Client) {
	expired, err := b.db.GetExpiredDeadlines(ctx)
	if err != nil {
		log.Printf("Scheduler: failed to get expired deadlines: %v", err)
		return
	}

	for _, dl := range expired {
		userID := strconv.FormatInt(dl.DiscordID, 10)

		// DM the user
		b.dmUser(b.session, userID,
			"**Your 30-day verification deadline has passed.**\n\n"+
				"An admin has been notified. Please contact your upline to discuss next steps.")

		// Email the user (if opted in)
		if mailer != nil {
			agent, err := b.db.GetAgent(ctx, dl.DiscordID)
			if err == nil && agent != nil && agent.Email != "" && agent.EmailOptIn {
				mailer.SendDeadlineExpired(agent.Email, dl.FirstName+" "+dl.LastName)
			}
		}

		// Notify admin channel
		b.notifyAdmin(dl, userID)

		// Mark as admin notified
		b.db.MarkAdminNotified(ctx, dl.DiscordID)
		log.Printf("Scheduler: expired deadline for %d (%s %s), admin notified", dl.DiscordID, dl.FirstName, dl.LastName)
	}
}

func (b *Bot) notifyAdmin(dl db.VerificationDeadline, userID string) {
	channelID := b.cfg.AdminNotificationChannelID
	if channelID == "" {
		channelID = b.cfg.LicenseCheckChannelID
	}
	if channelID == "" {
		channelID = b.cfg.HiringLogChannelID
	}
	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "Verification Deadline Expired",
		Description: fmt.Sprintf(
			"<@%s> has not verified their license within 30 days.\n\n"+
				"**Name:** %s %s\n"+
				"**State:** %s\n"+
				"**Status:** %s\n"+
				"**Deadline:** %s\n\n"+
				"Please follow up with this recruit.",
			userID, dl.FirstName, dl.LastName,
			nvl(dl.HomeState, "Unknown"),
			dl.LicenseStatus,
			dl.DeadlineAt.Format("January 2, 2006"),
		),
		Color:     0xE74C3C, // Red
		Timestamp: time.Now().Format(time.RFC3339),
	}

	b.session.ChannelMessageSendEmbed(channelID, embed)
}

func (b *Bot) postSchedulerVerifyToChannel(match *scrapers.LicenseResult, state, userID string) {
	channelID := b.verifyLogChannelID()
	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "License Auto-Verified (Scheduled Check)",
		Description: fmt.Sprintf(
			"<@%s> was automatically verified during a scheduled check.\n\n"+
				"**Name:** %s\n"+
				"**License #:** %s\n"+
				"**State:** %s | **Status:** %s",
			userID, match.FullName, nvl(match.LicenseNumber, "N/A"), state, match.Status,
		),
		Color:     0x2ECC71,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	b.session.ChannelMessageSendEmbed(channelID, embed)
}

// sendRecruiterNudges DMs recruiters about their unlicensed recruits past the nudge threshold.
func (b *Bot) sendRecruiterNudges(ctx context.Context) {
	agents, err := b.db.GetAgentsNeedingNudge(ctx, b.cfg.NudgeAfterDays, 7)
	if err != nil {
		log.Printf("Scheduler: failed to get agents needing nudge: %v", err)
		return
	}
	if len(agents) == 0 {
		return
	}

	// Group by upline
	type nudgeGroup struct {
		recruiterDiscordID int64
		recruiterName      string
		agentNames         []string
	}
	groups := make(map[string]*nudgeGroup)

	for _, a := range agents {
		key := a.UplineManager
		if key == "" {
			key = "__unassigned__"
		}
		g, ok := groups[key]
		if !ok {
			g = &nudgeGroup{
				recruiterDiscordID: a.UplineManagerDiscordID,
				recruiterName:      a.UplineManager,
			}
			groups[key] = g
		}
		name := a.FirstName + " " + a.LastName
		if name == " " {
			name = fmt.Sprintf("Agent #%d", a.DiscordID)
		}
		g.agentNames = append(g.agentNames, name)
	}

	for _, g := range groups {
		var lines []string
		for _, name := range g.agentNames {
			lines = append(lines, "- "+name)
		}
		msg := fmt.Sprintf(
			"**Recruiter Nudge:** The following recruits have been unlicensed for %d+ days:\n\n%s\n\n"+
				"Please follow up with them to check on their licensing progress.",
			b.cfg.NudgeAfterDays, strings.Join(lines, "\n"))

		if g.recruiterDiscordID != 0 {
			recruiterID := strconv.FormatInt(g.recruiterDiscordID, 10)
			b.dmUser(b.session, recruiterID, msg)
			log.Printf("Scheduler: nudged recruiter %s (%d) about %d unlicensed agents",
				g.recruiterName, g.recruiterDiscordID, len(g.agentNames))
		} else {
			channelID := b.cfg.AdminNotificationChannelID
			if channelID == "" {
				channelID = b.cfg.LicenseCheckChannelID
			}
			if channelID != "" {
				embed := &discordgo.MessageEmbed{
					Title:       fmt.Sprintf("Recruiter Nudge: %s", nvl(g.recruiterName, "Unknown")),
					Description: msg,
					Color:       0xF39C12,
					Timestamp:   time.Now().Format(time.RFC3339),
				}
				b.session.ChannelMessageSendEmbed(channelID, embed)
			}
		}
	}

	// Mark nudges sent
	for _, a := range agents {
		b.db.UpdateNudgeSent(ctx, a.DiscordID)
	}
}

// postDailyTracker posts the overall tracker stats to the configured tracker channel.
func (b *Bot) postDailyTracker(ctx context.Context) {
	channelID := b.cfg.TrackerChannelID
	if channelID == "" {
		return
	}

	stats, err := b.db.GetOverallTrackerStats(ctx)
	if err != nil {
		log.Printf("Scheduler: failed to get tracker stats: %v", err)
		return
	}

	bar := progressBar(stats.Percentage)

	agencies, _ := b.db.GetAgencyTrackerStats(ctx)
	var agencyLines []string
	for _, a := range agencies {
		agencyLines = append(agencyLines, fmt.Sprintf("**%s:** %d/%d (%.0f%%)",
			a.Agency, a.LicensedAgents, a.TotalAgents, a.Percentage))
	}

	description := fmt.Sprintf(
		"Licensed: **%d/%d** (%.1f%%)\n%s",
		stats.LicensedAgents, stats.TotalAgents, stats.Percentage, bar)

	if len(agencyLines) > 0 {
		description += "\n\n" + strings.Join(agencyLines, "\n")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Daily License Tracker",
		Description: description,
		Color:       0x3498DB,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	b.session.ChannelMessageSendEmbed(channelID, embed)
	log.Println("Scheduler: posted daily tracker update")
}

// checkUnlicensedKicks checks all unlicensed (Student) agents and:
// - Sends DM warnings at configured days (default: 15, 30, 45, 59)
// - Kicks at the configured deadline day (default: 60)
// - Logs kicks to #audit-logs and DMs the agent's upline manager
func (b *Bot) checkUnlicensedKicks(ctx context.Context) {
	kickDays := b.cfg.UnlicensedKickDays
	if kickDays <= 0 {
		return
	}
	warnDays := b.cfg.UnlicensedWarnDaysList()

	// Get all agents in Student stage who completed the form
	agents, err := b.db.GetUnlicensedAgents(ctx)
	if err != nil {
		log.Printf("Scheduler: failed to get unlicensed agents: %v", err)
		return
	}

	guildID := b.cfg.GuildID

	for _, agent := range agents {
		if agent.FormCompletedAt == nil {
			continue
		}

		daysSinceForm := int(time.Since(*agent.FormCompletedAt).Hours() / 24)
		userID := strconv.FormatInt(agent.DiscordID, 10)

		// Check if they should be kicked
		if daysSinceForm >= kickDays {
			b.kickUnlicensedAgent(ctx, agent, userID, guildID, daysSinceForm)
			continue
		}

		// Check if they need a warning
		for _, warnDay := range warnDays {
			if daysSinceForm == warnDay {
				daysLeft := kickDays - daysSinceForm
				b.warnUnlicensedAgent(agent, userID, daysSinceForm, daysLeft)
				break
			}
		}
	}
}

// warnUnlicensedAgent sends a DM warning about the approaching kick deadline.
func (b *Bot) warnUnlicensedAgent(agent db.AgentRow, userID string, daysSinceForm, daysLeft int) {
	var urgency string
	if daysLeft <= 1 {
		urgency = "**FINAL WARNING:** "
	} else if daysLeft <= 7 {
		urgency = "**URGENT:** "
	} else {
		urgency = ""
	}

	name := agent.FirstName
	if name == "" {
		name = "Agent"
	}

	msg := fmt.Sprintf(
		"%sHi %s, you have **%d day(s) remaining** to get your insurance license verified.\n\n"+
			"You joined VIPA %d days ago. All agents must be licensed within %d days or will be removed from the server.\n\n"+
			"**How to verify:** Use `/verify first_name:YourFirst last_name:YourLast state:XX` in the server.\n\n"+
			"Need help? Contact your upline manager.",
		urgency, name, daysLeft, daysSinceForm, b.cfg.UnlicensedKickDays)

	b.dmUser(b.session, userID, msg)
	log.Printf("Scheduler: warned unlicensed agent %s (%s) — day %d, %d days left",
		name, userID, daysSinceForm, daysLeft)
}

// kickUnlicensedAgent removes the agent from the server and logs it.
func (b *Bot) kickUnlicensedAgent(ctx context.Context, agent db.AgentRow, userID, guildID string, daysSinceForm int) {
	name := agent.FirstName + " " + agent.LastName
	if name == " " {
		name = fmt.Sprintf("Agent #%d", agent.DiscordID)
	}

	// DM the agent before kicking
	b.dmUser(b.session, userID, fmt.Sprintf(
		"Hi %s, your %d-day deadline to obtain your insurance license has expired.\n\n"+
			"You have been removed from the VIPA Discord server. "+
			"If you believe this is an error or wish to return, please contact your upline manager to request a re-invite.",
		agent.FirstName, b.cfg.UnlicensedKickDays))

	// Kick from server
	reason := fmt.Sprintf("Unlicensed after %d days (auto-kick)", daysSinceForm)
	err := b.session.GuildMemberDeleteWithReason(guildID, userID, reason)
	if err != nil {
		log.Printf("Scheduler: failed to kick %s (%s): %v", name, userID, err)
		return
	}

	log.Printf("Scheduler: kicked unlicensed agent %s (%s) after %d days", name, userID, daysSinceForm)

	// Update DB
	stage := db.StageKicked
	b.db.UpsertAgent(ctx, agent.DiscordID, agent.GuildID, db.AgentUpdate{
		CurrentStage: &stage,
	})
	b.db.LogActivity(ctx, agent.DiscordID, "kicked", reason)

	// Post to #audit-logs
	auditChannelID := b.cfg.AuditLogChannelID
	if auditChannelID == "" {
		auditChannelID = b.cfg.AdminNotificationChannelID
	}
	if auditChannelID != "" {
		embed := &discordgo.MessageEmbed{
			Title: "Agent Kicked — Unlicensed Deadline Expired",
			Description: fmt.Sprintf(
				"**Agent:** %s (<@%s>)\n"+
					"**Agency:** %s\n"+
					"**Upline:** %s\n"+
					"**Days Elapsed:** %d\n"+
					"**Deadline:** %d days\n\n"+
					"Agent was automatically removed for not obtaining their license within the required timeframe.",
				name, userID,
				nvl(agent.Agency, "Unknown"),
				nvl(agent.UplineManager, "Unknown"),
				daysSinceForm,
				b.cfg.UnlicensedKickDays),
			Color:     0xE74C3C,
			Timestamp: time.Now().Format(time.RFC3339),
			Footer: &discordgo.MessageEmbedFooter{
				Text: "VIPA Auto-Kick System",
			},
		}
		b.session.ChannelMessageSendEmbed(auditChannelID, embed)
	}

	// Also post to #hiring-log
	if b.cfg.HiringLogChannelID != "" {
		embed := &discordgo.MessageEmbed{
			Title: "Agent Removed — License Deadline",
			Description: fmt.Sprintf("**%s** (Agency: %s) was removed after %d days without a license.",
				name, nvl(agent.Agency, "Unknown"), daysSinceForm),
			Color:     0xE74C3C,
			Timestamp: time.Now().Format(time.RFC3339),
		}
		b.session.ChannelMessageSendEmbed(b.cfg.HiringLogChannelID, embed)
	}

	// DM the upline manager
	if agent.UplineManagerDiscordID != 0 {
		managerID := strconv.FormatInt(agent.UplineManagerDiscordID, 10)
		b.dmUser(b.session, managerID, fmt.Sprintf(
			"**Agent Removed:** %s from your team has been removed from the VIPA server after %d days without obtaining their license.\n\n"+
				"If you'd like to give them another chance, you can re-invite them to the server. They'll need to go through onboarding again.",
			name, daysSinceForm))
	}
}
