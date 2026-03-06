package bot

import "github.com/bwmarrin/discordgo"

// verifyCommands returns verification-related slash commands.
func verifyCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "verify",
			Description: "Verify your insurance license and get promoted to Licensed Agent",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "first_name", Description: "Your legal first name (as on your license)", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "last_name", Description: "Your legal last name (as on your license)", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "state", Description: "Your home state (2-letter code, e.g. FL, TX, CA)", Required: false},
				{Type: discordgo.ApplicationCommandOptionString, Name: "phone", Description: "Your phone number for license update texts (optional)", Required: false},
			},
		},
		{Name: "license-history", Description: "View your license check history"},
		{
			Name:        "email-optin",
			Description: "Opt in to receive email notifications about your license verification",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "email", Description: "Your email address", Required: true},
			},
		},
		{Name: "email-optout", Description: "Stop receiving email notifications"},
		{
			Name:        "npn",
			Description: "Look up an agent's NPN (National Producer Number) by name",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionString, Name: "first_name", Description: "Agent's first name", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "last_name", Description: "Agent's last name", Required: true},
				{Type: discordgo.ApplicationCommandOptionString, Name: "state", Description: "State to search (2-letter code). Leave blank to search all 31+ states", Required: false},
			},
		},
	}
}
