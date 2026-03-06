package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

func (b *Bot) postAutoVerifyToChannel(s *discordgo.Session, e *discordgo.GuildMemberUpdate, match *scrapers.LicenseResult, state string) {
	channelID := b.verifyLogChannelID()
	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "License Auto-Verified!",
		Description: fmt.Sprintf(
			"<@%s> was automatically verified as a licensed agent.\n\n"+
				"**Name:** %s\n"+
				"**License #:** %s\n"+
				"**State:** %s | **Status:** %s",
			e.User.ID, match.FullName, nvl(match.LicenseNumber, "N/A"), state, match.Status,
		),
		Color:     0x2ECC71,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}
