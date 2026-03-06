package bot

import (
	"context"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/api/websocket"
	"license-bot-go/db"
)

// onLicensedRoleAdded triggers when a member claims to be licensed.
func (b *Bot) onLicensedRoleAdded(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("onLicensedRoleAdded panic: %v", r)
		}
	}()

	userID := e.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("onLicensedRoleAdded: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(e.GuildID)
	if err != nil {
		log.Printf("onLicensedRoleAdded: %v", err)
		return
	}

	log.Printf("Auto-verify: Licensed role added for %s (%s)", e.User.Username, userID)

	// Look up agent info from local DB first
	agent, err := b.db.GetAgent(context.Background(), userIDInt)

	var firstName, lastName, state string
	if err == nil && agent != nil && agent.FirstName != "" && agent.State != "" {
		firstName = agent.FirstName
		lastName = agent.LastName
		state = agent.State
		log.Printf("Auto-verify: Found agent in local DB: %s %s (%s)", firstName, lastName, state)
	} else {
		// No local data — ask user to verify manually
		log.Printf("Auto-verify: No local record for %s, asking to verify manually", userID)
		b.dmVerifyManually(s, userID)
		return
	}

	if firstName == "" || state == "" {
		b.dmVerifyManually(s, userID)
		return
	}

	// Attempt verification
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result := b.performVerification(ctx, firstName, lastName, state, userIDInt, guildIDInt)

	if result.Found && result.Match != nil {
		log.Printf("Auto-verify: SUCCESS for %s %s (%s)", firstName, lastName, state)
		b.publishEvent(websocket.EventLicenseVerified, websocket.LicenseVerifiedData{
			DiscordID: userID,
			State:     state,
			FullName:  firstName + " " + lastName,
			NPN:       result.Match.NPN,
		})
		b.db.MarkDeadlineVerified(context.Background(), userIDInt)
		b.dmVerificationSuccess(s, e, result.Match, state)
		b.postAutoVerifyToChannel(s, e, result.Match, state)
		go b.syncGHLStage(userIDInt, db.StageVerified)
	} else {
		b.handleAutoVerifyFail(s, userID, firstName, lastName, state, userIDInt, guildIDInt, result)
	}
}
