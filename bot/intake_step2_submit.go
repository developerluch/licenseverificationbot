package bot

import (
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleStep2Submit processes the Step 2 modal submission and completes onboarding.
func (b *Bot) handleStep2Submit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("handleStep2Submit panic: %v", r)
		}
	}()

	// Defer response first (must be within 3 seconds)
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		log.Printf("handleStep2Submit: defer failed: %v", err)
		return
	}

	data := i.ModalSubmitData()
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	// Get Step 1 temp data
	val, ok := b.modalState.Load(userID)
	if !ok {
		b.followUp(s, i, "Your session expired. Please click **Get Started** again.")
		return
	}
	step1 := val.(*ModalTempData)

	// Check if modal state has expired
	if time.Now().After(step1.ExpiresAt) {
		b.modalState.Delete(userID)
		b.followUp(s, i, "Your session expired. Please click **Get Started** again.")
		return
	}
	b.modalState.Delete(userID)

	// Extract Step 2 fields
	homeState := strings.ToUpper(strings.TrimSpace(getModalValue(data, "home_state")))
	roleBackground := getModalValue(data, "role_background")
	goalsVision := getModalValue(data, "goals_vision")
	funHobbies := getModalValue(data, "fun_hobbies")
	phoneRaw := getModalValue(data, "phone_number")
	phone := cleanPhoneNumber(phoneRaw)

	// Parse name into first/last
	firstName, lastName := splitName(step1.FullName)

	// Save to DB and trigger async operations
	if err := b.saveStep2FormData(s, i, step1, userID, firstName, lastName, homeState, roleBackground, goalsVision, funHobbies, phone); err != nil {
		return
	}

	// Send response and optional Step 2b prompt
	b.sendStep2CompleteResponse(s, i, step1.LicenseStatus, step1.CourseEnrolled)
}

