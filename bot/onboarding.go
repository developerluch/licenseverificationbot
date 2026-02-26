package bot

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/api/websocket"
	"license-bot-go/db"
)

// handleMemberJoin fires when a new member joins the guild.
// Creates a DB record at Stage 1 and posts a personalized welcome in #start-here.
func (b *Bot) handleMemberJoin(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("handleMemberJoin panic: %v", r)
		}
	}()

	if m.User.Bot {
		return
	}

	userIDInt, err := parseDiscordID(m.User.ID)
	if err != nil {
		log.Printf("handleMemberJoin: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(m.GuildID)
	if err != nil {
		log.Printf("handleMemberJoin: %v", err)
		return
	}

	stage := db.StageWelcome
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		CurrentStage: &stage,
	})
	b.db.LogActivity(context.Background(), userIDInt, "joined", "Member joined the server")

	log.Printf("Onboarding: new member %s (%s) joined", m.User.Username, m.User.ID)

	// Broadcast agent_joined event to WebSocket clients
	b.publishEvent(websocket.EventAgentJoined, websocket.AgentJoinedData{
		DiscordID: m.User.ID,
		Username:  m.User.Username,
		Stage:     db.StageWelcome,
	})

	// Assign @New role so they only see #start-here
	if b.cfg.NewRoleID != "" {
		if err := s.GuildMemberRoleAdd(m.GuildID, m.User.ID, b.cfg.NewRoleID); err != nil {
			log.Printf("Onboarding: failed to add @New role to %s: %v", m.User.ID, err)
		}
	}

	// Post personalized welcome in #start-here
	b.postStartHereWelcome(s, m.User.ID)
}

// handleRulesScreeningComplete is called when a member's pending status flips to false.
// Posts a personalized welcome in #start-here (if not already posted by handleMemberJoin).
func (b *Bot) handleRulesScreeningComplete(s *discordgo.Session, member *discordgo.Member) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("handleRulesScreeningComplete panic: %v", r)
		}
	}()

	userID := member.User.ID
	log.Printf("Onboarding: rules screening complete for %s (%s)", member.User.Username, userID)

	userIDInt, err := parseDiscordID(userID)
	if err == nil {
		b.db.LogActivity(context.Background(), userIDInt, "rules_accepted", "Completed rules screening")
	}

	// If they don't already have a welcome message in #start-here, post one now
	if _, exists := b.welcomeMessages.Load(userID); !exists {
		b.postStartHereWelcome(s, userID)
	}
}

// postStartHereWelcome posts a personalized welcome message with a "Get Started" button
// in #start-here, pinging the user so they know it's for them.
func (b *Bot) postStartHereWelcome(s *discordgo.Session, userID string) {
	channelID := b.cfg.StartHereChannelID
	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "\U0001f44b Welcome to VIPA!",
		Description: "We're excited to have you here! Click **Get Started** below to begin your onboarding.\n\n" +
			"This takes about 60 seconds and unlocks access to all server channels.",
		Color: 0x2ECC71,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "\U0001f4cb What You'll Do",
				Value: "1\ufe0f\u20e3 Click **Get Started**\n" +
					"2\ufe0f\u20e3 Fill out a quick intro form\n" +
					"3\ufe0f\u20e3 Get your roles assigned\n" +
					"4\ufe0f\u20e3 Unlock all channels!",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA \u2022 This message is just for you \u2b07\ufe0f",
		},
	}

	msg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "<@" + userID + ">",
		Embeds:  []*discordgo.MessageEmbed{embed},
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
	if err != nil {
		log.Printf("Onboarding: failed to post start-here welcome for %s: %v", userID, err)
		return
	}

	// Track the message so we can delete it when they click "Get Started"
	b.welcomeMessages.Store(userID, welcomeMsgRef{
		ChannelID: channelID,
		MessageID: msg.ID,
	})
	log.Printf("Onboarding: posted welcome in #start-here for %s (msg %s)", userID, msg.ID)
}

// handleStart is a failsafe /start command that opens the Get Started modal directly.
// Users can use this if their welcome message was missed or deleted.
func (b *Bot) handleStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Present the Step 1 modal directly — same as clicking "Get Started"
	b.handleGetStarted(s, i)
}
