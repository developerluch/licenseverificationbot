package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) sendApprovalDM(s *discordgo.Session, ownerID, userID, fullName, agency, licenseStatus string, reqID int, ctx context.Context) {
	embed := &discordgo.MessageEmbed{
		Title: "New Agent Approval Request",
		Description: fmt.Sprintf(
			"A new agent has completed onboarding and needs your approval.\n\n"+
				"**Agent:** <@%s> (%s)\n"+
				"**Agency:** %s\n"+
				"**License Status:** %s\n\n"+
				"Please approve or deny this agent.",
			userID, fullName, agency, licenseStatus),
		Color:     0x3498DB,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	channel, err := s.UserChannelCreate(ownerID)
	if err != nil {
		log.Printf("Approval: failed to DM owner %s: %v", ownerID, err)
		b.postApprovalToChannel(s, embed, reqID)
		return
	}

	msg, err := s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Approve",
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("vipa:approve:%d", reqID),
					},
					discordgo.Button{
						Label:    "Deny",
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("vipa:deny:%d", reqID),
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Approval: failed to send DM to owner %s: %v", ownerID, err)
		b.postApprovalToChannel(s, embed, reqID)
		return
	}

	// Store message ID for editing later
	b.db.UpdateApprovalDMMessageID(ctx, reqID, msg.ID)
	log.Printf("Approval: sent request #%d to owner %s", reqID, ownerID)
}
