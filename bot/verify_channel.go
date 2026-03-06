package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

func (b *Bot) postToChannel(s *discordgo.Session, i *discordgo.InteractionCreate, match *scrapers.LicenseResult, state string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("postToChannel panic: %v", r)
		}
	}()

	channelID := b.verifyLogChannelID()
	if channelID == "" {
		return
	}

	userID := i.Member.User.ID
	embed := &discordgo.MessageEmbed{
		Title: "License Verified!",
		Description: fmt.Sprintf(
			"<@%s> verified as a licensed agent.\n\n"+
				"**Name:** %s\n"+
				"**License #:** %s\n"+
				"**State:** %s | **Status:** %s",
			userID, match.FullName, nvl(match.LicenseNumber, "N/A"), state, match.Status,
		),
		Color:     0x2ECC71,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}
