package bot

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func (b *Bot) handleVerify(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command can only be used in a server.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Step 1: Defer (ephemeral)
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

	// Step 2: Extract options
	opts := i.ApplicationCommandData().Options
	optMap := make(map[string]string)
	for _, opt := range opts {
		optMap[opt.Name] = opt.StringValue()
	}

	firstName := optMap["first_name"]
	lastName := optMap["last_name"]
	state := strings.ToUpper(strings.TrimSpace(optMap["state"]))
	phone := optMap["phone"]

	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}

	log.Printf("License verify: %s %s (%s) by %s", firstName, lastName, state, userID)

	// Step 3: Pull state from DB if not provided
	if state == "" {
		agent, err := b.db.GetAgent(context.Background(), userIDInt)
		if err == nil && agent != nil {
			state = agent.State
		}
	}

	if state == "" || len(state) != 2 {
		b.followUp(s, i, "Please provide your 2-letter state code.\nExample: `/verify first_name:John last_name:Doe state:FL`")
		return
	}

	// Step 4: Save phone number if provided
	if phone != "" {
		cleanPhone := cleanPhoneNumber(phone)
		if cleanPhone != "" {
			b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
				PhoneNumber: &cleanPhone,
			})
		}
	}

	// Step 5: Perform verification lookup
	b.performVerifyLookup(s, i, firstName, lastName, state, userID, userIDInt, guildIDInt)
}
