package bot

import "github.com/bwmarrin/discordgo"

// zoomCommands returns zoom training vertical commands.
func zoomCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "zoom",
			Description: "Manage zoom training verticals",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List available zoom verticals",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "join",
					Description: "Join a zoom vertical",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "id", Description: "Vertical ID", Required: true},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "leave",
					Description: "Leave a zoom vertical",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "id", Description: "Vertical ID", Required: true},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "create",
					Description: "Create a new zoom vertical (Staff only)",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionString, Name: "name", Description: "Vertical name", Required: true},
						{Type: discordgo.ApplicationCommandOptionString, Name: "description", Description: "Description", Required: false},
						{Type: discordgo.ApplicationCommandOptionString, Name: "zoom_link", Description: "Zoom meeting link", Required: false},
						{Type: discordgo.ApplicationCommandOptionString, Name: "schedule", Description: "Meeting schedule (e.g. 'Mon/Wed 7pm ET')", Required: false},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "delete",
					Description: "Delete a zoom vertical (Staff only)",
					Options: []*discordgo.ApplicationCommandOption{
						{Type: discordgo.ApplicationCommandOptionInteger, Name: "id", Description: "Vertical ID to delete", Required: true},
					},
				},
			},
		},
	}
}
