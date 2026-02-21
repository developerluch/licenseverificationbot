package bot

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// handleMemberJoin fires when a new member joins the guild.
// Creates a DB record at Stage 1.
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
}

// handleRulesScreeningComplete is called when a member's pending status flips to false.
// Sends welcome DM with embed + "Get Started" button.
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

	// Build welcome embed
	embed := buildWelcomeEmbed()

	// Build "Get Started" button
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Get Started",
					Style:    discordgo.SuccessButton,
					CustomID: "vipa:onboarding_get_started",
					Emoji: &discordgo.ComponentEmoji{
						Name: "\U0001f680",
					},
				},
			},
		},
	}

	// Send DM
	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		log.Printf("Onboarding: cannot create DM channel for %s: %v", userID, err)
		// Fallback: post in #start-here
		b.postStartHereFallback(s, member)
		return
	}

	_, err = s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	})
	if err != nil {
		log.Printf("Onboarding: cannot send welcome DM to %s: %v", userID, err)
		b.postStartHereFallback(s, member)
	}
}

// postStartHereFallback posts a nudge in #start-here when DMs are disabled.
func (b *Bot) postStartHereFallback(s *discordgo.Session, member *discordgo.Member) {
	channelID := b.cfg.StartHereChannelID
	if channelID == "" {
		return
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: "<@" + member.User.ID + "> Welcome! Please enable DMs or click the button below to get started.",
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
		log.Printf("Onboarding: failed to post start-here fallback for %s: %v", member.User.ID, err)
	}
}
