package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

func (b *Bot) dmNextSteps(s *discordgo.Session, i *discordgo.InteractionCreate, match *scrapers.LicenseResult, state string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("dmNextSteps panic: %v", r)
		}
	}()

	channel, err := s.UserChannelCreate(i.Member.User.ID)
	if err != nil {
		log.Printf("Cannot create DM channel: %v", err)
		return
	}

	fields := buildLicenseFields(match, state)

	embed := &discordgo.MessageEmbed{
		Title:       "License Verified!",
		Description: fmt.Sprintf("Welcome **%s**! Your license has been confirmed. Here are your full license details:", i.Member.User.Username),
		Color:       0x2ECC71,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA License Verification",
		},
	}

	s.ChannelMessageSendEmbed(channel.ID, embed)
}
