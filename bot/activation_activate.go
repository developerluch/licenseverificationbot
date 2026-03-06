package bot

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// activateAgent promotes an agent to active status.
func (b *Bot) activateAgent(s *discordgo.Session, discordID, guildID int64, member *discordgo.Member) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("activateAgent panic: %v", r)
		}
	}()

	now := time.Now()
	stage := db.StageActive
	setupDone := true
	b.db.UpsertAgent(context.Background(), discordID, guildID, db.AgentUpdate{
		CurrentStage:   &stage,
		SetupCompleted: &setupDone,
		ActivatedAt:    &now,
		LastActive:     &now,
	})
	b.db.LogActivity(context.Background(), discordID, "activated", "Agent completed setup and is now active")

	// GHL sync
	go b.syncGHLStage(discordID, db.StageActive)

	userID := strconv.FormatInt(discordID, 10)
	guildIDStr := strconv.FormatInt(guildID, 10)

	// Swap roles: remove Student + Licensed-Agent, add Active-Agent
	if b.cfg.ActiveAgentRoleID != "" {
		if err := s.GuildMemberRoleAdd(guildIDStr, userID, b.cfg.ActiveAgentRoleID); err != nil {
			log.Printf("Activation: failed to add Active-Agent role: %v", err)
		}
	}
	if b.cfg.StudentRoleID != "" {
		s.GuildMemberRoleRemove(guildIDStr, userID, b.cfg.StudentRoleID)
	}
	if b.cfg.LicensedAgentRoleID != "" {
		s.GuildMemberRoleRemove(guildIDStr, userID, b.cfg.LicensedAgentRoleID)
	}

	// Post activation announcement
	agent, _ := b.db.GetAgent(context.Background(), discordID)
	agency := ""
	if agent != nil {
		agency = agent.Agency
	}
	embed := buildActivationEmbed(member, agency)

	if b.cfg.GreetingsChannelID != "" {
		s.ChannelMessageSendEmbed(b.cfg.GreetingsChannelID, embed)
	}
	if b.cfg.HiringLogChannelID != "" {
		s.ChannelMessageSendEmbed(b.cfg.HiringLogChannelID, embed)
	}

	// Congrats DM
	b.dmUser(s, userID,
		"\U0001f389 **Congratulations!** You're now a fully active VIPA agent!\n\n"+
			"All setup steps are complete. Welcome to the team!")
}
