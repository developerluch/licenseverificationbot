package bot

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleNPNLookup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Defer failed: %v", err)
		return
	}

	opts := i.ApplicationCommandData().Options
	optMap := make(map[string]string)
	for _, opt := range opts {
		optMap[opt.Name] = opt.StringValue()
	}

	firstName := strings.TrimSpace(optMap["first_name"])
	lastName := strings.TrimSpace(optMap["last_name"])
	state := strings.ToUpper(strings.TrimSpace(optMap["state"]))

	if firstName == "" || lastName == "" {
		b.followUp(s, i, "Please provide both first and last name.\nExample: `/npn first_name:John last_name:Doe`")
		return
	}

	// If a specific state is given, search just that state
	if state != "" && len(state) == 2 {
		b.npnSingleState(s, i, firstName, lastName, state)
		return
	}

	// Otherwise search across all NAIC states in parallel
	b.npnMultiState(s, i, firstName, lastName)
}
