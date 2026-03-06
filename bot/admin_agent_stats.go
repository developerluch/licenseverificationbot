package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleAgentStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	counts, err := b.db.GetAgentCounts(context.Background())
	if err != nil {
		log.Printf("admin handler: handleAgentStats: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An error occurred. Please try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	kicked, _ := b.db.GetKickedCount(context.Background())
	total := 0
	var lines []string
	for stage := 1; stage <= 8; stage++ {
		count := counts[stage]
		total += count
		barLen := count
		if barLen > 30 {
			barLen = 30
		}
		bar := strings.Repeat("█", barLen)
		lines = append(lines, fmt.Sprintf("`%d` %s %s (%d)",
			stage, stageLabel(stage), bar, count))
	}

	content := fmt.Sprintf("**Onboarding Dashboard**\nTotal active: %d | Kicked: %d\n\n%s",
		total, kicked, strings.Join(lines, "\n"))

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
