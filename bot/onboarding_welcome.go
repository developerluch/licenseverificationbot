package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// postStartHereWelcome posts a personalized welcome message with a "Get Started" button
// in #start-here, pinging the user so they know it's for them.
func (b *Bot) postStartHereWelcome(s *discordgo.Session, userID string) {
	channelID := b.cfg.StartHereChannelID
	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "\U0001f44b Welcome to VIPA!",
		Description: "We're excited to have you here! Click **Get Started** below to begin your onboarding.\n\n" +
			"This takes about 60 seconds and unlocks access to all server channels.",
		Color: 0x2ECC71,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "\U0001f4cb What You'll Do",
				Value: "1\ufe0f\u20e3 Click **Get Started**\n" +
					"2\ufe0f\u20e3 Fill out a quick intro form\n" +
					"3\ufe0f\u20e3 Get your roles assigned\n" +
					"4\ufe0f\u20e3 Unlock all channels!",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA \u2022 This message is just for you \u2b07\ufe0f",
		},
	}

	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "<@" + userID + ">",
		Embeds:  []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Get Started",
						Style:    discordgo.SuccessButton,
						CustomID: "vipa:onboarding_get_started",
						Emoji:    &discordgo.ComponentEmoji{Name: "\U0001f680"},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Onboarding: failed to post start-here welcome for %s: %v", userID, err)
		return
	}

	// Track the message so we can delete it when they click "Get Started"
	b.welcomeMessages.Store(userID, welcomeMsgRef{
		ChannelID: channelID,
		MessageID: msg.ID,
	})
	log.Printf("Onboarding: posted welcome in #start-here for %s (msg %s)", userID, msg.ID)
}
