package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleAgentNudge(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	if len(opts) == 0 {
		return
	}
	targetUser := opts[0].UserValue(s)
	if targetUser == nil {
		return
	}

	userIDInt, err := parseDiscordID(targetUser.ID)
	if err != nil {
		log.Printf("handleAgentNudge: %v", err)
		return
	}
	agent, _ := b.db.GetAgent(context.Background(), userIDInt)
	name := "Agent"
	if agent != nil && agent.FirstName != "" {
		name = agent.FirstName
	}

	weeksIn := 1
	if agent != nil {
		weeksIn = int(time.Since(agent.CreatedAt).Hours() / (24 * 7))
		if weeksIn < 1 {
			weeksIn = 1
		}
	}

	embed := buildCheckinEmbed(name, weeksIn)
	weekStartStr := time.Now().Truncate(24 * time.Hour).Format("2006-01-02")

	channel, err := s.UserChannelCreate(targetUser.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("Cannot DM <@%s>: %v", targetUser.ID, err),
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	_, sendErr := s.ChannelMessageSendComplex(channel.ID, &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{embed},
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "✅ On Track",
						Style:    discordgo.SuccessButton,
						CustomID: fmt.Sprintf("vipa:checkin:on_track:%s", weekStartStr),
					},
					discordgo.Button{
						Label:    "⏸️ Need Help",
						Style:    discordgo.SecondaryButton,
						CustomID: fmt.Sprintf("vipa:checkin:need_help:%s", weekStartStr),
					},
					discordgo.Button{
						Label:    "🎓 Got Licensed!",
						Style:    discordgo.PrimaryButton,
						CustomID: fmt.Sprintf("vipa:checkin:got_licensed:%s", weekStartStr),
					},
				},
			},
		},
	})

	b.db.LogActivity(context.Background(), userIDInt, "nudge", "Manual check-in sent by staff")

	msg := fmt.Sprintf("Check-in DM sent to <@%s>.", targetUser.ID)
	if sendErr != nil {
		msg = fmt.Sprintf("Failed to send DM to <@%s>: %v", targetUser.ID, sendErr)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
