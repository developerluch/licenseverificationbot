package bot

import "github.com/bwmarrin/discordgo"

// ticketsCommands returns ticket panel setup commands.
func ticketsCommands() []*discordgo.ApplicationCommand {
	return []*discordgo.ApplicationCommand{
		{
			Name:        "ticket-setup",
			Description: "Post the general support ticket panel (Staff only)",
		},
		{
			Name:        "wavv-ticket-setup",
			Description: "Post the WAVV support ticket panel (Staff only)",
		},
	}
}
