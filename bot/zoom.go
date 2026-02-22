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

// handleZoomCommand routes /zoom subcommands.
func (b *Bot) handleZoomCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command can only be used in a server.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		b.followUp(s, i, "Please specify a subcommand: list, join, leave, create, or delete.")
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "list":
		b.handleZoomList(s, i)
	case "join":
		b.handleZoomJoin(s, i, sub.Options)
	case "leave":
		b.handleZoomLeave(s, i, sub.Options)
	case "create":
		if !b.cfg.IsStaff(i.Member.Roles) {
			b.followUp(s, i, "This subcommand is restricted to staff.")
			return
		}
		b.handleZoomCreate(s, i, sub.Options)
	case "delete":
		if !b.cfg.IsStaff(i.Member.Roles) {
			b.followUp(s, i, "This subcommand is restricted to staff.")
			return
		}
		b.handleZoomDelete(s, i, sub.Options)
	}
}

func (b *Bot) handleZoomList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	verticals, err := b.db.GetZoomVerticals(ctx)
	if err != nil {
		log.Printf("zoom list: %v", err)
		b.followUp(s, i, "An error occurred. Please try again later.")
		return
	}

	if len(verticals) == 0 {
		b.followUp(s, i, "No zoom verticals available.")
		return
	}

	// Get user's joined verticals
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	}
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("zoom list: invalid user ID %s: %v", userID, err)
	}
	myVerticals, _ := b.db.GetUserZoomVerticals(ctx, userIDInt)
	mySet := make(map[int]bool)
	for _, v := range myVerticals {
		mySet[v.ID] = true
	}

	var lines []string
	for _, v := range verticals {
		joined := ""
		if mySet[v.ID] {
			joined = " [JOINED]"
		}
		line := fmt.Sprintf("**%d. %s**%s", v.ID, v.Name, joined)
		if v.Description != "" {
			line += "\n   " + v.Description
		}
		if v.Schedule != "" {
			line += "\n   Schedule: " + v.Schedule
		}
		lines = append(lines, line)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Zoom Verticals",
		Description: strings.Join(lines, "\n\n"),
		Color:       0x3498DB,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /zoom join id:<number> to join a vertical",
		},
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

func (b *Bot) handleZoomJoin(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var verticalID int
	for _, opt := range opts {
		if opt.Name == "id" {
			verticalID = int(opt.IntValue())
		}
	}

	if verticalID == 0 {
		b.followUp(s, i, "Please specify a vertical ID. Use `/zoom list` to see available verticals.")
		return
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	}
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v, err := b.db.GetZoomVertical(ctx, verticalID)
	if err != nil || v == nil {
		b.followUp(s, i, "Vertical not found.")
		return
	}

	if err := b.db.JoinZoomVertical(ctx, userIDInt, verticalID); err != nil {
		log.Printf("zoom join: %v", err)
		b.followUp(s, i, "An error occurred joining the vertical. Please try again later.")
		return
	}

	msg := fmt.Sprintf("You've joined **%s**!", v.Name)
	if v.ZoomLink != "" {
		msg += fmt.Sprintf("\n\nZoom Link: %s", v.ZoomLink)
	}
	if v.Schedule != "" {
		msg += fmt.Sprintf("\nSchedule: %s", v.Schedule)
	}
	b.followUp(s, i, msg)
}

func (b *Bot) handleZoomLeave(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var verticalID int
	for _, opt := range opts {
		if opt.Name == "id" {
			verticalID = int(opt.IntValue())
		}
	}

	if verticalID == 0 {
		b.followUp(s, i, "Please specify a vertical ID.")
		return
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	}
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	b.db.LeaveZoomVertical(ctx, userIDInt, verticalID)
	b.followUp(s, i, "You've left the vertical.")
}

func (b *Bot) handleZoomCreate(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var name, description, zoomLink, schedule string
	for _, opt := range opts {
		switch opt.Name {
		case "name":
			name = opt.StringValue()
		case "description":
			description = opt.StringValue()
		case "zoom_link":
			zoomLink = opt.StringValue()
		case "schedule":
			schedule = opt.StringValue()
		}
	}

	if name == "" {
		b.followUp(s, i, "Name is required.")
		return
	}

	if zoomLink != "" && !strings.HasPrefix(zoomLink, "https://") {
		b.followUp(s, i, "Zoom link must start with https://")
		return
	}

	creatorID := ""
	if i.Member != nil {
		creatorID = i.Member.User.ID
	}
	creatorIDInt, err := parseDiscordID(creatorID)
	if err != nil {
		log.Printf("zoom create: invalid creator ID %s: %v", creatorID, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := b.db.CreateZoomVertical(ctx, db.ZoomVertical{
		Name:        name,
		Description: description,
		ZoomLink:    zoomLink,
		Schedule:    schedule,
		CreatedBy:   creatorIDInt,
	})
	if err != nil {
		log.Printf("zoom create: %v", err)
		b.followUp(s, i, "An error occurred creating the vertical. Please try again later.")
		return
	}

	b.followUp(s, i, fmt.Sprintf("Created zoom vertical **%s** (ID: %d)", name, id))
}

func (b *Bot) handleZoomDelete(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var verticalID int
	for _, opt := range opts {
		if opt.Name == "id" {
			verticalID = int(opt.IntValue())
		}
	}

	if verticalID == 0 {
		b.followUp(s, i, "Please specify a vertical ID.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	b.db.DeleteZoomVertical(ctx, verticalID)
	b.followUp(s, i, fmt.Sprintf("Vertical #%d has been deactivated.", verticalID))
}

// handleRoleAudit checks for role conflicts across all guild members.
func (b *Bot) handleRoleAudit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command can only be used in a server.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if !b.cfg.IsStaff(i.Member.Roles) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is restricted to staff.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	agents, err := b.db.GetAllAgents(ctx, false)
	if err != nil {
		b.followUp(s, i, fmt.Sprintf("Error: %v", err))
		return
	}

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

	if len(conflicts) == 0 {
		b.followUp(s, i, "No role conflicts found.")
		return
	}

	desc := strings.Join(conflicts, "\n")
	if len(desc) > 4000 {
		desc = desc[:4000] + "\n... (truncated)"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("Role Audit — %d Conflicts", len(conflicts)),
		Description: desc,
		Color:       0xE74C3C,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

// roleInList checks if targetID exists in the roles slice.
func roleInList(roles []string, targetID string) bool {
	if targetID == "" {
		return false
	}
	for _, r := range roles {
		if r == targetID {
			return true
		}
	}
	return false
}
