package bot

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func (b *Bot) auditAgentRoles(s *discordgo.Session, agents []db.AgentRow) []string {
	var conflicts []string
	for _, agent := range agents {
		userID := strconv.FormatInt(agent.DiscordID, 10)
		guildID := strconv.FormatInt(agent.GuildID, 10)

		member, err := s.GuildMember(guildID, userID)
		if err != nil {
			continue // Member left server
		}

		hasStudent := roleInList(member.Roles, b.cfg.StudentRoleID)
		hasLicensed := roleInList(member.Roles, b.cfg.LicensedAgentRoleID)
		hasActive := roleInList(member.Roles, b.cfg.ActiveAgentRoleID)

		// Conflict: both Student + Licensed
		if hasStudent && hasLicensed {
			conflicts = append(conflicts, fmt.Sprintf("<@%s>: has both Student + Licensed roles", userID))
		}

		// DB says verified but no Licensed role
		if agent.LicenseVerified && !hasLicensed && !hasActive {
			conflicts = append(conflicts, fmt.Sprintf("<@%s>: DB verified but missing Licensed/Active role", userID))
		}

		// Has Active but DB stage < 8
		if hasActive && agent.CurrentStage < db.StageActive {
			conflicts = append(conflicts, fmt.Sprintf("<@%s>: has Active role but stage=%d", userID, agent.CurrentStage))
		}
	}
	return conflicts
}
