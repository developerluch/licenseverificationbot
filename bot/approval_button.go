package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleApprovalButton handles Approve/Deny button clicks.
func (b *Bot) handleApprovalButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return
	}

	action := parts[1] // "approve" or "deny"
	reqID, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := b.db.GetApprovalRequest(ctx, reqID)
	if err != nil || req == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Approval request not found.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if req.Status != "pending" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("This request has already been %s.", req.Status),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Authorization: only the designated owner or staff can approve/deny
	responderID := ""
	if i.Member != nil {
		responderID = i.Member.User.ID
	} else if i.User != nil {
		responderID = i.User.ID
	}
	ownerIDStr := strconv.FormatInt(req.OwnerDiscordID, 10)
	if responderID != ownerIDStr && (i.Member == nil || !b.cfg.IsStaff(i.Member.Roles)) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are not authorized to respond to this approval request.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	agentUserID := strconv.FormatInt(req.AgentDiscordID, 10)
	guildID := strconv.FormatInt(req.GuildID, 10)

	switch action {
	case "approve":
		b.handleApprovalApprove(s, i, ctx, reqID, req, agentUserID, guildID)
	case "deny":
		b.handleApprovalDeny(s, i, reqID)
	}
}
