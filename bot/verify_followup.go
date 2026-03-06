package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) followUp(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("Follow-up failed: %v", err)
	}
}

// verifyLogChannelID returns the channel ID for posting verification results.
// Falls back: LicenseVerifyLogChannelID -> LicenseCheckChannelID -> HiringLogChannelID.
func (b *Bot) verifyLogChannelID() string {
	if b.cfg.LicenseVerifyLogChannelID != "" {
		return b.cfg.LicenseVerifyLogChannelID
	}
	if b.cfg.LicenseCheckChannelID != "" {
		return b.cfg.LicenseCheckChannelID
	}
	return b.cfg.HiringLogChannelID
}
