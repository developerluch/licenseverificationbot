package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) assignRoles(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("assignRoles panic: %v", r)
		}
	}()

	if b.cfg.LicensedAgentRoleID != "" {
		err := s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, b.cfg.LicensedAgentRoleID)
		if err != nil {
			log.Printf("Failed to add Licensed Agent role: %v", err)
		}
	}

	if b.cfg.StudentRoleID != "" {
		err := s.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, b.cfg.StudentRoleID)
		if err != nil {
			log.Printf("Failed to remove Student role: %v", err)
		}
	}
}
