package bot

import "github.com/bwmarrin/discordgo"

// miscCommands returns miscellaneous setup and onboarding commands.
func miscCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "start",
			Description: "Restart onboarding if you missed the welcome message",
		},
		{
			Name:        "onboarding-setup",
			Description: "Post the Get Started panel in #start-here (Admin only)",
		},
		{
			Name:        "setup-rules",
			Description: "Post the rules embed in #rules (Admin only)",
		},
	}
}
