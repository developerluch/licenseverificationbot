package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// postApprovalToChannel posts the approval request to admin channel as fallback.
func (b *Bot) postApprovalToChannel(s *discordgo.Session, embed *discordgo.MessageEmbed, reqID int) {
	channelID := b.cfg.AdminNotificationChannelID
	if channelID == "" {
		channelID = b.cfg.HiringLogChannelID
	}
	if channelID == "" {
		return
	}
	s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
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
}
