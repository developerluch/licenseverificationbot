package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func (b *Bot) handleAgentList(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var stage int
	for _, opt := range opts {
		if opt.Name == "stage" {
			stage = int(opt.IntValue())
		}
	}

	var agents []db.Agent
	var err error
	if stage > 0 {
		agents, err = b.db.GetAgentsByStage(context.Background(), stage)
	} else {
		agents, err = b.db.GetAllAgents(context.Background(), false)
	}

	if err != nil {
		log.Printf("admin handler: handleAgentList: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An error occurred. Please try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(agents) == 0 {
		msg := "No agents found."
		if stage > 0 {
			msg = fmt.Sprintf("No agents at stage %d.", stage)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	limit := 25
	if len(agents) < limit {
		limit = len(agents)
	}

	var lines []string
	for _, a := range agents[:limit] {
		name := strings.TrimSpace(a.FirstName + " " + a.LastName)
		if name == "" {
			name = "Unknown"
		}
		lines = append(lines, fmt.Sprintf("<@%d> — %s (%s) [Stage %d]",
			a.DiscordID, name, nvl(a.Agency, "N/A"), a.CurrentStage))
	}

	title := "All Agents"
	if stage > 0 {
		title = fmt.Sprintf("Agents at Stage %d — %s", stage, stageLabel(stage))
	}

	content := fmt.Sprintf("**%s** (%d total)\n\n%s", title, len(agents), strings.Join(lines, "\n"))
	if len(agents) > limit {
		content += fmt.Sprintf("\n\n...and %d more", len(agents)-limit)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
