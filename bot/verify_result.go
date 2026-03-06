package bot

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"license-bot-go/api/websocket"
	"license-bot-go/db"
	"license-bot-go/scrapers"
)

func (b *Bot) handleVerifyResult(s *discordgo.Session, i *discordgo.InteractionCreate,
	match *scrapers.LicenseResult, results []scrapers.LicenseResult,
	firstName, lastName, state, userID string, userIDInt, guildIDInt int64) {
	if match != nil {
		b.handleVerifySuccess(s, i, match, firstName, lastName, state, userID, userIDInt, guildIDInt)
	} else if len(results) > 0 && results[0].Error != "" {
		b.followUp(s, i, fmt.Sprintf("**Lookup Error:** %s\n\nThe %s lookup may be temporarily unavailable. Try again later.", results[0].Error, state))
	} else {
		b.handleVerifyNotFound(s, i, firstName, lastName, state)
	}
}

func (b *Bot) handleVerifySuccess(s *discordgo.Session, i *discordgo.InteractionCreate,
	match *scrapers.LicenseResult, firstName, lastName, state, userID string, userIDInt, guildIDInt int64) {
	verified := true
	stg := db.StageVerified
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		FirstName:       &firstName,
		LastName:        &lastName,
		State:           &state,
		LicenseVerified: &verified,
		LicenseNPN:      &match.NPN,
		CurrentStage:    &stg,
	})
	b.saveLicenseCheckRecord(userIDInt, guildIDInt, firstName, lastName, state, match)
	b.broadcastVerifySuccess(userID, state, firstName, lastName, match)
	b.executeVerifySecondaryActions(s, i, match, state, userIDInt)
	b.respondVerifySuccess(s, i, match, state)
}

func (b *Bot) saveLicenseCheckRecord(userIDInt, guildIDInt int64, firstName, lastName, state string, match *scrapers.LicenseResult) {
	b.db.SaveLicenseCheck(context.Background(), db.LicenseCheck{
		DiscordID:      userIDInt,
		GuildID:        guildIDInt,
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
}

func (b *Bot) broadcastVerifySuccess(userID, state, firstName, lastName string, match *scrapers.LicenseResult) {
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
}

func (b *Bot) executeVerifySecondaryActions(s *discordgo.Session, i *discordgo.InteractionCreate, match *scrapers.LicenseResult, state string, userIDInt int64) {
	go b.postToChannel(s, i, match, state)
	go b.dmNextSteps(s, i, match, state)
	go b.assignRoles(s, i)
	go b.syncGHLStage(userIDInt, db.StageVerified)
}

func (b *Bot) respondVerifySuccess(s *discordgo.Session, i *discordgo.InteractionCreate, match *scrapers.LicenseResult, state string) {
	b.followUp(s, i, fmt.Sprintf("**License Verified!**\n\n**Name:** %s\n**License #:** %s\n**NPN:** %s\n**State:** %s\n**Status:** %s\n**Type:** %s\n\nYou've been promoted to **Licensed Agent**!",
		nvl(match.FullName, "N/A"), nvl(match.LicenseNumber, "N/A"), nvl(match.NPN, "N/A"), state, nvl(match.Status, "N/A"), nvl(match.LicenseType, "N/A"),
	))
}

func (b *Bot) handleVerifyNotFound(s *discordgo.Session, i *discordgo.InteractionCreate, firstName, lastName, state string) {
	msg := fmt.Sprintf("**Could not verify your license.**\n\n- Your name may not match what's on file\n- Your license may not be processed yet\n- You may be licensed in a different state\n\n**Searched:** %s %s in %s\n\nContact your upline for manual verification.", firstName, lastName, state,
	)
	scraper := b.registry.GetScraper(state)
	if url := scraper.ManualLookupURL(); url != "" {
		msg += fmt.Sprintf("\n\nManual lookup: %s", url)
	}
	b.followUp(s, i, msg)
}
