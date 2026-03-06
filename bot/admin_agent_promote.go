package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func (b *Bot) handleAgentPromote(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	var level string
	for _, opt := range opts {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "level":
			level = opt.StringValue()
		}
	}

	if targetUser == nil {
		return
	}

	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleAgentPromote: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleAgentPromote: %v", err)
		return
	}

	switch level {
	case "licensed":
		stage := db.StageVerified
		b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
			CurrentStage: &stage,
		})
		if b.cfg.LicensedAgentRoleID != "" {
			s.GuildMemberRoleAdd(i.GuildID, targetUser.ID, b.cfg.LicensedAgentRoleID)
		}
		if b.cfg.StudentRoleID != "" {
			s.GuildMemberRoleRemove(i.GuildID, targetUser.ID, b.cfg.StudentRoleID)
		}
		b.db.LogActivity(context.Background(), userIDInt, "promoted", "Manually promoted to Licensed by staff")

	case "active":
		member, _ := s.GuildMember(i.GuildID, targetUser.ID)
		if member == nil {
			member = &discordgo.Member{User: targetUser}
		}
		b.activateAgent(s, userIDInt, guildIDInt, member)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@%s> has been promoted to **%s**.", targetUser.ID, level),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
