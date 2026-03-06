package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

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
						Placeholder: "e.g. Producer — I was a realtor for 5 years",
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
