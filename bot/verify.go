package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/api/websocket"
	"license-bot-go/db"
	"license-bot-go/scrapers"
)

func (b *Bot) handleVerify(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Step 1: Defer (ephemeral)
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Defer failed: %v", err)
		return
	}

	// Step 2: Extract options
	opts := i.ApplicationCommandData().Options
	optMap := make(map[string]string)
	for _, opt := range opts {
		optMap[opt.Name] = opt.StringValue()
	}

	firstName := optMap["first_name"]
	lastName := optMap["last_name"]
	state := strings.ToUpper(strings.TrimSpace(optMap["state"]))
	phone := optMap["phone"]

	userID := i.Member.User.ID
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}

	log.Printf("License verify: %s %s (%s) by %s", firstName, lastName, state, userID)

	// Step 3: Pull state from DB if not provided
	if state == "" {
		agent, err := b.db.GetAgent(context.Background(), userIDInt)
		if err == nil && agent != nil {
			state = agent.State
		}
	}

	if state == "" || len(state) != 2 {
		b.followUp(s, i, "Please provide your 2-letter state code.\nExample: `/verify first_name:John last_name:Doe state:FL`")
		return
	}

	// Step 4: Save phone number if provided
	if phone != "" {
		cleanPhone := cleanPhoneNumber(phone)
		if cleanPhone != "" {
			b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
				PhoneNumber: &cleanPhone,
			})
		}
	}

	// Step 5: Get scraper and lookup
	scraper := b.registry.GetScraper(state)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	results, err := scraper.LookupByName(ctx, firstName, lastName)
	if err != nil {
		log.Printf("Scraper error for %s: %v", state, err)
		msg := fmt.Sprintf("The %s license lookup is temporarily unavailable. Please try again later.", state)
		if url := scraper.ManualLookupURL(); url != "" {
			msg += fmt.Sprintf("\n\nManual lookup: %s", url)
		}
		b.followUp(s, i, msg)
		return
	}

	// Step 6: Find best match
	var match *scrapers.LicenseResult

	// First pass: prefer life-licensed active results
	for idx := range results {
		if results[idx].Found && results[idx].Active && results[idx].IsLifeLicensed() {
			match = &results[idx]
			break
		}
	}
	// Second pass: any active result
	if match == nil {
		for idx := range results {
			if results[idx].Found && results[idx].Active {
				match = &results[idx]
				break
			}
		}
	}

	// Step 7: Respond based on result
	if match != nil {
		// SUCCESS
		verified := true
		stage := db.StageVerified
		b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
			FirstName:       &firstName,
			LastName:        &lastName,
			State:           &state,
			LicenseVerified: &verified,
			LicenseNPN:      &match.NPN,
			CurrentStage:    &stage,
		})

		b.db.SaveLicenseCheck(context.Background(), db.LicenseCheck{
			DiscordID:      userIDInt,
			GuildID:        guildIDInt,
			FirstName:      firstName,
			LastName:        lastName,
			State:          state,
			NPN:            match.NPN,
			LicenseNumber:  match.LicenseNumber,
			LicenseType:    match.LicenseType,
			LicenseStatus:  match.Status,
			ExpirationDate: match.ExpirationDate,
			LOAs:           match.LOAs,
			Found:          true,
		})

		// Broadcast license_verified event
		b.publishEvent(websocket.EventLicenseVerified, websocket.LicenseVerifiedData{
			DiscordID: userID,
			State:     state,
			FullName:  firstName + " " + lastName,
			NPN:       match.NPN,
		})
		b.publishEvent(websocket.EventStageChanged, websocket.StageChangedData{
			DiscordID: userID,
			NewStage:  db.StageVerified,
			ChangedBy: "verify_command",
		})

		// Fire-and-forget secondary actions
		go b.postToChannel(s, i, match, state)
		go b.dmNextSteps(s, i, match, state)
		go b.assignRoles(s, i)
		go b.syncGHLStage(userIDInt, db.StageVerified)

		b.followUp(s, i, fmt.Sprintf(
			"**License Verified!**\n\n"+
				"**Name:** %s\n"+
				"**License #:** %s\n"+
				"**NPN:** %s\n"+
				"**State:** %s\n"+
				"**Status:** %s\n"+
				"**Type:** %s\n\n"+
				"You've been promoted to **Licensed Agent**!",
			nvl(match.FullName, "N/A"),
			nvl(match.LicenseNumber, "N/A"),
			nvl(match.NPN, "N/A"),
			state,
			nvl(match.Status, "N/A"),
			nvl(match.LicenseType, "N/A"),
		))

	} else if len(results) > 0 && results[0].Error != "" {
		// Error from scraper (e.g., manual URL fallback)
		b.followUp(s, i, fmt.Sprintf("**Lookup Error:** %s\n\nThe %s lookup may be temporarily unavailable. Try again later.", results[0].Error, state))

	} else {
		// Not found
		msg := fmt.Sprintf(
			"**Could not verify your license.**\n\n"+
				"- Your name may not match what's on file\n"+
				"- Your license may not be processed yet\n"+
				"- You may be licensed in a different state\n\n"+
				"**Searched:** %s %s in %s\n\n"+
				"Contact your upline for manual verification.",
			firstName, lastName, state,
		)
		if url := scraper.ManualLookupURL(); url != "" {
			msg += fmt.Sprintf("\n\nManual lookup: %s", url)
		}
		b.followUp(s, i, msg)
	}
}

func (b *Bot) followUp(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: content,
		Flags:   discordgo.MessageFlagsEphemeral,
	})
	if err != nil {
		log.Printf("Follow-up failed: %v", err)
	}
}

// verifyLogChannelID returns the channel ID for posting verification results.
// Falls back: LicenseVerifyLogChannelID -> LicenseCheckChannelID -> HiringLogChannelID.
func (b *Bot) verifyLogChannelID() string {
	if b.cfg.LicenseVerifyLogChannelID != "" {
		return b.cfg.LicenseVerifyLogChannelID
	}
	if b.cfg.LicenseCheckChannelID != "" {
		return b.cfg.LicenseCheckChannelID
	}
	return b.cfg.HiringLogChannelID
}

func (b *Bot) postToChannel(s *discordgo.Session, i *discordgo.InteractionCreate, match *scrapers.LicenseResult, state string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("postToChannel panic: %v", r)
		}
	}()

	channelID := b.verifyLogChannelID()
	if channelID == "" {
		return
	}

	userID := i.Member.User.ID
	embed := &discordgo.MessageEmbed{
		Title: "License Verified!",
		Description: fmt.Sprintf(
			"<@%s> verified as a licensed agent.\n\n"+
				"**Name:** %s\n"+
				"**License #:** %s\n"+
				"**State:** %s | **Status:** %s",
			userID, match.FullName, nvl(match.LicenseNumber, "N/A"), state, match.Status,
		),
		Color:     0x2ECC71,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.ChannelMessageSendEmbed(channelID, embed)
}

func (b *Bot) dmNextSteps(s *discordgo.Session, i *discordgo.InteractionCreate, match *scrapers.LicenseResult, state string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("dmNextSteps panic: %v", r)
		}
	}()

	channel, err := s.UserChannelCreate(i.Member.User.ID)
	if err != nil {
		log.Printf("Cannot create DM channel: %v", err)
		return
	}

	// Build license info fields
	var fields []*discordgo.MessageEmbedField

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Full Name",
		Value:  nvl(match.FullName, "N/A"),
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "State",
		Value:  state,
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Status",
		Value:  nvl(match.Status, "N/A"),
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "License #",
		Value:  nvl(match.LicenseNumber, "N/A"),
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "NPN",
		Value:  nvl(match.NPN, "N/A"),
		Inline: true,
	})
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "License Type",
		Value:  nvl(match.LicenseType, "N/A"),
		Inline: true,
	})

	if match.IssueDate != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Issue Date",
			Value:  match.IssueDate,
			Inline: true,
		})
	}
	if match.ExpirationDate != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Expiration Date",
			Value:  match.ExpirationDate,
			Inline: true,
		})
	}
	if match.Resident {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Residency",
			Value:  "Resident",
			Inline: true,
		})
	}
	if match.LOAs != "" {
		// Truncate LOAs if too long for embed field (max 1024 chars)
		loas := match.LOAs
		if len(loas) > 900 {
			loas = loas[:900] + "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Lines of Authority",
			Value:  loas,
			Inline: false,
		})
	}
	if match.BusinessAddress != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Business Address",
			Value:  match.BusinessAddress,
			Inline: false,
		})
	}
	if match.BusinessPhone != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Business Phone",
			Value:  match.BusinessPhone,
			Inline: true,
		})
	}
	if match.Email != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Email",
			Value:  match.Email,
			Inline: true,
		})
	}
	if match.County != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "County",
			Value:  match.County,
			Inline: true,
		})
	}

	// Add next steps section
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
		Title:       "License Verified!",
		Description: fmt.Sprintf("Welcome **%s**! Your license has been confirmed. Here are your full license details:", i.Member.User.Username),
		Color:       0x2ECC71,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA License Verification",
		},
	}

	s.ChannelMessageSendEmbed(channel.ID, embed)
}

func (b *Bot) assignRoles(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("assignRoles panic: %v", r)
		}
	}()

	if b.cfg.LicensedAgentRoleID != "" {
		err := s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, b.cfg.LicensedAgentRoleID)
		if err != nil {
			log.Printf("Failed to add Licensed Agent role: %v", err)
		}
	}

	if b.cfg.StudentRoleID != "" {
		err := s.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, b.cfg.StudentRoleID)
		if err != nil {
			log.Printf("Failed to remove Student role: %v", err)
		}
	}
}

// VerifyResult is the outcome of a license verification attempt.
type VerifyResult struct {
	Found   bool
	Match   *scrapers.LicenseResult
	Error   string
	Message string
}

// performVerification runs a license lookup and returns the result without any Discord interaction.
func (b *Bot) performVerification(ctx context.Context, firstName, lastName, state string, discordID, guildID int64) VerifyResult {
	if state == "" || len(state) != 2 {
		return VerifyResult{Error: "invalid state code"}
	}

	scraper := b.registry.GetScraper(state)

	results, err := scraper.LookupByName(ctx, firstName, lastName)
	if err != nil {
		msg := fmt.Sprintf("Lookup error for %s: %v", state, err)
		return VerifyResult{Error: msg}
	}

	// Find best match: prefer life-licensed active results
	var match *scrapers.LicenseResult
	for idx := range results {
		if results[idx].Found && results[idx].Active && results[idx].IsLifeLicensed() {
			match = &results[idx]
			break
		}
	}
	if match == nil {
		for idx := range results {
			if results[idx].Found && results[idx].Active {
				match = &results[idx]
				break
			}
		}
	}

	if match != nil {
		// Save to DB
		verified := true
		stage := db.StageVerified
		b.db.UpsertAgent(ctx, discordID, guildID, db.AgentUpdate{
			FirstName:       &firstName,
			LastName:        &lastName,
			State:           &state,
			LicenseVerified: &verified,
			LicenseNPN:      &match.NPN,
			CurrentStage:    &stage,
		})
		b.db.SaveLicenseCheck(ctx, db.LicenseCheck{
			DiscordID:      discordID,
			GuildID:        guildID,
			FirstName:      firstName,
			LastName:       lastName,
			State:          state,
			NPN:            match.NPN,
			LicenseNumber:  match.LicenseNumber,
			LicenseType:    match.LicenseType,
			LicenseStatus:  match.Status,
			ExpirationDate: match.ExpirationDate,
			LOAs:           match.LOAs,
			Found:          true,
		})
		return VerifyResult{Found: true, Match: match}
	}

	if len(results) > 0 && results[0].Error != "" {
		return VerifyResult{Error: results[0].Error}
	}

	return VerifyResult{
		Found:   false,
		Message: fmt.Sprintf("No active license found for %s %s in %s", firstName, lastName, state),
	}
}

func cleanPhoneNumber(phone string) string {
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)
	// Remove leading 1 for US numbers
	if len(digits) == 11 && digits[0] == '1' {
		digits = digits[1:]
	}
	// Require exactly 10 digits for a valid US phone number
	if len(digits) != 10 {
		return ""
	}
	return "+1" + digits
}

func nvl(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
