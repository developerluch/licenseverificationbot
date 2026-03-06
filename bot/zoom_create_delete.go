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

func (b *Bot) handleZoomCreate(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var name, description, zoomLink, schedule string
	for _, opt := range opts {
		switch opt.Name {
		case "name":
			name = opt.StringValue()
		case "description":
			description = opt.StringValue()
		case "zoom_link":
			zoomLink = opt.StringValue()
		case "schedule":
			schedule = opt.StringValue()
		}
	}

	if name == "" {
		b.followUp(s, i, "Name is required.")
		return
	}

	if zoomLink != "" && !strings.HasPrefix(zoomLink, "https://") {
		b.followUp(s, i, "Zoom link must start with https://")
		return
	}

	creatorID := ""
	if i.Member != nil {
		creatorID = i.Member.User.ID
	}
	creatorIDInt, err := parseDiscordID(creatorID)
	if err != nil {
		log.Printf("zoom create: invalid creator ID %s: %v", creatorID, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := b.db.CreateZoomVertical(ctx, db.ZoomVertical{
		Name:        name,
		Description: description,
		ZoomLink:    zoomLink,
		Schedule:    schedule,
		CreatedBy:   creatorIDInt,
	})
	if err != nil {
		log.Printf("zoom create: %v", err)
		b.followUp(s, i, "An error occurred creating the vertical. Please try again later.")
		return
	}

	b.followUp(s, i, fmt.Sprintf("Created zoom vertical **%s** (ID: %d)", name, id))
}

func (b *Bot) handleZoomDelete(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	b.db.DeleteZoomVertical(ctx, verticalID)
	b.followUp(s, i, fmt.Sprintf("Vertical #%d has been deactivated.", verticalID))
}
