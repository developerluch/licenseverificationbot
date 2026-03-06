package bot

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func buildSetupEmbed(agentName string, completed map[string]bool) *discordgo.MessageEmbed {
	total := len(db.SetupItems)
	done := 0
	var fieldLines []string

	for _, item := range db.SetupItems {
		status := "\u25fd" // white medium square
		statusText := "Click button below to mark done"
		if completed[item.Key] {
			status = "\u2705"
			statusText = "Completed"
			done++
		}
		fieldLines = append(fieldLines, fmt.Sprintf("%s %s %s \u2014 %s", status, item.Emoji, item.Label, statusText))
	}

	// Progress bar
	barLen := 20
	filled := 0
	if total > 0 {
		filled = (done * barLen) / total
	}
	bar := strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", barLen-filled)
	progress := fmt.Sprintf("%s %d/%d", bar, done, total)

	color := 0x3498DB
	if done == total {
		color = 0x2ECC71
	}

	return &discordgo.MessageEmbed{
		Title:       "\U0001f527 Agent Setup Checklist",
		Description: fmt.Sprintf("**Progress:** %s\n\nComplete all items below to unlock full agent access.\n\n%s", progress, strings.Join(fieldLines, "\n")),
		Color:       color,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Agent Setup \u2022 Complete all items to unlock full access",
		},
	}
}

func buildContractingEmbed(managers []db.ContractingManager) *discordgo.MessageEmbed {
	fields := []*discordgo.MessageEmbedField{}

	for _, m := range managers {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("\U0001f4c5 %s", m.ManagerName),
			Value:  m.CalendlyURL,
			Inline: true,
		})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "\U0001f4dd What to Prepare",
		Value: "- Government-issued photo ID\n" +
			"- Social Security number\n" +
			"- E&O insurance info\n" +
			"- Bank info for direct deposit\n" +
			"- Resident state license number",
		Inline: false,
	})

	return &discordgo.MessageEmbed{
		Title:       "\U0001f4c5 Book Your Contracting Appointment",
		Description: "Click a link below to schedule with a contracting manager.",
		Color:       0x9B59B6,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Contracting \u2022 Appointments are typically 30 minutes",
		},
	}
}
