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
)

// handleGetStarted responds to the "Get Started" button click by presenting Step 1 modal.
func (b *Bot) handleGetStarted(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Delete the welcome message from #start-here (fire-and-forget)
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	if userID != "" {
		if ref, ok := b.welcomeMessages.LoadAndDelete(userID); ok {
			r := ref.(welcomeMsgRef)
			go func() {
				if err := s.ChannelMessageDelete(r.ChannelID, r.MessageID); err != nil {
					log.Printf("Onboarding: failed to delete welcome msg for %s: %v", userID, err)
				}
			}()
		}
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "vipa:modal_step1",
			Title:    "VIPA Onboarding \u2014 Step 1 of 2",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "full_name",
						Label:       "Full Name",
						Style:       discordgo.TextInputShort,
						Placeholder: "First and Last Name",
						Required:    true,
						MaxLength:   100,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "upline_manager",
						Label:       "Your Upline / Manager",
						Style:       discordgo.TextInputShort,
						Placeholder: "Who recruited you?",
						Required:    true,
						MaxLength:   100,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "agency",
						Label:       "Agency Team",
						Style:       discordgo.TextInputShort,
						Placeholder: "TFC, Radiant, GBU, Illuminate, Synergy, Elite One, etc.",
						Required:    true,
						MaxLength:   50,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "license_status",
						Label:       "License Status",
						Style:       discordgo.TextInputShort,
						Placeholder: "Licensed, Currently Studying, or No License Yet",
						Required:    true,
						MaxLength:   50,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "experience_level",
						Label:       "Experience Level",
						Style:       discordgo.TextInputShort,
						Placeholder: "None, <6 months, 6-12 months, 1-2 years, 2+ years",
						Required:    true,
						MaxLength:   50,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("Intake: failed to show Step 1 modal: %v", err)
	}
}

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
		embed := &discordgo.MessageEmbed{
			Title: "\u2705 Step 1 Complete!",
			Description: fmt.Sprintf(
				"**Name:** %s\n**Agency:** %s\n**License:** %s\n**Experience:** %s\n\n"+
					"\U0001f393 **Are you currently enrolled in a pre-licensing course?**",
				fullName, agency, licenseStatus, expLabel),
			Color: 0xF39C12,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "VIPA Onboarding \u2022 Course enrollment helps us set you up correctly",
			},
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
				Flags:  discordgo.MessageFlagsEphemeral,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.Button{
								Label:    "\u2705 Yes, I'm enrolled",
								Style:    discordgo.SuccessButton,
								CustomID: "vipa:course_enrolled_yes",
							},
							discordgo.Button{
								Label:    "\u274c No, not yet",
								Style:    discordgo.DangerButton,
								CustomID: "vipa:course_enrolled_no",
							},
						},
					},
				},
			},
		})
		if err != nil {
			log.Printf("Intake: failed to show course enrollment question: %v", err)
		}
		return
	}

	// Licensed agents skip straight to Continue
	embed := &discordgo.MessageEmbed{
		Title: "\u2705 Step 1 Complete!",
		Description: fmt.Sprintf(
			"**Name:** %s\n**Agency:** %s\n**License:** %s\n**Experience:** %s\n\n"+
				"Click **Continue** below to answer a few intro questions!",
			fullName, agency, licenseStatus, expLabel),
		Color: 0x2ECC71,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Onboarding \u2022 Step 2 is a quick intro \u2014 takes 30 seconds!",
		},
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Continue \u2192 Introduce Yourself",
							Style:    discordgo.PrimaryButton,
							CustomID: "vipa:step2_continue",
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("Intake: failed to respond to Step 1: %v", err)
	}
}

// handleCourseEnrolledYes handles the "Yes, I'm enrolled" button for unlicensed agents.
func (b *Bot) handleCourseEnrolledYes(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	// Update temp data with course enrollment
	if val, ok := b.modalState.Load(userID); ok {
		if data, ok := val.(*ModalTempData); ok {
			data.CourseEnrolled = true
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "\U0001f393 Course Enrollment Confirmed!",
		Description: "Great \u2014 you're on the right track! You'll get access to all training resources.\n\nClick **Continue** below to introduce yourself to the team!",
		Color:       0x2ECC71,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Onboarding \u2022 Step 2 is a quick intro \u2014 takes 30 seconds!",
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Continue \u2192 Introduce Yourself",
							Style:    discordgo.PrimaryButton,
							CustomID: "vipa:step2_continue",
						},
					},
				},
			},
		},
	})
}

// handleCourseEnrolledNo handles the "No, not yet" button for unlicensed agents.
func (b *Bot) handleCourseEnrolledNo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	// Update temp data — not enrolled
	if val, ok := b.modalState.Load(userID); ok {
		if data, ok := val.(*ModalTempData); ok {
			data.CourseEnrolled = false
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "\U0001f4d6 Get Enrolled in Your Pre-License Course",
		Description: "Before you can access training resources, you'll need to enroll in a pre-licensing course.\n\n**Book a call with Isabel** to get onboarded and set up with your training:\n\n\U0001f449 **[Schedule Enrollment Call](https://link.msgsndr.com/widget/bookings/illuminate-enrollment)**\n\nYou can still continue with onboarding below \u2014 once you're enrolled, you'll get full training access!",
		Color:       0xE74C3C,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Onboarding \u2022 Book your enrollment call, then continue below",
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Continue \u2192 Introduce Yourself",
							Style:    discordgo.PrimaryButton,
							CustomID: "vipa:step2_continue",
						},
						discordgo.Button{
							Label:    "\U0001f4c5 Book Enrollment Call",
							Style:    discordgo.LinkButton,
							URL:      "https://link.msgsndr.com/widget/bookings/illuminate-enrollment",
						},
					},
				},
			},
		},
	})
}

// handleStep2Continue opens the Step 2 modal.
func (b *Bot) handleStep2Continue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "vipa:modal_step2",
			Title:    "Tell Us About Yourself!",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "home_state",
						Label:       "Home State",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g. FL, TX, UT, CA",
						Required:    true,
						MaxLength:   2,
						MinLength:   2,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "role_background",
						Label:       "Your role & what you did before insurance",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g. Producer \u2014 I was a realtor for 5 years",
						Required:    true,
						MaxLength:   200,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "goals_vision",
						Label:       "Your goal & where you'll be in 6 months",
						Style:       discordgo.TextInputParagraph,
						Placeholder: "What are you looking to accomplish?",
						Required:    true,
						MaxLength:   300,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "fun_hobbies",
						Label:       "Fun & Hobbies",
						Style:       discordgo.TextInputParagraph,
						Placeholder: "What do you do for fun?",
						Required:    true,
						MaxLength:   300,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "phone_number",
						Label:       "Phone Number (for license updates)",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g. 555-123-4567",
						Required:    false,
						MaxLength:   20,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("Intake: failed to show Step 2 modal: %v", err)
	}
}

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

	// Determine stage based on license status
	var stage int
	if step1.LicenseStatus == "licensed" {
		stage = db.StageVerified // 5
	} else {
		stage = db.StageStudent // 4
	}

	now := time.Now()

	// Save to DB
	courseEnrolled := step1.CourseEnrolled
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

	// Broadcast form_completed event
	b.publishEvent(websocket.EventFormCompleted, websocket.FormCompletedData{
		DiscordID: userID,
		FullName:  step1.FullName,
		Agency:    step1.Agency,
		License:   step1.LicenseStatus,
	})
	// Broadcast stage_changed event
	b.publishEvent(websocket.EventStageChanged, websocket.StageChangedData{
		DiscordID: userID,
		NewStage:  stage,
		ChangedBy: "onboarding_form",
	})

	// GHL CRM sync (fire-and-forget)
	go b.syncAgentToGHL(userIDInt, guildIDInt)
	go b.syncGHLStage(userIDInt, stage)

	// Resolve upline manager's Discord ID (fire-and-forget)
	go b.resolveUplineDiscordID(s, i.GuildID, step1.UplineManager, userIDInt, guildIDInt)

	// Assign roles — gate through approval if agency owner is configured
	ownerID := b.cfg.GetAgencyOwnerID(step1.Agency)
	if ownerID != "" {
		go b.triggerApprovalFlow(s, userID, i.GuildID, step1.Agency, step1.FullName, step1.LicenseStatus, ownerID)
	} else {
		go b.sortAndAssignRoles(s, userID, i.GuildID, step1.Agency, step1.LicenseStatus)
	}

	// Post hiring log
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

	go b.postHiringLog(s, i.Member, agentData)
	go b.postGreetingsCard(s, i.Member, agentData)

	// Respond to user via followup (deferred above)
	responseMsg := "\u2705 **You're all set!** Welcome to VIPA!"
	if step1.LicenseStatus == "licensed" {
		responseMsg += "\n\nYour license will be verified automatically. Use `/contract` when you're ready to book contracting."

		// Show Step 2b button for licensed agents
		_, followErr := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Content: responseMsg,
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "Add Production Details (Optional)",
							Style:    discordgo.SecondaryButton,
							CustomID: "vipa:step2b_continue",
						},
					},
				},
			},
		})
		if followErr != nil {
			log.Printf("handleStep2Submit: followup failed: %v", followErr)
		}
	} else {
		if courseEnrolled {
			responseMsg += "\n\n\U0001f393 You're enrolled in your pre-licensing course \u2014 you have access to all training resources!"
			responseMsg += "\nOnce you pass your exam, use `/verify` to verify your license."
		} else {
			responseMsg += "\n\n\U0001f4d6 **Don't forget to enroll in your pre-licensing course!**"
			responseMsg += "\n\U0001f449 [Book your enrollment call with Isabel](https://link.msgsndr.com/widget/bookings/illuminate-enrollment)"
			responseMsg += "\nOnce enrolled and licensed, use `/verify` to verify your license."
		}
		b.followUp(s, i, responseMsg)
	}
}

// handleStep2bContinue opens the optional Step 2b modal for licensed agents.
func (b *Bot) handleStep2bContinue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "vipa:modal_step2b",
			Title:    "Almost Done \u2014 Production Details",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "production_written",
						Label:       "Previous Production Written (monthly avg)",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g., $5,000 AP/month, 10 apps/week, or 'Just starting'",
						Required:    false,
						MaxLength:   200,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "lead_source",
						Label:       "Lead Source",
						Style:       discordgo.TextInputShort,
						Placeholder: "Buy own leads, Agency funded, Both, or Other",
						Required:    false,
						MaxLength:   50,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "vision_goals",
						Label:       "Your Vision / Goals at VIPA",
						Style:       discordgo.TextInputParagraph,
						Placeholder: "What are you looking to accomplish?",
						Required:    false,
						MaxLength:   1000,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "comp_pct",
						Label:       "Compensation % Given (PRIVATE)",
						Style:       discordgo.TextInputShort,
						Placeholder: "e.g., 80%, 90%, 110% \u2014 NEVER shared publicly",
						Required:    false,
						MaxLength:   20,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "show_comp",
						Label:       "Show comp on your profile? (yes / no)",
						Style:       discordgo.TextInputShort,
						Placeholder: "yes or no \u2014 Default: no",
						Required:    false,
						MaxLength:   5,
					},
				}},
			},
		},
	})
	if err != nil {
		log.Printf("Intake: failed to show Step 2b modal: %v", err)
	}
}

// handleStep2bSubmit processes the optional production details modal.
func (b *Bot) handleStep2bSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ModalSubmitData()
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("handleStep2bSubmit: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleStep2bSubmit: %v", err)
		return
	}

	production := getModalValue(data, "production_written")
	leadSource := normalizeLeadSource(getModalValue(data, "lead_source"))
	vision := getModalValue(data, "vision_goals")
	comp := getModalValue(data, "comp_pct")
	showCompRaw := getModalValue(data, "show_comp")
	showComp := normalizeShowComp(showCompRaw)

	update := db.AgentUpdate{
		ProductionWritten: &production,
		LeadSource:        &leadSource,
		VisionGoals:       &vision,
		CompPct:           &comp,
		ShowComp:          &showComp,
	}

	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, update)
	b.db.LogActivity(context.Background(), userIDInt, "form_step2b", "Production details submitted")

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "\u2705 Production details saved! Use `/contract` to book your contracting appointment.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// handleRestart reopens the onboarding form for a user (staff only).
func (b *Bot) handleRestart(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.cfg.IsStaff(i.Member.Roles) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is restricted to staff.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		return
	}
	targetUser := opts[0].UserValue(s)
	if targetUser == nil {
		return
	}

	// Reset stage to 1 so they can re-do intake
	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleRestart: %v", err)
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		log.Printf("handleRestart: %v", err)
		return
	}
	stage := db.StageWelcome
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		CurrentStage: &stage,
	})
	b.db.LogActivity(context.Background(), userIDInt, "restart", "Onboarding restarted by staff")

	// Send welcome DM to the target user
	channel, err := s.UserChannelCreate(targetUser.ID)
	if err == nil {
		embed := buildWelcomeEmbed()
		s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{embed},
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
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Onboarding restarted for <@%s>. They've been sent a new welcome DM.", targetUser.ID),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// sortAndAssignRoles assigns @Student + agency role + @Onboarded, and optionally @Licensed-Agent.
func (b *Bot) sortAndAssignRoles(s *discordgo.Session, userID, guildID, agency, licenseStatus string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("sortAndAssignRoles panic: %v", r)
		}
	}()

	// Remove @New role (join-gate) and assign @Onboarded role (unlock all channels)
	if b.cfg.NewRoleID != "" {
		if err := s.GuildMemberRoleRemove(guildID, userID, b.cfg.NewRoleID); err != nil {
			log.Printf("Intake: failed to remove @New role from %s: %v", userID, err)
		}
	}
	if b.cfg.OnboardedRoleID != "" {
		if err := s.GuildMemberRoleAdd(guildID, userID, b.cfg.OnboardedRoleID); err != nil {
			log.Printf("Intake: failed to add Onboarded role to %s: %v", userID, err)
		}
	}

	// Always assign Student role first
	if b.cfg.StudentRoleID != "" {
		if err := s.GuildMemberRoleAdd(guildID, userID, b.cfg.StudentRoleID); err != nil {
			log.Printf("Intake: failed to add Student role to %s: %v", userID, err)
		}
	}

	// Assign agency role
	agencyRoleID := b.cfg.GetAgencyRoleID(agency)
	if agencyRoleID != "" {
		if err := s.GuildMemberRoleAdd(guildID, userID, agencyRoleID); err != nil {
			log.Printf("Intake: failed to add agency role %s to %s: %v", agencyRoleID, userID, err)
		}
	}

	// If licensed, also assign Licensed-Agent and remove Student
	if licenseStatus == "licensed" {
		if b.cfg.LicensedAgentRoleID != "" {
			if err := s.GuildMemberRoleAdd(guildID, userID, b.cfg.LicensedAgentRoleID); err != nil {
				log.Printf("Intake: failed to add Licensed-Agent role to %s: %v", userID, err)
			}
		}
		if b.cfg.StudentRoleID != "" {
			if err := s.GuildMemberRoleRemove(guildID, userID, b.cfg.StudentRoleID); err != nil {
				log.Printf("Intake: failed to remove Student role from %s: %v", userID, err)
			}
		}
	}
}

// postHiringLog posts the hiring log embed in #hiring-log.
func (b *Bot) postHiringLog(s *discordgo.Session, member *discordgo.Member, data map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("postHiringLog panic: %v", r)
		}
	}()

	channelID := b.cfg.HiringLogChannelID
	if channelID == "" {
		return
	}

	embed := buildHiringLogEmbed(member, data)
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Intake: failed to post hiring log: %v", err)
	}
}

// postGreetingsCard posts the greetings card embed in #greetings.
func (b *Bot) postGreetingsCard(s *discordgo.Session, member *discordgo.Member, data map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("postGreetingsCard panic: %v", r)
		}
	}()

	channelID := b.cfg.GreetingsChannelID
	if channelID == "" {
		return
	}

	embed, rolePing := buildGreetingsCardEmbed(member, data)

	content := ""
	agencyRoleID := b.cfg.GetAgencyRoleID(data["agency"])
	if agencyRoleID != "" {
		content = fmt.Sprintf("<@&%s> ", agencyRoleID)
	}
	if rolePing != "" {
		content += rolePing
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("Intake: failed to post greetings card: %v", err)
	}
}

// resolveUplineDiscordID tries to match the upline manager name to a guild member
// and stores the Discord ID for future nudge targeting.
func (b *Bot) resolveUplineDiscordID(s *discordgo.Session, guildID, uplineName string, agentDiscordID, agentGuildID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("resolveUplineDiscordID panic: %v", r)
		}
	}()

	if uplineName == "" {
		return
	}

	// Search guild members by the upline name
	members, err := s.GuildMembersSearch(guildID, uplineName, 5)
	if err != nil {
		log.Printf("Intake: upline search failed for %q: %v", uplineName, err)
		return
	}

	if len(members) == 0 {
		return
	}

	// Pick the best match — exact nickname or username match preferred
	var bestMatch *discordgo.Member
	lowerName := strings.ToLower(uplineName)
	for _, m := range members {
		nick := strings.ToLower(m.Nick)
		uname := strings.ToLower(m.User.Username)
		gname := strings.ToLower(m.User.GlobalName)
		if nick == lowerName || uname == lowerName || gname == lowerName {
			bestMatch = m
			break
		}
	}
	if bestMatch == nil {
		bestMatch = members[0] // Use first result as fallback
	}

	uplineID, err := parseDiscordID(bestMatch.User.ID)
	if err != nil {
		return
	}

	b.db.UpsertAgent(context.Background(), agentDiscordID, agentGuildID, db.AgentUpdate{
		UplineManagerDiscordID: &uplineID,
	})
	log.Printf("Intake: resolved upline %q -> %s (%d)", uplineName, bestMatch.User.Username, uplineID)
}

// === Normalization functions ===

func normalizeAgency(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case lower == "tfc" || lower == "topfloorclosers" || lower == "top floor closers":
		return "TFC"
	case lower == "radiant" || lower == "radiant financial":
		return "Radiant"
	case lower == "gbu":
		return "GBU"
	case strings.Contains(lower, "trulight") || strings.Contains(lower, "tru light"):
		return "TruLight"
	case strings.Contains(lower, "thrive"):
		return "Thrive"
	case strings.Contains(lower, "the point") || lower == "thepoint":
		return "The Point"
	case strings.Contains(lower, "synergy"):
		return "Synergy"
	case strings.Contains(lower, "illuminate"):
		return "Illuminate"
	case strings.Contains(lower, "elite one") || strings.Contains(lower, "elite 1") || lower == "eliteone":
		return "Elite One"
	default:
		return "Other"
	}
}

func normalizeLicenseStatus(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(lower, "licensed") || lower == "yes":
		return "licensed"
	case strings.Contains(lower, "study") || strings.Contains(lower, "studying"):
		return "studying"
	default:
		return "none"
	}
}

func normalizeExperience(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(lower, "2+") || strings.Contains(lower, "2 year") || strings.Contains(lower, "2yr"):
		return "2yr_plus"
	case strings.Contains(lower, "1-2") || strings.Contains(lower, "1 to 2") || strings.Contains(lower, "1yr"):
		return "1_2yr"
	case strings.Contains(lower, "6-12") || strings.Contains(lower, "6 to 12"):
		return "6_12mo"
	case strings.Contains(lower, "<6") || strings.Contains(lower, "less than 6") || strings.Contains(lower, "6 mo"):
		return "less_6mo"
	default:
		return "none"
	}
}

func normalizeLeadSource(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(lower, "both"):
		return "both"
	case strings.Contains(lower, "buy") || strings.Contains(lower, "own"):
		return "buy_own"
	case strings.Contains(lower, "agency") || strings.Contains(lower, "funded"):
		return "agency_funded"
	default:
		if raw == "" {
			return ""
		}
		return raw
	}
}

func normalizeShowComp(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return lower == "yes" || lower == "true" || lower == "1"
}

// splitName splits a full name into first and last name.
func splitName(fullName string) (string, string) {
	parts := strings.Fields(strings.TrimSpace(fullName))
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

// getModalValue extracts a text input value from modal submit data.
func getModalValue(data discordgo.ModalSubmitInteractionData, customID string) string {
	for _, row := range data.Components {
		for _, comp := range row.(*discordgo.ActionsRow).Components {
			if ti, ok := comp.(*discordgo.TextInput); ok && ti.CustomID == customID {
				return ti.Value
			}
		}
	}
	return ""
}
