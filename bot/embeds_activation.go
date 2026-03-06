package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func buildActivationEmbed(member *discordgo.Member, agency string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "\U0001f389\U0001f680 New Active Agent!",
		Description: fmt.Sprintf("<@%s> has completed all setup steps and is now a fully active VIPA agent!", member.User.ID),
		Color:       0x2ECC71,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Agency", Value: nvl(agency, "N/A"), Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA \u2022 Welcome to the team!",
		},
	}

	if member.User.AvatarURL("") != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: member.User.AvatarURL("128")}
	}

	return embed
}
