package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"license-bot-go/db"
)

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
