package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleDenyReasonModal processes the denial reason modal.
func (b *Bot) handleDenyReasonModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return
	}
	reqID, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}

	reason := getModalValue(i.ModalSubmitData(), "reason")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := b.db.GetApprovalRequest(ctx, reqID)
	if err != nil || req == nil || req.Status != "pending" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Request already processed.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	b.db.DenyAgent(ctx, reqID, reason)

	deniedStatus := "denied"
	b.db.UpsertAgent(ctx, req.AgentDiscordID, req.GuildID, db.AgentUpdate{
		ApprovalStatus: &deniedStatus,
	})

	agentUserID := strconv.FormatInt(req.AgentDiscordID, 10)
	guildID := strconv.FormatInt(req.GuildID, 10)

	// Remove pending role
	if b.cfg.PendingRoleID != "" {
		s.GuildMemberRoleRemove(guildID, agentUserID, b.cfg.PendingRoleID)
	}

	// DM the agent
	msg := "Your onboarding request has been reviewed by your agency owner and was not approved at this time."
	if reason != "" {
		msg += fmt.Sprintf("\n\n**Reason:** %s", reason)
	}
	msg += "\n\nPlease contact your upline for more information."
	b.dmUser(s, agentUserID, msg)

	// Respond
	desc := fmt.Sprintf("<@%s> has been denied for **%s**.", agentUserID, req.Agency)
	if reason != "" {
		desc += fmt.Sprintf("\n**Reason:** %s", reason)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{
				Title:       "Agent Denied",
				Description: desc,
				Color:       0xE74C3C,
				Timestamp:   time.Now().Format(time.RFC3339),
			}},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	log.Printf("Approval: request #%d denied (reason: %s)", reqID, reason)
}
