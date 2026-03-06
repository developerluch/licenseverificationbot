package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// showCourseEnrollmentPrompt displays the course enrollment question for unlicensed agents.
func (b *Bot) showCourseEnrollmentPrompt(s *discordgo.Session, i *discordgo.InteractionCreate, fullName, agency, licenseStatus, expLabel string) {
	embed := &discordgo.MessageEmbed{
		Title: "✅ Step 1 Complete!",
		Description: fmt.Sprintf(
			"**Name:** %s\n**Agency:** %s\n**License:** %s\n**Experience:** %s\n\n"+
				"🎓 **Are you currently enrolled in a pre-licensing course?**",
			fullName, agency, licenseStatus, expLabel),
		Color: 0xF39C12,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Onboarding • Course enrollment helps us set you up correctly",
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "✅ Yes, I'm enrolled",
							Style:    discordgo.SuccessButton,
							CustomID: "vipa:course_enrolled_yes",
						},
						discordgo.Button{
							Label:    "❌ No, not yet",
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
}

// showLicensedAgentConfirmation displays the continuation prompt for licensed agents.
func (b *Bot) showLicensedAgentConfirmation(s *discordgo.Session, i *discordgo.InteractionCreate, fullName, agency, licenseStatus, expLabel string) {
	embed := &discordgo.MessageEmbed{
		Title: "✅ Step 1 Complete!",
		Description: fmt.Sprintf(
			"**Name:** %s\n**Agency:** %s\n**License:** %s\n**Experience:** %s\n\n"+
				"Click **Continue** below to answer a few intro questions!",
			fullName, agency, licenseStatus, expLabel),
		Color: 0x2ECC71,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "VIPA Onboarding • Step 2 is a quick intro — takes 30 seconds!",
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
	if err != nil {
		log.Printf("Intake: failed to respond to Step 1: %v", err)
	}
}
