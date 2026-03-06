package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// sendRecruiterNudges DMs recruiters about their unlicensed recruits past the nudge threshold.
func (b *Bot) sendRecruiterNudges(ctx context.Context) {
	agents, err := b.db.GetAgentsNeedingNudge(ctx, b.cfg.NudgeAfterDays, 7)
	if err != nil {
		log.Printf("Scheduler: failed to get agents needing nudge: %v", err)
		return
	}
	if len(agents) == 0 {
		return
	}

	// Group by upline
	type nudgeGroup struct {
		recruiterDiscordID int64
		recruiterName      string
		agentNames         []string
	}
	groups := make(map[string]*nudgeGroup)

	for _, a := range agents {
		key := a.UplineManager
		if key == "" {
			key = "__unassigned__"
		}
		g, ok := groups[key]
		if !ok {
			g = &nudgeGroup{
				recruiterDiscordID: a.UplineManagerDiscordID,
				recruiterName:      a.UplineManager,
			}
			groups[key] = g
		}
		name := a.FirstName + " " + a.LastName
		if name == " " {
			name = fmt.Sprintf("Agent #%d", a.DiscordID)
		}
		g.agentNames = append(g.agentNames, name)
	}

	for _, g := range groups {
		var lines []string
		for _, name := range g.agentNames {
			lines = append(lines, "- "+name)
		}
		msg := fmt.Sprintf(
			"**Recruiter Nudge:** The following recruits have been unlicensed for %d+ days:\n\n%s\n\n"+
				"Please follow up with them to check on their licensing progress.",
			b.cfg.NudgeAfterDays, strings.Join(lines, "\n"))

		if g.recruiterDiscordID != 0 {
			recruiterID := strconv.FormatInt(g.recruiterDiscordID, 10)
			b.dmUser(b.session, recruiterID, msg)
			log.Printf("Scheduler: nudged recruiter %s (%d) about %d unlicensed agents",
				g.recruiterName, g.recruiterDiscordID, len(g.agentNames))
		} else {
			channelID := b.cfg.AdminNotificationChannelID
			if channelID == "" {
				channelID = b.cfg.LicenseCheckChannelID
			}
			if channelID != "" {
				embed := &discordgo.MessageEmbed{
					Title:       fmt.Sprintf("Recruiter Nudge: %s", nvl(g.recruiterName, "Unknown")),
					Description: msg,
					Color:       0xF39C12,
					Timestamp:   time.Now().Format(time.RFC3339),
				}
				b.session.ChannelMessageSendEmbed(channelID, embed)
			}
		}
	}

	// Mark nudges sent
	for _, a := range agents {
		b.db.UpdateNudgeSent(ctx, a.DiscordID)
	}
}
