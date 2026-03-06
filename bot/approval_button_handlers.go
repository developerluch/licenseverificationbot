package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func (b *Bot) handleApprovalApprove(s *discordgo.Session, i *discordgo.InteractionCreate,
	ctx context.Context, reqID int, req *db.ApprovalRequest, agentUserID, guildID string) {

	b.db.ApproveAgent(ctx, reqID)

	// Update agent approval status
	approvedStatus := "approved"
	b.db.UpsertAgent(ctx, req.AgentDiscordID, req.GuildID, db.AgentUpdate{
		ApprovalStatus: &approvedStatus,
	})

	// Get the agent to determine license status
	agent, _ := b.db.GetAgent(ctx, req.AgentDiscordID)
	licenseStatus := "none"
	if agent != nil {
		licenseStatus = agent.LicenseStatus
	}

	// Remove pending role, run sort
	if b.cfg.PendingRoleID != "" {
		s.GuildMemberRoleRemove(guildID, agentUserID, b.cfg.PendingRoleID)
	}
	go b.sortAndAssignRoles(s, agentUserID, guildID, req.Agency, licenseStatus)

	// DM the agent
	b.dmUser(s, agentUserID, fmt.Sprintf(
		"**Great news!** Your agency owner has approved your onboarding for **%s**.\n\n"+
			"You're all set! Check the server for your next steps.", req.Agency))

	// Respond to owner
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{
				Title:       "Agent Approved",
				Description: fmt.Sprintf("<@%s> has been approved for **%s**.", agentUserID, req.Agency),
				Color:       0x2ECC71,
				Timestamp:   time.Now().Format(time.RFC3339),
			}},
			Components: []discordgo.MessageComponent{}, // Remove buttons
		},
	})

	responderID := ""
	if i.Member != nil {
		responderID = i.Member.User.ID
	} else if i.User != nil {
		responderID = i.User.ID
	}
	log.Printf("Approval: request #%d approved by %s", reqID, responderID)
}

func (b *Bot) handleApprovalDeny(s *discordgo.Session, i *discordgo.InteractionCreate, reqID int) {
	// Show modal for denial reason
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: fmt.Sprintf("vipa:deny_reason:%d", reqID),
			Title:    "Deny Agent — Reason (Optional)",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "reason",
						Label:       "Reason for denial",
						Style:       discordgo.TextInputParagraph,
						Placeholder: "Optional — leave blank if no reason",
						Required:    false,
						MaxLength:   500,
					},
				}},
			},
		},
	})
}
