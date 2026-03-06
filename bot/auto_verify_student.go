package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

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
