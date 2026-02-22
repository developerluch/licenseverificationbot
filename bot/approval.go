package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
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
		// Fall through to sortAndAssign directly
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
		// Fall through
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

	// DM the agency owner
	embed := &discordgo.MessageEmbed{
		Title: "New Agent Approval Request",
		Description: fmt.Sprintf(
			"A new agent has completed onboarding and needs your approval.\n\n"+
				"**Agent:** <@%s> (%s)\n"+
				"**Agency:** %s\n"+
				"**License Status:** %s\n\n"+
				"Please approve or deny this agent.",
			userID, fullName, agency, licenseStatus),
		Color:     0x3498DB,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	channel, err := s.UserChannelCreate(ownerID)
	if err != nil {
		log.Printf("Approval: failed to DM owner %s: %v", ownerID, err)
		// Fall back to admin channel
		b.postApprovalToChannel(s, embed, reqID)
		return
	}

	msg, err := s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Approve",
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("vipa:approve:%d", reqID),
					},
					discordgo.Button{
						Label:    "Deny",
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("vipa:deny:%d", reqID),
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Approval: failed to send DM to owner %s: %v", ownerID, err)
		b.postApprovalToChannel(s, embed, reqID)
		return
	}

	// Store message ID for editing later
	b.db.UpdateApprovalDMMessageID(ctx, reqID, msg.ID)
	log.Printf("Approval: sent request #%d to owner %s for agent %s", reqID, ownerID, userID)
}

// postApprovalToChannel posts the approval request to admin channel as fallback.
func (b *Bot) postApprovalToChannel(s *discordgo.Session, embed *discordgo.MessageEmbed, reqID int) {
	channelID := b.cfg.AdminNotificationChannelID
	if channelID == "" {
		channelID = b.cfg.HiringLogChannelID
	}
	if channelID == "" {
		return
	}
	s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Approve",
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("vipa:approve:%d", reqID),
					},
					discordgo.Button{
						Label:    "Deny",
						Style:    discordgo.DangerButton,
						CustomID: fmt.Sprintf("vipa:deny:%d", reqID),
					},
				},
			},
		},
	})
}

// handleApprovalButton handles Approve/Deny button clicks.
func (b *Bot) handleApprovalButton(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return
	}

	action := parts[1] // "approve" or "deny"
	reqID, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := b.db.GetApprovalRequest(ctx, reqID)
	if err != nil || req == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Approval request not found.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if req.Status != "pending" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("This request has already been %s.", req.Status),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Authorization: only the designated owner or staff can approve/deny
	responderID := ""
	if i.Member != nil {
		responderID = i.Member.User.ID
	} else if i.User != nil {
		responderID = i.User.ID
	}
	ownerIDStr := strconv.FormatInt(req.OwnerDiscordID, 10)
	if responderID != ownerIDStr && (i.Member == nil || !b.cfg.IsStaff(i.Member.Roles)) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You are not authorized to respond to this approval request.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	agentUserID := strconv.FormatInt(req.AgentDiscordID, 10)
	guildID := strconv.FormatInt(req.GuildID, 10)

	switch action {
	case "approve":
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

	case "deny":
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
}

// handleDenyReasonModal processes the denial reason modal.
func (b *Bot) handleDenyReasonModal(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID
	parts := strings.Split(customID, ":")
	if len(parts) < 3 {
		return
	}
	reqID, err := strconv.Atoi(parts[2])
	if err != nil {
		return
	}

	reason := getModalValue(i.ModalSubmitData(), "reason")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := b.db.GetApprovalRequest(ctx, reqID)
	if err != nil || req == nil || req.Status != "pending" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Request already processed.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	b.db.DenyAgent(ctx, reqID, reason)

	deniedStatus := "denied"
	b.db.UpsertAgent(ctx, req.AgentDiscordID, req.GuildID, db.AgentUpdate{
		ApprovalStatus: &deniedStatus,
	})

	agentUserID := strconv.FormatInt(req.AgentDiscordID, 10)
	guildID := strconv.FormatInt(req.GuildID, 10)

	// Remove pending role
	if b.cfg.PendingRoleID != "" {
		s.GuildMemberRoleRemove(guildID, agentUserID, b.cfg.PendingRoleID)
	}

	// DM the agent
	msg := "Your onboarding request has been reviewed by your agency owner and was not approved at this time."
	if reason != "" {
		msg += fmt.Sprintf("\n\n**Reason:** %s", reason)
	}
	msg += "\n\nPlease contact your upline for more information."
	b.dmUser(s, agentUserID, msg)

	// Respond
	desc := fmt.Sprintf("<@%s> has been denied for **%s**.", agentUserID, req.Agency)
	if reason != "" {
		desc += fmt.Sprintf("\n**Reason:** %s", reason)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{
				Title:       "Agent Denied",
				Description: desc,
				Color:       0xE74C3C,
				Timestamp:   time.Now().Format(time.RFC3339),
			}},
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	log.Printf("Approval: request #%d denied (reason: %s)", reqID, reason)
}
