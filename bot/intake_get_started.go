package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
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
			Title:    "VIPA Onboarding — Step 1 of 2",
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
