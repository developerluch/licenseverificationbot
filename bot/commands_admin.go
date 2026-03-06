package bot

import "github.com/bwmarrin/discordgo"

// adminCommands returns admin-only management commands.
func adminCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "contracting",
			Description: "Manage contracting managers (Admin only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a contracting manager",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Manager name", Required: true},
						{Type: discordgo.ApplicationCommandOptionString, Name: "url", Description: "Calendly URL", Required: true},
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "priority", Description: "Priority (lower = higher)", Required: false},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Deactivate a contracting manager",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Manager name to remove", Required: true},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List active contracting managers",
				},
			},
		},
		{
			Name:        "tracker",
			Description: "License verification tracker (Staff only)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "overview",
					Description: "Overall license verification progress",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "agency",
					Description: "License verification progress by agency",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "recruiter",
					Description: "License verification progress by recruiter within an agency",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionString, Name: "agency", Description: "Agency name (e.g. TFC, Radiant)", Required: true},
					},
				},
			},
		},
		{
			Name:        "role-audit",
			Description: "Audit role conflicts across all members (Staff only)",
		},
	}
}
