package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleContractingList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	managers, err := b.db.GetContractingManagers(context.Background())
	if err != nil {
		log.Printf("admin handler: handleContractingList: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An error occurred. Please try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(managers) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No active contracting managers.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var lines []string
	for _, m := range managers {
		lines = append(lines, fmt.Sprintf("**%s** (priority %d) — %s", m.ManagerName, m.Priority, m.CalendlyURL))
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "**Active Contracting Managers:**\n\n" + strings.Join(lines, "\n"),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
