package bot

import (
	"github.com/bwmarrin/discordgo"
)

// handleZoomCommand routes /zoom subcommands.
func (b *Bot) handleZoomCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		b.followUp(s, i, "Please specify a subcommand: list, join, leave, create, or delete.")
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "list":
		b.handleZoomList(s, i)
	case "join":
		b.handleZoomJoin(s, i, sub.Options)
	case "leave":
		b.handleZoomLeave(s, i, sub.Options)
	case "create":
		if !b.cfg.IsStaff(i.Member.Roles) {
			b.followUp(s, i, "This subcommand is restricted to staff.")
			return
		}
		b.handleZoomCreate(s, i, sub.Options)
	case "delete":
		if !b.cfg.IsStaff(i.Member.Roles) {
			b.followUp(s, i, "This subcommand is restricted to staff.")
			return
		}
		b.handleZoomDelete(s, i, sub.Options)
	}
}
