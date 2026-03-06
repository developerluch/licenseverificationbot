package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func buildGreetingsCardEmbed(member *discordgo.Member, data map[string]string) (*discordgo.MessageEmbed, string) {
	fields := []*discordgo.MessageEmbedField{}

	if data["role_background"] != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "\U0001f4bc Role & Background", Value: data["role_background"], Inline: false,
		})
	}
	if data["home_state"] != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "\U0001f4cd Home State", Value: data["home_state"], Inline: true,
		})
	}
	if data["agency"] != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "\U0001f3e2 Agency", Value: data["agency"], Inline: true,
		})
	}
	if data["goals_vision"] != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "\U0001f3af Goals & Vision", Value: data["goals_vision"], Inline: false,
		})
	}
	if data["fun_hobbies"] != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "\U0001f3ae Fun & Hobbies", Value: data["fun_hobbies"], Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("\U0001f91d Meet %s!", data["full_name"]),
		Description: fmt.Sprintf("Welcome <@%s> to the team! Drop a \U0001f44b and say hello!", member.User.ID),
		Color:       0x2ECC71,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Onboarding \u2022 Welcome to the team!",
		},
	}

	if member.User.AvatarURL("") != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: member.User.AvatarURL("128")}
	}

	rolePing := ""
	return embed, rolePing
}
