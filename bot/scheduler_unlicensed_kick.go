package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

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
