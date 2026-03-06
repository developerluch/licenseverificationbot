package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

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
