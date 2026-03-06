package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func (b *Bot) handleZoomList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	verticals, err := b.db.GetZoomVerticals(ctx)
	if err != nil {
		log.Printf("zoom list: %v", err)
		b.followUp(s, i, "An error occurred. Please try again later.")
		return
	}

	if len(verticals) == 0 {
		b.followUp(s, i, "No zoom verticals available.")
		return
	}

	// Get user's joined verticals
	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	}
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		log.Printf("zoom list: invalid user ID %s: %v", userID, err)
	}
	var myVerticals []db.ZoomVertical
	if userIDInt != 0 {
		myVerticals, _ = b.db.GetUserZoomVerticals(ctx, userIDInt)
	}
	mySet := make(map[int]bool)
	for _, v := range myVerticals {
		mySet[v.ID] = true
	}

	var lines []string
	for _, v := range verticals {
		joined := ""
		if mySet[v.ID] {
			joined = " [JOINED]"
		}
		line := fmt.Sprintf("**%d. %s**%s", v.ID, v.Name, joined)
		if v.Description != "" {
			line += "\n   " + v.Description
		}
		if v.Schedule != "" {
			line += "\n   Schedule: " + v.Schedule
		}
		lines = append(lines, line)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Zoom Verticals",
		Description: strings.Join(lines, "\n\n"),
		Color:       0x3498DB,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /zoom join id:<number> to join a vertical",
		},
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}
