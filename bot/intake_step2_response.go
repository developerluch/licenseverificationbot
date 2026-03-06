package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// sendStep2CompleteResponse sends the completion message with optional Step 2b prompt.
func (b *Bot) sendStep2CompleteResponse(s *discordgo.Session, i *discordgo.InteractionCreate, licenseStatus string, courseEnrolled bool) {
	responseMsg := "✅ **You're all set!** Welcome to VIPA!"
	if licenseStatus == "licensed" {
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
			responseMsg += "\n\n🎓 You're enrolled in your pre-licensing course — you have access to all training resources!"
			responseMsg += "\nOnce you pass your exam, use `/verify` to verify your license."
		} else {
			responseMsg += "\n\n📖 **Don't forget to enroll in your pre-licensing course!**"
			responseMsg += "\n👉 [Book your enrollment call with Isabel](https://link.msgsndr.com/widget/bookings/illuminate-enrollment)"
			responseMsg += "\nOnce enrolled and licensed, use `/verify` to verify your license."
		}
		b.followUp(s, i, responseMsg)
	}
}
