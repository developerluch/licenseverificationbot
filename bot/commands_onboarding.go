package bot

import "github.com/bwmarrin/discordgo"

// onboardingCommands returns onboarding-related slash commands.
func onboardingCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "contract",
			Description: "Book a contracting appointment with a manager",
		},
		{
			Name:        "setup",
			Description: "View or manage your agent setup checklist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "action",
					Description: "start to see your checklist, complete to finish setup",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "start", Value: "start"},
						{Name: "complete", Value: "complete"},
					},
				},
			},
		},
	}
}
