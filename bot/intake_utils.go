package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

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
