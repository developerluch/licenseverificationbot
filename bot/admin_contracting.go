package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// handleContractingCommand routes contracting subcommands.
func (b *Bot) handleContractingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.cfg.IsStaff(i.Member.Roles) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is restricted to staff.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "add":
		b.handleContractingAdd(s, i, sub.Options)
	case "remove":
		b.handleContractingRemove(s, i, sub.Options)
	case "list":
		b.handleContractingList(s, i)
	}
}
func (b *Bot) handleContractingAdd(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var name, url string
	priority := 1
	for _, opt := range opts {
		switch opt.Name {
		case "name":
			name = opt.StringValue()
		case "url":
			url = opt.StringValue()
		case "priority":
			priority = int(opt.IntValue())
		}
	}

	if url != "" && !strings.HasPrefix(url, "https://calendly.com/") {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "URL must be a valid Calendly link (https://calendly.com/...)",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	err := b.db.AddContractingManager(context.Background(), name, url, priority)
	msg := fmt.Sprintf("Added contracting manager **%s** (priority %d).", name, priority)
	if err != nil {
		log.Printf("admin handler: handleContractingAdd: %v", err)
		msg = "An error occurred. Please try again later."
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleContractingRemove(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(opts) == 0 {
		return
	}
	name := opts[0].StringValue()

	err := b.db.DeactivateContractingManager(context.Background(), name)
	msg := fmt.Sprintf("Deactivated contracting manager **%s**.", name)
	if err != nil {
		log.Printf("admin handler: handleContractingRemove: %v", err)
		msg = "An error occurred. Please try again later."
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}