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

// handleStart is a failsafe /start command that opens the Get Started modal directly.
// Users can use this if their welcome message was missed or deleted.
func (b *Bot) handleStart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Present the Step 1 modal directly — same as clicking "Get Started"
	b.handleGetStarted(s, i)
}
