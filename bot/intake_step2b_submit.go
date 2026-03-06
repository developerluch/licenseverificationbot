package bot

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleStep2bSubmit processes the optional production details modal.
func (b *Bot) handleStep2bSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleStep2bSubmit: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleStep2bSubmit: %v", err)
		return
	}

	production := getModalValue(data, "production_written")
	leadSource := normalizeLeadSource(getModalValue(data, "lead_source"))
	vision := getModalValue(data, "vision_goals")
	comp := getModalValue(data, "comp_pct")
	showCompRaw := getModalValue(data, "show_comp")
	showComp := normalizeShowComp(showCompRaw)

	update := db.AgentUpdate{
		ProductionWritten: &production,
		LeadSource:        &leadSource,
		VisionGoals:       &vision,
		CompPct:           &comp,
		ShowComp:          &showComp,
	}

	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, update)
	b.db.LogActivity(context.Background(), userIDInt, "form_step2b", "Production details submitted")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "✅ Production details saved! Use `/contract` to book your contracting appointment.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
