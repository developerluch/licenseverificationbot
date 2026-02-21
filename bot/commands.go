package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// registerCommands registers all slash commands with Discord.
func (b *Bot) registerCommands() {
	commands := []*discordgo.ApplicationCommand{
		// === Existing commands (DO NOT TOUCH definitions) ===
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

		// === Onboarding commands ===
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

		// === Staff commands ===
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
			},
		},

		// === Admin commands ===
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
			Name:        "onboarding-setup",
			Description: "Post the Get Started panel in #start-here (Admin only)",
		},
		{
			Name:        "setup-rules",
			Description: "Post the rules embed in #rules (Admin only)",
		},
	}

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.cfg.GuildID, cmd)
		if err != nil {
			log.Printf("Cannot register command %s: %v", cmd.Name, err)
		}
	}

	log.Printf("Slash commands registered for guild %s", b.cfg.GuildID)
}
