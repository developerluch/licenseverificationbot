package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// postTicketToThread posts ticket details to the thread and responds to the user.
func (b *Bot) postTicketToThread(s *discordgo.Session, i *discordgo.InteractionCreate,
	thread *discordgo.Channel, userID, userName, subject, category, description, ticketType string, embedColor int) {

	// Add the user to the thread
	s.ThreadMemberAdd(thread.ID, userID)

	// Add staff members to the thread
	staffRoles := b.cfg.StaffRoleIDList()
	members, err := s.GuildMembers(i.GuildID, "", 1000)
	if err == nil {
		for _, m := range members {
			for _, memberRoleID := range m.Roles {
				for _, staffRoleID := range staffRoles {
					if memberRoleID == staffRoleID {
						s.ThreadMemberAdd(thread.ID, m.User.ID)
						break
					}
				}
			}
		}
	}

	// Post the ticket embed
	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("%s Ticket — %s", ticketType, subject),
		Color: embedColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Submitted By", Value: fmt.Sprintf("<@%s>", userID), Inline: true},
			{Name: "Category", Value: category, Inline: true},
			{Name: "Type", Value: ticketType, Inline: true},
			{Name: "Description", Value: description},
		},
		Footer:    &discordgo.MessageEmbedFooter{Text: fmt.Sprintf("Ticket opened by %s", userName)},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	closeBtn := discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Close Ticket",
				Style:    discordgo.DangerButton,
				CustomID: fmt.Sprintf("vipa:ticket_close:%s", thread.ID),
			},
		},
	}

	s.ChannelMessageSendComplex(thread.ID, &discordgo.MessageSend{
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{closeBtn},
	})

	// Respond to the user
	respondEphemeral(s, i, fmt.Sprintf("Your %s ticket has been created: <#%s>\nA staff member will assist you shortly.", ticketType, thread.ID))
}
