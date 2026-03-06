package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleStep1Submit processes the Step 1 modal submission.
func (b *Bot) handleStep1Submit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	fullName := getModalValue(data, "full_name")
	uplineManager := getModalValue(data, "upline_manager")
	agency := normalizeAgency(getModalValue(data, "agency"))
	licenseStatus := normalizeLicenseStatus(getModalValue(data, "license_status"))
	experienceLevel := normalizeExperience(getModalValue(data, "experience_level"))

	// Store temp data for Step 2
	b.modalState.Store(userID, &ModalTempData{
		FullName:        fullName,
		Agency:          agency,
		UplineManager:   uplineManager,
		ExperienceLevel: experienceLevel,
		LicenseStatus:   licenseStatus,
		ExpiresAt:       time.Now().Add(15 * time.Minute),
	})

	userIDInt, err := parseDiscordID(userID)
	if err == nil {
		b.db.LogActivity(context.Background(), userIDInt, "form_step1", fmt.Sprintf("Name: %s, Agency: %s", fullName, agency))
	}

	// Send ephemeral confirmation
	expLabel := experienceLevel
	if expLabel == "" {
		expLabel = "Not specified"
	}

	// If NOT licensed, ask about course enrollment before continuing
	if licenseStatus != "licensed" {
		b.showCourseEnrollmentPrompt(s, i, fullName, agency, licenseStatus, expLabel)
		return
	}

	// Licensed agents skip straight to Continue
	b.showLicensedAgentConfirmation(s, i, fullName, agency, licenseStatus, expLabel)
}

