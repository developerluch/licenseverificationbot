package bot

import (
	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// buildSetupButtons creates the button rows for the setup checklist.
func buildSetupButtons(progress map[string]bool) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent
	for _, item := range db.SetupItems {
		style := discordgo.SecondaryButton
		label := item.Label
		disabled := false
		if progress[item.Key] {
			style = discordgo.SuccessButton
			label = "\u2705 " + item.Label
			disabled = true
		}
		buttons = append(buttons, discordgo.Button{
			Label:    label,
			Style:    style,
			CustomID: "vipa:setup:" + item.Key,
			Disabled: disabled,
		})
	}

	// Discord allows max 5 buttons per row
	var rows []discordgo.MessageComponent
	for j := 0; j < len(buttons); j += 5 {
		end := j + 5
		if end > len(buttons) {
			end = len(buttons)
		}
		rows = append(rows, discordgo.ActionsRow{Components: buttons[j:end]})
	}

	// Add "Complete Setup" button if all items done
	allDone := true
	for _, item := range db.SetupItems {
		if !progress[item.Key] {
			allDone = false
			break
		}
	}
	if allDone {
		rows = append(rows, discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "\U0001f389 Complete Setup",
					Style:    discordgo.SuccessButton,
					CustomID: "vipa:setup_complete_all",
				},
			},
		})
	}

	return rows
}
