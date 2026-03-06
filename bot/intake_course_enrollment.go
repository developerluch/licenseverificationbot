package bot

import (
	"github.com/bwmarrin/discordgo"
)

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
		Title:       "🎓 Course Enrollment Confirmed!",
		Description: "Great — you're on the right track! You'll get access to all training resources.\n\nClick **Continue** below to introduce yourself to the team!",
		Color:       0x2ECC71,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Onboarding • Step 2 is a quick intro — takes 30 seconds!",
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
							Label:    "Continue → Introduce Yourself",
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
		Title:       "📖 Get Enrolled in Your Pre-License Course",
		Description: "Before you can access training resources, you'll need to enroll in a pre-licensing course.\n\n**Book a call with Isabel** to get onboarded and set up with your training:\n\n👉 **[Schedule Enrollment Call](https://link.msgsndr.com/widget/bookings/illuminate-enrollment)**\n\nYou can still continue with onboarding below — once you're enrolled, you'll get full training access!",
		Color:       0xE74C3C,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Onboarding • Book your enrollment call, then continue below",
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
							Label:    "Continue → Introduce Yourself",
							Style:    discordgo.PrimaryButton,
							CustomID: "vipa:step2_continue",
						},
						discordgo.Button{
							Label:    "📅 Book Enrollment Call",
							Style:    discordgo.LinkButton,
							URL:      "https://link.msgsndr.com/widget/bookings/illuminate-enrollment",
						},
					},
				},
			},
		},
	})
}
