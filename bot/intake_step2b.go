package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// handleStep2bContinue opens the optional Step 2b modal for licensed agents.
func (b *Bot) handleStep2bContinue(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "vipa:modal_step2b",
			Title:    "Almost Done — Production Details",
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
						Placeholder: "e.g., 80%, 90%, 110% — NEVER shared publicly",
						Required:    false,
						MaxLength:   20,
					},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{
						CustomID:    "show_comp",
						Label:       "Show comp on your profile? (yes / no)",
						Style:       discordgo.TextInputShort,
						Placeholder: "yes or no — Default: no",
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
