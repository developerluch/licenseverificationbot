package bot

import (
	"github.com/bwmarrin/discordgo"
)

// isStaff checks if the interaction user has a staff role.
func (b *Bot) isStaff(i *discordgo.InteractionCreate) bool {
	if i.Member == nil {
		return false
	}
	return b.cfg.IsStaff(i.Member.Roles)
}

// respondEphemeral sends an ephemeral message response to an interaction.
func respondEphemeral(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
