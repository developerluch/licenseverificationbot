package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleRoleAudit checks for role conflicts across all guild members.
func (b *Bot) handleRoleAudit(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	if !b.cfg.IsStaff(i.Member.Roles) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is restricted to staff.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	agents, err := b.db.GetAllAgents(ctx, false)
	if err != nil {
		b.followUp(s, i, fmt.Sprintf("Error: %v", err))
		return
	}

	conflicts := b.auditAgentRoles(s, agents)

	if len(conflicts) == 0 {
		b.followUp(s, i, "No role conflicts found.")
		return
	}

	desc := strings.Join(conflicts, "\n")
	if len(desc) > 4000 {
		desc = desc[:4000] + "\n... (truncated)"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Role Audit — %d Conflicts", len(conflicts)),
		Description: desc,
		Color:       0xE74C3C,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}
