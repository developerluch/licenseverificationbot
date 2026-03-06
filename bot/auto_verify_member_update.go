package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// handleMemberUpdate fires when a member's roles change.
// We watch for the onboarding bot adding @Licensed-Agent or @Student roles.
func (b *Bot) handleMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("handleMemberUpdate panic: %v", r)
		}
	}()

	if e.BeforeUpdate == nil {
		return // Can't compare without before state
	}

	// Check if rules screening was completed (pending -> not pending)
	if e.BeforeUpdate.Pending && !e.Pending {
		go b.handleRulesScreeningComplete(s, e.Member)
		return
	}

	newRoles := roleSet(e.Roles)
	oldRoles := roleSet(e.BeforeUpdate.Roles)

	// Check if @Licensed-Agent role was just added
	if b.cfg.LicensedAgentRoleID != "" {
		if newRoles[b.cfg.LicensedAgentRoleID] && !oldRoles[b.cfg.LicensedAgentRoleID] {
			go b.onLicensedRoleAdded(s, e)
			return
		}
	}

	// Check if @Student role was just added (unlicensed path)
	if b.cfg.StudentRoleID != "" {
		if newRoles[b.cfg.StudentRoleID] && !oldRoles[b.cfg.StudentRoleID] {
			go b.onStudentRoleAdded(s, e)
			return
		}
	}
}
