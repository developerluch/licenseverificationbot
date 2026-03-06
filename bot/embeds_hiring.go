package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

func buildHiringLogEmbed(member *discordgo.Member, data map[string]string) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{
		{Name: "Discord", Value: fmt.Sprintf("<@%s>", member.User.ID), Inline: true},
		{Name: "Agency", Value: nvl(data["agency"], "N/A"), Inline: true},
		{Name: "Upline", Value: nvl(data["upline"], "N/A"), Inline: true},
		{Name: "Experience", Value: titleCase(nvl(data["experience"], "N/A")), Inline: true},
		{Name: "License Status", Value: titleCase(nvl(data["license_status"], "N/A")), Inline: true},
	}

	if data["production_written"] != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Production", Value: data["production_written"], Inline: true,
		})
	}
	if data["lead_source"] != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Lead Source", Value: data["lead_source"], Inline: true,
		})
	}
	if data["goals_vision"] != "" {
		vision := data["goals_vision"]
		if len(vision) > 200 {
			vision = vision[:200] + "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Vision / Goals", Value: vision, Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("\U0001f4cb New Agent: %s", data["full_name"]),
		Color:     0x3498DB,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Agent ID: %s \u2022 VIPA Onboarding", member.User.ID),
		},
	}

	if member.User.AvatarURL("") != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: member.User.AvatarURL("128")}
	}

	return embed
}
