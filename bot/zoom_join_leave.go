package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleZoomJoin(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var verticalID int
	for _, opt := range opts {
		if opt.Name == "id" {
			verticalID = int(opt.IntValue())
		}
	}

	if verticalID == 0 {
		b.followUp(s, i, "Please specify a vertical ID. Use `/zoom list` to see available verticals.")
		return
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	}
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v, err := b.db.GetZoomVertical(ctx, verticalID)
	if err != nil || v == nil {
		b.followUp(s, i, "Vertical not found.")
		return
	}

	if err := b.db.JoinZoomVertical(ctx, userIDInt, verticalID); err != nil {
		log.Printf("zoom join: %v", err)
		b.followUp(s, i, "An error occurred joining the vertical. Please try again later.")
		return
	}

	msg := fmt.Sprintf("You've joined **%s**!", v.Name)
	if v.ZoomLink != "" {
		msg += fmt.Sprintf("\n\nZoom Link: %s", v.ZoomLink)
	}
	if v.Schedule != "" {
		msg += fmt.Sprintf("\nSchedule: %s", v.Schedule)
	}
	b.followUp(s, i, msg)
}

func (b *Bot) handleZoomLeave(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var verticalID int
	for _, opt := range opts {
		if opt.Name == "id" {
			verticalID = int(opt.IntValue())
		}
	}

	if verticalID == 0 {
		b.followUp(s, i, "Please specify a vertical ID.")
		return
	}

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	}
	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.db.LeaveZoomVertical(ctx, userIDInt, verticalID); err != nil {
		log.Printf("zoom leave: %v", err)
		b.followUp(s, i, "An error occurred leaving the vertical. Please try again later.")
		return
	}
	b.followUp(s, i, "You've left the vertical.")
}
