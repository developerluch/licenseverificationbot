package bot

import "github.com/bwmarrin/discordgo"

// activityCommands returns activity logging and leaderboard commands.
func activityCommands() []*discordgo.ApplicationCommand {
	minZero := float64(0)
	return []*discordgo.ApplicationCommand{
		{
			Name:        "log",
			Description: "Log your daily activity (calls, appointments, etc.)",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "calls", Description: "Number of calls made", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "appointments", Description: "Number of appointments set", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "presentations", Description: "Number of presentations given", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "policies", Description: "Number of policies written", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionInteger, Name: "recruits", Description: "Number of recruits signed", Required: false, MinValue: &minZero},
				{Type: discordgo.ApplicationCommandOptionString, Name: "notes", Description: "Optional notes", Required: false},
			},
		},
		{
			Name:        "leaderboard",
			Description: "View activity leaderboard",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "weekly",
					Description: "This week's leaderboard",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "type",
							Description: "Activity type to rank by",
							Required:    false,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "All", Value: "all"},
								{Name: "Calls", Value: "calls"},
								{Name: "Appointments", Value: "appointments"},
								{Name: "Presentations", Value: "presentations"},
								{Name: "Policies", Value: "policies"},
								{Name: "Recruits", Value: "recruits"},
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "monthly",
					Description: "This month's leaderboard",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "type",
							Description: "Activity type to rank by",
							Required:    false,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "All", Value: "all"},
								{Name: "Calls", Value: "calls"},
								{Name: "Appointments", Value: "appointments"},
								{Name: "Presentations", Value: "presentations"},
								{Name: "Policies", Value: "policies"},
								{Name: "Recruits", Value: "recruits"},
							},
						},
					},
				},
			},
		},
	}
}
