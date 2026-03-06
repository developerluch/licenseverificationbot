package bot

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleContract shows contracting manager Calendly links.
func (b *Bot) handleContract(s *discordgo.Session, i *discordgo.InteractionCreate) {
	managers, err := b.db.GetContractingManagers(context.Background())
	if err != nil {
		log.Printf("Contract: failed to get managers: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to load contracting managers. Try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(managers) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No contracting managers are currently available. Contact your upline for help.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := buildContractingEmbed(managers)

	// Mark contracting_booked in DB
	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleContract: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleContract: %v", err)
		return
	}
	booked := true
	stage := db.StageContracting
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		ContractingBooked: &booked,
		CurrentStage:      &stage,
	})
	b.db.LogActivity(context.Background(), userIDInt, "contracting_viewed", "Viewed contracting managers")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}
