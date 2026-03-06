package bot

import (
	"github.com/bwmarrin/discordgo"
)

func buildRulesEmbed() *discordgo.MessageEmbed {
	rulesText := "1. **Professionalism** \u2014 Treat everyone with respect.\n" +
		"2. **No Poaching** \u2014 Do not recruit agents from within VIPA.\n" +
		"3. **Comp Privacy** \u2014 Never share your compensation details publicly.\n" +
		"4. **No Spam** \u2014 Keep channels on-topic.\n" +
		"5. **Chain of Command** \u2014 Follow your upline structure.\n" +
		"6. **Client Info** \u2014 Never share client PII in public channels.\n" +
		"7. **Real Names** \u2014 Use your real name as your display name.\n" +
		"8. **Stay Active** \u2014 Check in weekly or risk removal.\n" +
		"9. **No Drama** \u2014 Handle conflicts privately or with leadership.\n" +
		"10. **Discord ToS** \u2014 Follow Discord's Terms of Service."

	return &discordgo.MessageEmbed{
		Title:       "\U0001f4cb VIPA Server Rules",
		Description: rulesText,
		Color:       0x3498DB,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "\u26a0\ufe0f Enforcement",
				Value:  "Violations may result in warnings, role removal, or server removal at leadership's discretion.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA \u2022 Last updated February 2026",
		},
	}
}
