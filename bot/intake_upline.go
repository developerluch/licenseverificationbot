package bot

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

// resolveUplineDiscordID tries to match the upline manager name to a guild member
// and stores the Discord ID for future nudge targeting.
func (b *Bot) resolveUplineDiscordID(s *discordgo.Session, guildID, uplineName string, agentDiscordID, agentGuildID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("resolveUplineDiscordID panic: %v", r)
		}
	}()

	if uplineName == "" {
		return
	}

	// Search guild members by the upline name
	members, err := s.GuildMembersSearch(guildID, uplineName, 5)
	if err != nil {
		log.Printf("Intake: upline search failed for %q: %v", uplineName, err)
		return
	}

	if len(members) == 0 {
		return
	}

	// Pick the best match — exact nickname or username match preferred
	var bestMatch *discordgo.Member
	lowerName := strings.ToLower(uplineName)
	for _, m := range members {
		nick := strings.ToLower(m.Nick)
		uname := strings.ToLower(m.User.Username)
		gname := strings.ToLower(m.User.GlobalName)
		if nick == lowerName || uname == lowerName || gname == lowerName {
			bestMatch = m
			break
		}
	}
	if bestMatch == nil {
		bestMatch = members[0] // Use first result as fallback
	}

	uplineID, err := parseDiscordID(bestMatch.User.ID)
	if err != nil {
		return
	}

	b.db.UpsertAgent(context.Background(), agentDiscordID, agentGuildID, db.AgentUpdate{
		UplineManagerDiscordID: &uplineID,
	})
	log.Printf("Intake: resolved upline %q -> %s (%d)", uplineName, bestMatch.User.Username, uplineID)
}
