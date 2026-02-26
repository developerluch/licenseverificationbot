package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/api/websocket"
	"license-bot-go/db"
	"license-bot-go/scrapers"
)

// handleMemberUpdate fires when a member's roles change.
// We watch for the onboarding bot adding @Licensed-Agent or @Student roles.
func (b *Bot) handleMemberUpdate(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("handleMemberUpdate panic: %v", r)
		}
	}()

	if e.BeforeUpdate == nil {
		return // Can't compare without before state
	}

	// Check if rules screening was completed (pending -> not pending)
	if e.BeforeUpdate.Pending && !e.Pending {
		go b.handleRulesScreeningComplete(s, e.Member)
		return
	}

	newRoles := roleSet(e.Roles)
	oldRoles := roleSet(e.BeforeUpdate.Roles)

	// Check if @Licensed-Agent role was just added
	if b.cfg.LicensedAgentRoleID != "" {
		if newRoles[b.cfg.LicensedAgentRoleID] && !oldRoles[b.cfg.LicensedAgentRoleID] {
			go b.onLicensedRoleAdded(s, e)
			return
		}
	}

	// Check if @Student role was just added (unlicensed path)
	if b.cfg.StudentRoleID != "" {
		if newRoles[b.cfg.StudentRoleID] && !oldRoles[b.cfg.StudentRoleID] {
			go b.onStudentRoleAdded(s, e)
			return
		}
	}
}

// onLicensedRoleAdded triggers when a member claims to be licensed.
// We attempt immediate verification using their info from the onboarding DB.
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
		b.dmUser(s, userID,
			"Welcome! We see you're a licensed agent. "+
				"Please use `/verify first_name:YourFirst last_name:YourLast state:XX` "+
				"to verify your license and get access to agent channels.")
		return
	}

	if firstName == "" || state == "" {
		b.dmUser(s, userID,
			"Welcome! We see you're a licensed agent. "+
				"Please use `/verify first_name:YourFirst last_name:YourLast state:XX` "+
				"to verify your license and get access to agent channels.")
		return
	}

	// Attempt verification
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result := b.performVerification(ctx, firstName, lastName, state, userIDInt, guildIDInt)

	if result.Found && result.Match != nil {
		log.Printf("Auto-verify: SUCCESS for %s %s (%s)", firstName, lastName, state)

		// Broadcast license_verified event
		b.publishEvent(websocket.EventLicenseVerified, websocket.LicenseVerifiedData{
			DiscordID: userID,
			State:     state,
			FullName:  firstName + " " + lastName,
			NPN:       result.Match.NPN,
		})

		// Mark any existing deadline as verified
		b.db.MarkDeadlineVerified(context.Background(), userIDInt)

		// Send DM with license details
		b.dmVerificationSuccess(s, e, result.Match, state)

		// Post to channel
		b.postAutoVerifyToChannel(s, e, result.Match, state)

		// GHL sync
		go b.syncGHLStage(userIDInt, db.StageVerified)

	} else {
		log.Printf("Auto-verify: FAILED for %s %s (%s): %s",
			firstName, lastName, state, result.Error+result.Message)

		// Set a 30-day deadline
		b.createVerificationDeadline(userIDInt, guildIDInt, firstName, lastName, state, "pending_licensed")

		b.dmUser(s, userID, fmt.Sprintf(
			"**License Verification Pending**\n\n"+
				"We couldn't automatically verify your license for **%s %s** in **%s**.\n\n"+
				"This could happen if:\n"+
				"- Your name doesn't match what's on file\n"+
				"- Your license is still being processed\n\n"+
				"You have **30 days** to get verified. You can:\n"+
				"1. Use `/verify first_name:YourFirst last_name:YourLast state:XX`\n"+
				"2. Contact your upline for manual verification\n\n"+
				"We'll check again periodically and send you reminders.",
			firstName, lastName, state))
	}
}

// onStudentRoleAdded triggers when a member is unlicensed/studying.
// We set a 30-day deadline for them to get licensed.
func (b *Bot) onStudentRoleAdded(s *discordgo.Session, e *discordgo.GuildMemberUpdate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("onStudentRoleAdded panic: %v", r)
		}
	}()

	userID := e.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("onStudentRoleAdded: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(e.GuildID)
	if err != nil {
		log.Printf("onStudentRoleAdded: %v", err)
		return
	}

	log.Printf("Auto-verify: Student role added for %s (%s)", e.User.Username, userID)

	// Look up agent info from local DB
	var firstName, lastName, state string
	agent, err := b.db.GetAgent(context.Background(), userIDInt)
	if err == nil && agent != nil && agent.FirstName != "" {
		firstName = agent.FirstName
		lastName = agent.LastName
		state = agent.State
	}
	if firstName == "" {
		firstName = e.User.Username
	}

	// Create 30-day deadline
	b.createVerificationDeadline(userIDInt, guildIDInt, firstName, lastName, state, "student")

	b.dmUser(s, userID, fmt.Sprintf(
		"**Welcome to the team!**\n\n"+
			"You've been registered as a student agent. "+
			"You have **30 days** to get your insurance license.\n\n"+
			"Once you pass your exam, use `/verify first_name:YourFirst last_name:YourLast state:XX` "+
			"to verify your license and unlock agent channels.\n\n"+
			"We'll send you reminders as your deadline approaches. Good luck with your studies!"))
}

func (b *Bot) createVerificationDeadline(discordID, guildID int64, firstName, lastName, state, status string) {
	deadline := db.VerificationDeadline{
		DiscordID:     discordID,
		GuildID:       guildID,
		FirstName:     firstName,
		LastName:      lastName,
		HomeState:     state,
		LicenseStatus: status,
		DeadlineAt:    time.Now().Add(30 * 24 * time.Hour),
	}
	if err := b.db.CreateDeadline(context.Background(), deadline); err != nil {
		log.Printf("Failed to create deadline for %d: %v", discordID, err)
	}
}

func (b *Bot) dmUser(s *discordgo.Session, userID, content string) {
	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		log.Printf("Cannot create DM channel for %s: %v", userID, err)
		return
	}
	_, err = s.ChannelMessageSend(channel.ID, content)
	if err != nil {
		log.Printf("Cannot send DM to %s: %v", userID, err)
	}
}

func (b *Bot) dmVerificationSuccess(s *discordgo.Session, e *discordgo.GuildMemberUpdate, match *scrapers.LicenseResult, state string) {
	channel, err := s.UserChannelCreate(e.User.ID)
	if err != nil {
		log.Printf("Cannot create DM channel for auto-verify: %v", err)
		return
	}

	var fields []*discordgo.MessageEmbedField
	fields = append(fields, &discordgo.MessageEmbedField{Name: "Full Name", Value: nvl(match.FullName, "N/A"), Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "State", Value: state, Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "Status", Value: nvl(match.Status, "N/A"), Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "License #", Value: nvl(match.LicenseNumber, "N/A"), Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "NPN", Value: nvl(match.NPN, "N/A"), Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "License Type", Value: nvl(match.LicenseType, "N/A"), Inline: true})

	if match.ExpirationDate != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Expiration Date", Value: match.ExpirationDate, Inline: true})
	}
	if match.LOAs != "" {
		loas := match.LOAs
		if len(loas) > 900 {
			loas = loas[:900] + "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Lines of Authority", Value: loas, Inline: false})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "\u200b\nNext Step: Contracting",
		Value: "Use `/contract` in the server to book your contracting appointment.\n\n" +
			"**What to Prepare:**\n" +
			"- Government-issued photo ID\n" +
			"- Social Security number\n" +
			"- E&O insurance info\n" +
			"- Bank info for direct deposit\n" +
			"- Resident state license number",
		Inline: false,
	})

	embed := &discordgo.MessageEmbed{
		Title:       "License Automatically Verified!",
		Description: fmt.Sprintf("Welcome **%s**! Your license was verified automatically. Here are your details:", e.User.Username),
		Color:       0x2ECC71,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "VIPA License Verification"},
	}

	s.ChannelMessageSendEmbed(channel.ID, embed)
}

func (b *Bot) postAutoVerifyToChannel(s *discordgo.Session, e *discordgo.GuildMemberUpdate, match *scrapers.LicenseResult, state string) {
	channelID := b.verifyLogChannelID()
	if channelID == "" {
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "License Auto-Verified!",
		Description: fmt.Sprintf(
			"<@%s> was automatically verified as a licensed agent.\n\n"+
				"**Name:** %s\n"+
				"**License #:** %s\n"+
				"**State:** %s | **Status:** %s",
			e.User.ID, match.FullName, nvl(match.LicenseNumber, "N/A"), state, match.Status,
		),
		Color:     0x2ECC71,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}

func roleSet(roles []string) map[string]bool {
	m := make(map[string]bool, len(roles))
	for _, r := range roles {
		m[r] = true
	}
	return m
}
