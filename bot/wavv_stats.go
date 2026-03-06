package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleWavvStatsCommand shows an agent's WAVV production stats.
func (b *Bot) handleWavvStatsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})

	userID := ""
	if i.Member != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}

	userIDInt, err := parseDiscordID(userID)
	if err != nil {
		b.followUp(s, i, "Internal error.")
		return
	}

	// Parse period
	period := "week"
	opts := i.ApplicationCommandData().Options
	for _, opt := range opts {
		if opt.Name == "period" {
			period = opt.StringValue()
		}
	}

	now := time.Now()
	var from, to time.Time
	var label string

	switch period {
	case "month":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 1, -1)
		label = now.Format("January 2006")
	case "7d":
		from = now.AddDate(0, 0, -7)
		to = now
		label = "Last 7 Days"
	case "30d":
		from = now.AddDate(0, 0, -30)
		to = now
		label = "Last 30 Days"
	default: // week
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from = now.AddDate(0, 0, -(weekday - 1))
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 0, 6)
		label = "This Week"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := b.db.GetWavvAgentStats(ctx, userIDInt, from, to)
	if err != nil {
		b.followUp(s, i, "No WAVV sessions found for this period. Use `/wavv-log` to log a session!")
		return
	}

	connectRate := 0.0
	if stats.Dials > 0 {
		connectRate = float64(stats.Connections) / float64(stats.Dials) * 100
	}

	msg := fmt.Sprintf("**WAVV Production — %s** 📊\n\n"+
		"📞 **Dials:** %d\n"+
		"🔗 **Connections:** %d (%.1f%% rate)\n"+
		"🗣️ **Talk Time:** %d min\n"+
		"📅 **Appointments:** %d\n"+
		"🔄 **Callbacks:** %d\n"+
		"📋 **Policies:** %d\n"+
		"📝 **Sessions:** %d",
		label, stats.Dials, stats.Connections, connectRate,
		stats.TalkTimeMins, stats.Appointments, stats.Callbacks,
		stats.Policies, stats.SessionCount)

	b.followUp(s, i, msg)
}
