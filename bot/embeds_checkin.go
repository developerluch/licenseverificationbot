package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func buildCheckinEmbed(agentName string, weeksIn int) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("\U0001f4cb Weekly Check-In \u2014 Week %d", weeksIn),
		Description: "How's your progress this week?",
		Color:       0xF39C12,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "\u2705 On Track", Value: "Everything is going well, I'm making progress!", Inline: false},
			{Name: "\u23f8\ufe0f Need Help", Value: "I could use some guidance or support.", Inline: false},
			{Name: "\U0001f393 I Got Licensed!", Value: "I passed my exam and got my license!", Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Student Support \u2022 Reply within 7 days to stay active",
		},
	}
}
