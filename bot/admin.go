package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleAgentCommand routes /agent subcommands (Staff only).
func (b *Bot) handleAgentCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "info":
		b.handleAgentInfo(s, i, sub.Options)
	case "list":
		b.handleAgentList(s, i, sub.Options)
	case "nudge":
		b.handleAgentNudge(s, i, sub.Options)
	case "promote":
		b.handleAgentPromote(s, i, sub.Options)
	case "stats":
		b.handleAgentStats(s, i)
	case "kick":
		b.handleAgentKick(s, i, sub.Options)
	case "assign-manager":
		b.handleAssignManager(s, i, sub.Options)
	}
}

func (b *Bot) handleAgentInfo(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(opts) == 0 {
		return
	}
	targetUser := opts[0].UserValue(s)
	if targetUser == nil {
		return
	}

	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleAgentInfo: %v", err)
		return
	}
	agent, err := b.db.GetAgent(context.Background(), userIDInt)
	if err != nil || agent == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Agent not found in database.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	member, _ := s.GuildMember(i.GuildID, targetUser.ID)
	embed := buildAgentProfileEmbed(agent, member)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleAgentList(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var stage int
	for _, opt := range opts {
		if opt.Name == "stage" {
			stage = int(opt.IntValue())
		}
	}

	var agents []db.Agent
	var err error
	if stage > 0 {
		agents, err = b.db.GetAgentsByStage(context.Background(), stage)
	} else {
		agents, err = b.db.GetAllAgents(context.Background(), false)
	}

	if err != nil {
		log.Printf("admin handler: handleAgentList: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An error occurred. Please try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(agents) == 0 {
		msg := "No agents found."
		if stage > 0 {
			msg = fmt.Sprintf("No agents at stage %d.", stage)
		}
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: msg,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	limit := 25
	if len(agents) < limit {
		limit = len(agents)
	}

	var lines []string
	for _, a := range agents[:limit] {
		name := strings.TrimSpace(a.FirstName + " " + a.LastName)
		if name == "" {
			name = "Unknown"
		}
		lines = append(lines, fmt.Sprintf("<@%d> \u2014 %s (%s) [Stage %d]",
			a.DiscordID, name, nvl(a.Agency, "N/A"), a.CurrentStage))
	}

	title := "All Agents"
	if stage > 0 {
		title = fmt.Sprintf("Agents at Stage %d \u2014 %s", stage, stageLabel(stage))
	}

	content := fmt.Sprintf("**%s** (%d total)\n\n%s", title, len(agents), strings.Join(lines, "\n"))
	if len(agents) > limit {
		content += fmt.Sprintf("\n\n...and %d more", len(agents)-limit)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleAgentNudge(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(opts) == 0 {
		return
	}
	targetUser := opts[0].UserValue(s)
	if targetUser == nil {
		return
	}

	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleAgentNudge: %v", err)
		return
	}
	agent, _ := b.db.GetAgent(context.Background(), userIDInt)
	name := "Agent"
	if agent != nil && agent.FirstName != "" {
		name = agent.FirstName
	}

	weeksIn := 1
	if agent != nil {
		weeksIn = int(time.Since(agent.CreatedAt).Hours() / (24 * 7))
		if weeksIn < 1 {
			weeksIn = 1
		}
	}

	embed := buildCheckinEmbed(name, weeksIn)
	weekStartStr := time.Now().Truncate(24 * time.Hour).Format("2006-01-02")

	channel, err := s.UserChannelCreate(targetUser.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Cannot DM <@%s>: %v", targetUser.ID, err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_, sendErr := s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "\u2705 On Track",
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("vipa:checkin:on_track:%s", weekStartStr),
					},
					discordgo.Button{
						Label:    "\u23f8\ufe0f Need Help",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("vipa:checkin:need_help:%s", weekStartStr),
					},
					discordgo.Button{
						Label:    "\U0001f393 Got Licensed!",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("vipa:checkin:got_licensed:%s", weekStartStr),
					},
				},
			},
		},
	})

	b.db.LogActivity(context.Background(), userIDInt, "nudge", "Manual check-in sent by staff")

	msg := fmt.Sprintf("Check-in DM sent to <@%s>.", targetUser.ID)
	if sendErr != nil {
		msg = fmt.Sprintf("Failed to send DM to <@%s>: %v", targetUser.ID, sendErr)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleAgentPromote(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	var level string
	for _, opt := range opts {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "level":
			level = opt.StringValue()
		}
	}

	if targetUser == nil {
		return
	}

	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleAgentPromote: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleAgentPromote: %v", err)
		return
	}

	switch level {
	case "licensed":
		stage := db.StageVerified
		b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
			CurrentStage: &stage,
		})
		if b.cfg.LicensedAgentRoleID != "" {
			s.GuildMemberRoleAdd(i.GuildID, targetUser.ID, b.cfg.LicensedAgentRoleID)
		}
		if b.cfg.StudentRoleID != "" {
			s.GuildMemberRoleRemove(i.GuildID, targetUser.ID, b.cfg.StudentRoleID)
		}
		b.db.LogActivity(context.Background(), userIDInt, "promoted", "Manually promoted to Licensed by staff")

	case "active":
		member, _ := s.GuildMember(i.GuildID, targetUser.ID)
		if member == nil {
			member = &discordgo.Member{User: targetUser}
		}
		b.activateAgent(s, userIDInt, guildIDInt, member)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("<@%s> has been promoted to **%s**.", targetUser.ID, level),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleAgentStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	counts, err := b.db.GetAgentCounts(context.Background())
	if err != nil {
		log.Printf("admin handler: handleAgentStats: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An error occurred. Please try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	kicked, _ := b.db.GetKickedCount(context.Background())
	total := 0
	var lines []string
	for stage := 1; stage <= 8; stage++ {
		count := counts[stage]
		total += count
		barLen := count
		if barLen > 30 {
			barLen = 30
		}
		bar := strings.Repeat("\u2588", barLen)
		lines = append(lines, fmt.Sprintf("`%d` %s %s (%d)",
			stage, stageLabel(stage), bar, count))
	}

	content := fmt.Sprintf("**Onboarding Dashboard**\nTotal active: %d | Kicked: %d\n\n%s",
		total, kicked, strings.Join(lines, "\n"))

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: content,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleAgentKick(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var targetUser *discordgo.User
	reason := "Removed by staff"
	for _, opt := range opts {
		switch opt.Name {
		case "user":
			targetUser = opt.UserValue(s)
		case "reason":
			reason = opt.StringValue()
		}
	}

	if targetUser == nil {
		return
	}

	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleAgentKick: %v", err)
		return
	}

	// DM the user
	b.dmUser(s, targetUser.ID, fmt.Sprintf(
		"You have been removed from the VIPA server.\n\nReason: %s\n\nIf you believe this is an error, contact your upline.", reason))

	// Mark in DB
	b.db.KickAgent(context.Background(), userIDInt, reason)

	// GHL: mark opportunity as lost
	go b.markGHLLost(userIDInt)

	// Kick from server
	err = s.GuildMemberDeleteWithReason(i.GuildID, targetUser.ID, reason)
	msg := fmt.Sprintf("<@%s> has been removed. Reason: %s", targetUser.ID, reason)
	if err != nil {
		msg += fmt.Sprintf("\n\n\u26a0\ufe0f Discord kick failed: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleContractingCommand routes /contracting subcommands (Admin only).
func (b *Bot) handleContractingCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "add":
		b.handleContractingAdd(s, i, sub.Options)
	case "remove":
		b.handleContractingRemove(s, i, sub.Options)
	case "list":
		b.handleContractingList(s, i)
	}
}

func (b *Bot) handleContractingAdd(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var name, url string
	priority := 1
	for _, opt := range opts {
		switch opt.Name {
		case "name":
			name = opt.StringValue()
		case "url":
			url = opt.StringValue()
		case "priority":
			priority = int(opt.IntValue())
		}
	}

	if url != "" && !strings.HasPrefix(url, "https://calendly.com/") {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "URL must be a valid Calendly link (https://calendly.com/...)",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	err := b.db.AddContractingManager(context.Background(), name, url, priority)
	msg := fmt.Sprintf("Added contracting manager **%s** (priority %d).", name, priority)
	if err != nil {
		log.Printf("admin handler: handleContractingAdd: %v", err)
		msg = "An error occurred. Please try again later."
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleContractingRemove(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(opts) == 0 {
		return
	}
	name := opts[0].StringValue()

	err := b.db.DeactivateContractingManager(context.Background(), name)
	msg := fmt.Sprintf("Deactivated contracting manager **%s**.", name)
	if err != nil {
		log.Printf("admin handler: handleContractingRemove: %v", err)
		msg = "An error occurred. Please try again later."
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handleContractingList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	managers, err := b.db.GetContractingManagers(context.Background())
	if err != nil {
		log.Printf("admin handler: handleContractingList: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "An error occurred. Please try again later.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	if len(managers) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No active contracting managers.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var lines []string
	for _, m := range managers {
		lines = append(lines, fmt.Sprintf("**%s** (priority %d) \u2014 %s", m.ManagerName, m.Priority, m.CalendlyURL))
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "**Active Contracting Managers:**\n\n" + strings.Join(lines, "\n"),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleOnboardingSetup posts the "Get Started" panel in #start-here (Admin only).
func (b *Bot) handleOnboardingSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	channelID := b.cfg.StartHereChannelID
	if channelID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "START_HERE_CHANNEL_ID is not configured.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := buildWelcomeEmbed()
	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Get Started",
						Style:    discordgo.SuccessButton,
						CustomID: "vipa:onboarding_get_started",
						Emoji:    &discordgo.ComponentEmoji{Name: "\U0001f680"},
					},
				},
			},
		},
	})

	msg := "Get Started panel posted in <#" + channelID + ">!"
	if err != nil {
		msg = fmt.Sprintf("Failed to post panel: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleSetupRules posts the rules embed in #rules (Admin only).
func (b *Bot) handleSetupRules(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	channelID := b.cfg.RulesChannelID
	if channelID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "RULES_CHANNEL_ID is not configured.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	embed := buildRulesEmbed()
	_, err := s.ChannelMessageSendEmbed(channelID, embed)

	msg := "Rules embed posted in <#" + channelID + ">!"
	if err != nil {
		msg = fmt.Sprintf("Failed to post rules: %v", err)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleAssignManager assigns a direct manager to an agent.
func (b *Bot) handleAssignManager(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(opts) < 2 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Please provide both agent and manager.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	var agentUser, managerUser *discordgo.User
	for _, opt := range opts {
		switch opt.Name {
		case "agent":
			agentUser = opt.UserValue(s)
		case "manager":
			managerUser = opt.UserValue(s)
		}
	}

	if agentUser == nil || managerUser == nil {
		return
	}

	agentIDInt, err := parseDiscordID(agentUser.ID)
	if err != nil {
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		return
	}
	managerIDInt, err := parseDiscordID(managerUser.ID)
	if err != nil {
		return
	}

	managerName := managerUser.GlobalName
	if managerName == "" {
		managerName = managerUser.Username
	}

	b.db.UpsertAgent(context.Background(), agentIDInt, guildIDInt, db.AgentUpdate{
		DirectManagerDiscordID: &managerIDInt,
		DirectManagerName:      &managerName,
	})

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Assigned <@%s> as direct manager for <@%s>.", managerUser.ID, agentUser.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	log.Printf("Admin: assigned manager %s to agent %s", managerUser.ID, agentUser.ID)
}
