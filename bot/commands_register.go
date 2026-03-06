package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// registerCommands registers all slash commands with Discord.
func (b *Bot) registerCommands() {
	commands := []*discordgo.ApplicationCommand{}

	// Collect all commands from specialized functions
	commands = append(commands, verifyCommands()...)
	commands = append(commands, onboardingCommands()...)
	commands = append(commands, staffCommands()...)
	commands = append(commands, activityCommands()...)
	commands = append(commands, zoomCommands()...)
	commands = append(commands, ticketsCommands()...)
	commands = append(commands, wavvCommands()...)
	commands = append(commands, miscCommands()...)

	// Register all commands with Discord
	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.cfg.GuildID, cmd)
		if err != nil {
			log.Printf("Cannot register command %s: %v", cmd.Name, err)
		}
	}

	log.Printf("Slash commands registered for guild %s", b.cfg.GuildID)
}
