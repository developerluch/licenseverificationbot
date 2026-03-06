package bot

import "github.com/bwmarrin/discordgo"

// wavvCommands returns WAVV production tracker slash commands.
func wavvCommands() []*discordgo.ApplicationCommand {
	minZero := float64(0)
	return []*discordgo.ApplicationCommand{
		{
			Name:        "wavv-log",
			Description: "Log a WAVV dialing session",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "dials", Description: "Number of dials made", Required: true, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "connections", Description: "Number of live connections", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "talk-mins", Description: "Total talk time in minutes", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "appointments", Description: "Appointments set", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "callbacks", Description: "Callbacks scheduled", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "policies", Description: "Policies written", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionString, Name: "notes", Description: "Session notes", Required: false},
			},
		},
		{
			Name:        "wavv-stats",
			Description: "View your WAVV production stats",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "period",
					Description: "Time period to view",
					Required:    false,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "This Week", Value: "week"},
						{Name: "This Month", Value: "month"},
						{Name: "Last 7 Days", Value: "7d"},
						{Name: "Last 30 Days", Value: "30d"},
					},
				},
			},
		},
	}
}
