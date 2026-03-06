package bot

import (
	"github.com/bwmarrin/discordgo"
)

// handleTrackerCommand routes /tracker subcommands (Staff only).
func (b *Bot) handleTrackerCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Defer ephemeral response
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		b.followUp(s, i, "Please specify a subcommand: overview, agency, or recruiter.")
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "overview":
		b.handleTrackerOverview(s, i)
	case "agency":
		b.handleTrackerAgency(s, i)
	case "recruiter":
		b.handleTrackerRecruiter(s, i, sub.Options)
	}
}
