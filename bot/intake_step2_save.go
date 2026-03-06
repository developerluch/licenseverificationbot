package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/api/websocket"
	"license-bot-go/db"
)

// saveStep2FormData saves the Step 2 form submission to database and triggers async operations.
func (b *Bot) saveStep2FormData(s *discordgo.Session, i *discordgo.InteractionCreate, step1 *ModalTempData, userID string, firstName, lastName, homeState, roleBackground, goalsVision, funHobbies, phone string) error {
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return err
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return err
	}

	// Determine stage based on license status
	var stage int
	if step1.LicenseStatus == "licensed" {
		stage = db.StageVerified // 5
	} else {
		stage = db.StageStudent // 4
	}

	now := time.Now()
	courseEnrolled := step1.CourseEnrolled

	// Save to DB
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		FirstName:       &firstName,
		LastName:        &lastName,
		PhoneNumber:     &phone,
		State:           &homeState,
		Agency:          &step1.Agency,
		UplineManager:   &step1.UplineManager,
		ExperienceLevel: &step1.ExperienceLevel,
		LicenseStatus:   &step1.LicenseStatus,
		CourseEnrolled:  &courseEnrolled,
		RoleBackground:  &roleBackground,
		VisionGoals:     &goalsVision,
		FunHobbies:      &funHobbies,
		CurrentStage:    &stage,
		FormCompletedAt: &now,
		SortedAt:        &now,
		LastActive:      &now,
	})

	b.db.LogActivity(context.Background(), userIDInt, "form_complete",
		fmt.Sprintf("Agency: %s, License: %s, Enrolled: %v, State: %s", step1.Agency, step1.LicenseStatus, courseEnrolled, homeState))

	// Broadcast events
	b.publishEvent(websocket.EventFormCompleted, websocket.FormCompletedData{
		DiscordID: userID,
		FullName:  step1.FullName,
		Agency:    step1.Agency,
		License:   step1.LicenseStatus,
	})
	b.publishEvent(websocket.EventStageChanged, websocket.StageChangedData{
		DiscordID: userID,
		NewStage:  stage,
		ChangedBy: "onboarding_form",
	})

	// Prepare agent data for hiring log and greetings
	agentData := map[string]string{
		"full_name":        step1.FullName,
		"agency":           step1.Agency,
		"upline":           step1.UplineManager,
		"experience":       step1.ExperienceLevel,
		"license_status":   step1.LicenseStatus,
		"role_background":  roleBackground,
		"goals_vision":     goalsVision,
		"fun_hobbies":      funHobbies,
		"home_state":       homeState,
	}

	// Trigger async operations
	b.triggerStep2AsyncOperations(s, i, step1, userID, userIDInt, guildIDInt, stage, agentData)

	return nil
}
