package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// postDailyTracker posts the overall tracker stats to the configured tracker channel.
func (b *Bot) postDailyTracker(ctx context.Context) {
	channelID := b.cfg.TrackerChannelID
	if channelID == "" {
		return
	}

	stats, err := b.db.GetOverallTrackerStats(ctx)
	if err != nil {
		log.Printf("Scheduler: failed to get tracker stats: %v", err)
		return
	}

	bar := progressBar(stats.Percentage)

	agencies, _ := b.db.GetAgencyTrackerStats(ctx)
	var agencyLines []string
	for _, a := range agencies {
		agencyLines = append(agencyLines, fmt.Sprintf("**%s:** %d/%d (%.0f%%)",
			a.Agency, a.LicensedAgents, a.TotalAgents, a.Percentage))
	}

	description := fmt.Sprintf(
		"Licensed: **%d/%d** (%.1f%%)\n%s",
		stats.LicensedAgents, stats.TotalAgents, stats.Percentage, bar)

	if len(agencyLines) > 0 {
		description += "\n\n" + strings.Join(agencyLines, "\n")
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Daily License Tracker",
		Description: description,
		Color:       0x3498DB,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	b.session.ChannelMessageSendEmbed(channelID, embed)
	log.Println("Scheduler: posted daily tracker update")
}
