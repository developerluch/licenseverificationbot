package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func buildAgentProfileEmbed(agent *db.Agent, member *discordgo.Member) *discordgo.MessageEmbed {
	stageName := stageLabel(agent.CurrentStage)

	fields := []*discordgo.MessageEmbedField{
		{Name: "Discord", Value: fmt.Sprintf("<@%d>", agent.DiscordID), Inline: true},
		{Name: "Agency", Value: nvl(agent.Agency, "N/A"), Inline: true},
		{Name: "Stage", Value: stageName, Inline: true},
		{Name: "Upline", Value: nvl(agent.UplineManager, "N/A"), Inline: true},
		{Name: "Experience", Value: titleCase(nvl(agent.ExperienceLevel, "N/A")), Inline: true},
		{Name: "License", Value: titleCase(nvl(agent.LicenseStatus, "N/A")), Inline: true},
	}

	if agent.LicenseNPN != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "NPN", Value: agent.LicenseNPN, Inline: true})
	}
	if agent.State != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "State", Value: agent.State, Inline: true})
	}
	if agent.CompPct != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "\U0001f512 Comp %", Value: agent.CompPct, Inline: true})
	}
	if agent.ProductionWritten != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Production", Value: agent.ProductionWritten, Inline: true})
	}
	if agent.LeadSource != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Lead Source", Value: titleCase(agent.LeadSource), Inline: true})
	}
	if agent.VisionGoals != "" {
		vision := agent.VisionGoals
		if len(vision) > 200 {
			vision = vision[:200] + "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Vision", Value: vision, Inline: false})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "Joined", Value: agent.CreatedAt.Format("2006-01-02"), Inline: true,
	})
	if agent.ActivatedAt != nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: "Activated", Value: agent.ActivatedAt.Format("2006-01-02"), Inline: true,
		})
	}

	name := agent.FirstName + " " + agent.LastName
	embed := &discordgo.MessageEmbed{
		Title:  fmt.Sprintf("\U0001f464 Agent Profile: %s", strings.TrimSpace(name)),
		Color:  0x3498DB,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Agent ID: %d \u2022 \U0001f512 This info is private", agent.DiscordID),
		},
	}

	if member != nil && member.User.AvatarURL("") != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: member.User.AvatarURL("128")}
	}

	return embed
}
