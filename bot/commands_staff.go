package bot

import "github.com/bwmarrin/discordgo"

// staffCommands returns staff-only management commands.
func staffCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "restart",
			Description: "Reopen onboarding form for a user (Staff only)",
			Options: []*discordgo.ApplicationCommandOption{
				{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "The user to restart onboarding for", Required: true},
			},
		},
		{
			Name:        "agent",
			Description: "Agent management commands (Staff only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "info",
					Description: "View full agent profile",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "The agent to look up", Required: true},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List agents by stage",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "stage", Description: "Stage number (1-8)", Required: false},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "nudge",
					Description: "Send a manual check-in DM to an agent",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "The agent to nudge", Required: true},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "promote",
					Description: "Manually promote an agent",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "The agent to promote", Required: true},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "level",
							Description: "Promotion level",
							Required:    true,
							Choices: []*discordgo.ApplicationCommandOptionChoice{
								{Name: "licensed", Value: "licensed"},
								{Name: "active", Value: "active"},
							},
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "stats",
					Description: "Show onboarding dashboard with agent counts per stage",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "kick",
					Description: "Remove an agent from the server",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionUser, Name: "user", Description: "The agent to kick", Required: true},
						{Type: discordgo.ApplicationCommandOptionString, Name: "reason", Description: "Reason for removal", Required: false},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "assign-manager",
					Description: "Assign a direct manager to an agent",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionUser, Name: "agent", Description: "The agent", Required: true},
						{Type: discordgo.ApplicationCommandOptionUser, Name: "manager", Description: "The manager to assign", Required: true},
					},
				},
			},
		},
	}
}
