package bot

import (
	"github.com/bwmarrin/discordgo"
)

// triggerStep2AsyncOperations triggers role assignment, GHL sync, and post-boarding messages.
func (b *Bot) triggerStep2AsyncOperations(s *discordgo.Session, i *discordgo.InteractionCreate, step1 *ModalTempData, userID string, userIDInt, guildIDInt int64, stage int, agentData map[string]string) {
	// GHL CRM sync (fire-and-forget)
	go b.syncAgentToGHL(userIDInt, guildIDInt)
	go b.syncGHLStage(userIDInt, stage)

	// Resolve upline manager's Discord ID (fire-and-forget)
	go b.resolveUplineDiscordID(s, i.GuildID, step1.UplineManager, userIDInt, guildIDInt)

	// Assign roles — gate through approval if agency owner is configured
	ownerID := b.cfg.GetAgencyOwnerID(step1.Agency)
	if ownerID != "" {
		go b.triggerApprovalFlow(s, userID, i.GuildID, step1.Agency, step1.FullName, step1.LicenseStatus, ownerID)
	} else {
		go b.sortAndAssignRoles(s, userID, i.GuildID, step1.Agency, step1.LicenseStatus)
	}

	// Post hiring log and greetings card
	go b.postHiringLog(s, i.Member, agentData)
	go b.postGreetingsCard(s, i.Member, agentData)
}
