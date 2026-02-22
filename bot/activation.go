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

// handleContract shows contracting manager Calendly links.
func (b *Bot) handleContract(s *discordgo.Session, i *discordgo.InteractionCreate) {
	managers, err := b.db.GetContractingManagers(context.Background())
	if err != nil {
		log.Printf("Contract: failed to get managers: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Failed to load contracting managers. Try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(managers) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No contracting managers are currently available. Contact your upline for help.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := buildContractingEmbed(managers)

	// Mark contracting_booked in DB
	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleContract: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleContract: %v", err)
		return
	}
	booked := true
	stage := db.StageContracting
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		ContractingBooked: &booked,
		CurrentStage:      &stage,
	})
	b.db.LogActivity(context.Background(), userIDInt, "contracting_viewed", "Viewed contracting managers")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleSetup shows or manages the agent setup checklist.
func (b *Bot) handleSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleSetup: %v", err)
		return
	}

	opts := i.ApplicationCommandData().Options
	action := ""
	for _, opt := range opts {
		if opt.Name == "action" {
			action = opt.StringValue()
		}
	}

	if action == "complete" {
		complete, err := b.db.IsSetupComplete(context.Background(), userIDInt)
		if err != nil {
			log.Printf("Setup: check failed: %v", err)
		}
		if !complete {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You haven't completed all setup items yet. Use `/setup start` to see your checklist.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		guildIDInt, err := parseDiscordID(i.GuildID)
		if err != nil {
			log.Printf("handleSetup: %v", err)
			return
		}
		b.activateAgent(s, userIDInt, guildIDInt, i.Member)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "\U0001f389 **Congratulations!** You've completed all setup steps and are now a fully active VIPA agent!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Default: show checklist
	progress, err := b.db.GetSetupProgress(context.Background(), userIDInt)
	if err != nil {
		log.Printf("Setup: get progress failed: %v", err)
		progress = make(map[string]bool)
	}

	// Set stage to setup if not already there
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleSetup: %v", err)
		return
	}
	stage := db.StageSetup
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		CurrentStage: &stage,
	})

	embed := buildSetupEmbed(i.Member.User.Username, progress)
	rows := buildSetupButtons(progress)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: rows,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleSetupItem marks a setup item as completed and updates the message.
func (b *Bot) handleSetupItem(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID
	itemKey := strings.TrimPrefix(customID, "vipa:setup:")

	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleSetupItem: %v", err)
		return
	}

	if err := b.db.CompleteSetupItem(context.Background(), userIDInt, itemKey); err != nil {
		log.Printf("Setup: mark item failed: %v", err)
	}
	b.db.LogActivity(context.Background(), userIDInt, "setup_item", fmt.Sprintf("Completed: %s", itemKey))

	// Rebuild the checklist
	progress, err := b.db.GetSetupProgress(context.Background(), userIDInt)
	if err != nil {
		log.Printf("Setup: get progress failed: %v", err)
		progress = make(map[string]bool)
	}

	embed := buildSetupEmbed(i.Member.User.Username, progress)
	rows := buildSetupButtons(progress)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: rows,
		},
	})
}

// handleSetupCompleteAll fires when "Complete Setup" button is clicked.
func (b *Bot) handleSetupCompleteAll(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleSetupCompleteAll: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleSetupCompleteAll: %v", err)
		return
	}

	complete, err := b.db.IsSetupComplete(context.Background(), userIDInt)
	if err != nil || !complete {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Not all setup items are complete yet.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	b.activateAgent(s, userIDInt, guildIDInt, i.Member)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Content:    "\U0001f389 **Congratulations!** You've completed all setup steps and are now a fully active VIPA agent!",
			Embeds:     []*discordgo.MessageEmbed{},
			Components: []discordgo.MessageComponent{},
		},
	})
}

// activateAgent promotes an agent to active status.
func (b *Bot) activateAgent(s *discordgo.Session, discordID, guildID int64, member *discordgo.Member) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("activateAgent panic: %v", r)
		}
	}()

	now := time.Now()
	stage := db.StageActive
	setupDone := true
	b.db.UpsertAgent(context.Background(), discordID, guildID, db.AgentUpdate{
		CurrentStage:   &stage,
		SetupCompleted: &setupDone,
		ActivatedAt:    &now,
		LastActive:     &now,
	})
	b.db.LogActivity(context.Background(), discordID, "activated", "Agent completed setup and is now active")

	// GHL sync
	go b.syncGHLStage(discordID, db.StageActive)

	userID := strconv.FormatInt(discordID, 10)
	guildIDStr := strconv.FormatInt(guildID, 10)

	// Swap roles: remove Student + Licensed-Agent, add Active-Agent
	if b.cfg.ActiveAgentRoleID != "" {
		if err := s.GuildMemberRoleAdd(guildIDStr, userID, b.cfg.ActiveAgentRoleID); err != nil {
			log.Printf("Activation: failed to add Active-Agent role: %v", err)
		}
	}
	if b.cfg.StudentRoleID != "" {
		s.GuildMemberRoleRemove(guildIDStr, userID, b.cfg.StudentRoleID)
	}
	if b.cfg.LicensedAgentRoleID != "" {
		s.GuildMemberRoleRemove(guildIDStr, userID, b.cfg.LicensedAgentRoleID)
	}

	// Post activation announcement
	agent, _ := b.db.GetAgent(context.Background(), discordID)
	agency := ""
	if agent != nil {
		agency = agent.Agency
	}
	embed := buildActivationEmbed(member, agency)

	if b.cfg.GreetingsChannelID != "" {
		s.ChannelMessageSendEmbed(b.cfg.GreetingsChannelID, embed)
	}
	if b.cfg.HiringLogChannelID != "" {
		s.ChannelMessageSendEmbed(b.cfg.HiringLogChannelID, embed)
	}

	// Congrats DM
	b.dmUser(s, userID,
		"\U0001f389 **Congratulations!** You're now a fully active VIPA agent!\n\n"+
			"All setup steps are complete. Welcome to the team!")
}

// buildSetupButtons creates the button rows for the setup checklist.
func buildSetupButtons(progress map[string]bool) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent
	for _, item := range db.SetupItems {
		style := discordgo.SecondaryButton
		label := item.Label
		disabled := false
		if progress[item.Key] {
			style = discordgo.SuccessButton
			label = "\u2705 " + item.Label
			disabled = true
		}
		buttons = append(buttons, discordgo.Button{
			Label:    label,
			Style:    style,
			CustomID: "vipa:setup:" + item.Key,
			Disabled: disabled,
		})
	}

	// Discord allows max 5 buttons per row
	var rows []discordgo.MessageComponent
	for j := 0; j < len(buttons); j += 5 {
		end := j + 5
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, discordgo.ActionsRow{Components: buttons[j:end]})
	}

	// Add "Complete Setup" button if all items done
	allDone := true
	for _, item := range db.SetupItems {
		if !progress[item.Key] {
			allDone = false
			break
		}
	}
	if allDone {
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "\U0001f389 Complete Setup",
					Style:    discordgo.SuccessButton,
					CustomID: "vipa:setup_complete_all",
				},
			},
		})
	}

	return rows
}
