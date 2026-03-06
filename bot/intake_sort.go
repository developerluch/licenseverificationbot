package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// sortAndAssignRoles assigns @Student + agency role + @Onboarded, and optionally @Licensed-Agent.
func (b *Bot) sortAndAssignRoles(s *discordgo.Session, userID, guildID, agency, licenseStatus string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("sortAndAssignRoles panic: %v", r)
		}
	}()

	// Remove @New role (join-gate) and assign @Onboarded role (unlock all channels)
	if b.cfg.NewRoleID != "" {
		if err := s.GuildMemberRoleRemove(guildID, userID, b.cfg.NewRoleID); err != nil {
			log.Printf("Intake: failed to remove @New role from %s: %v", userID, err)
		}
	}
	if b.cfg.OnboardedRoleID != "" {
		if err := s.GuildMemberRoleAdd(guildID, userID, b.cfg.OnboardedRoleID); err != nil {
			log.Printf("Intake: failed to add Onboarded role to %s: %v", userID, err)
		}
	}

	// Always assign Student role first
	if b.cfg.StudentRoleID != "" {
		if err := s.GuildMemberRoleAdd(guildID, userID, b.cfg.StudentRoleID); err != nil {
			log.Printf("Intake: failed to add Student role to %s: %v", userID, err)
		}
	}

	// Assign agency role
	agencyRoleID := b.cfg.GetAgencyRoleID(agency)
	if agencyRoleID != "" {
		if err := s.GuildMemberRoleAdd(guildID, userID, agencyRoleID); err != nil {
			log.Printf("Intake: failed to add agency role %s to %s: %v", agencyRoleID, userID, err)
		}
	}

	// If licensed, also assign Licensed-Agent and remove Student
	if licenseStatus == "licensed" {
		if b.cfg.LicensedAgentRoleID != "" {
			if err := s.GuildMemberRoleAdd(guildID, userID, b.cfg.LicensedAgentRoleID); err != nil {
				log.Printf("Intake: failed to add Licensed-Agent role to %s: %v", userID, err)
			}
		}
		if b.cfg.StudentRoleID != "" {
			if err := s.GuildMemberRoleRemove(guildID, userID, b.cfg.StudentRoleID); err != nil {
				log.Printf("Intake: failed to remove Student role from %s: %v", userID, err)
			}
		}
	}
}
