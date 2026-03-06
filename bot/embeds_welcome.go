package bot

import (
	"github.com/bwmarrin/discordgo"
)

func buildWelcomeEmbed() *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       "\U0001f44b Welcome to VIPA!",
		Description: "Welcome to the **Virtual Insurance Producer Alliance**! We're excited to have you on the team.",
		Color:       0x2ECC71,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "\U0001f4cb What Happens Next",
				Value: "1\ufe0f\u20e3 Click **Get Started** below\n" +
					"2\ufe0f\u20e3 Fill out a quick intro form (2 steps)\n" +
					"3\ufe0f\u20e3 Get your roles assigned automatically\n" +
					"4\ufe0f\u20e3 Start your journey!",
				Inline: false,
			},
			{
				Name:   "\U0001f393 Not Licensed Yet?",
				Value:  "No worries! We support agents at every stage \u2014 from studying to producing.",
				Inline: false,
			},
			{
				Name:   "\U0001f6e1\ufe0f Your Info is Safe",
				Value:  "Everything you share is private and only used for onboarding.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA \u2022 Click Get Started below \u2b07\ufe0f",
		},
	}
}
