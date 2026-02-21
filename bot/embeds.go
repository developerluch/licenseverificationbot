package bot

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
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

// === Helpers ===

func stageLabel(stage int) string {
	switch stage {
	case 1:
		return "1 \u2014 Joined"
	case 2:
		return "2 \u2014 Form Started"
	case 3:
		return "3 \u2014 Sorted"
	case 4:
		return "4 \u2014 Student"
	case 5:
		return "5 \u2014 Licensed"
	case 6:
		return "6 \u2014 Contracting"
	case 7:
		return "7 \u2014 Setup"
	case 8:
		return "8 \u2014 Active"
	default:
		return fmt.Sprintf("%d \u2014 Unknown", stage)
	}
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "_", " ")
	return strings.Title(s)
}
