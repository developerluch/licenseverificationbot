package bot

import (
	"context"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// triggerApprovalFlow sends an approval DM to the agency owner.
func (b *Bot) triggerApprovalFlow(s *discordgo.Session, userID, guildID, agency, fullName, licenseStatus, ownerID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in triggerApprovalFlow for %s: %v", userID, r)
			b.sortAndAssignRoles(s, userID, guildID, agency, licenseStatus)
		}
	}()

	agentIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("Approval: invalid agent ID %s: %v", userID, err)
		return
	}
	guildIDInt, err := parseDiscordID(guildID)
	if err != nil {
		return
	}
	ownerIDInt, err := parseDiscordID(ownerID)
	if err != nil {
		log.Printf("WARNING: Approval: invalid owner ID %s, bypassing approval: %v", ownerID, err)
		b.sortAndAssignRoles(s, userID, guildID, agency, licenseStatus)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create DB record
	reqID, err := b.db.CreateApprovalRequest(ctx, db.ApprovalRequest{
		AgentDiscordID: agentIDInt,
		GuildID:        guildIDInt,
		Agency:         agency,
		OwnerDiscordID: ownerIDInt,
	})
	if err != nil {
		log.Printf("Approval: failed to create request: %v", err)
		b.sortAndAssignRoles(s, userID, guildID, agency, licenseStatus)
		return
	}

	// Update agent approval status
	status := "pending"
	b.db.UpsertAgent(ctx, agentIDInt, guildIDInt, db.AgentUpdate{
		ApprovalStatus: &status,
	})

	// Assign pending role if configured
	if b.cfg.PendingRoleID != "" {
		s.GuildMemberRoleAdd(guildID, userID, b.cfg.PendingRoleID)
	}

	// Send approval message to owner
	b.sendApprovalDM(s, ownerID, userID, fullName, agency, licenseStatus, reqID, ctx)
}
