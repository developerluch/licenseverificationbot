package bot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// postHiringLog posts the hiring log embed in #hiring-log.
func (b *Bot) postHiringLog(s *discordgo.Session, member *discordgo.Member, data map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("postHiringLog panic: %v", r)
		}
	}()

	channelID := b.cfg.HiringLogChannelID
	if channelID == "" {
		return
	}

	embed := buildHiringLogEmbed(member, data)
	_, err := s.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Printf("Intake: failed to post hiring log: %v", err)
	}
}

// postGreetingsCard posts the greetings card embed in #greetings.
func (b *Bot) postGreetingsCard(s *discordgo.Session, member *discordgo.Member, data map[string]string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("postGreetingsCard panic: %v", r)
		}
	}()

	channelID := b.cfg.GreetingsChannelID
	if channelID == "" {
		return
	}

	embed, rolePing := buildGreetingsCardEmbed(member, data)

	content := ""
	agencyRoleID := b.cfg.GetAgencyRoleID(data["agency"])
	if agencyRoleID != "" {
		content = fmt.Sprintf("<@&%s> ", agencyRoleID)
	}
	if rolePing != "" {
		content += rolePing
	}

	_, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: content,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		log.Printf("Intake: failed to post greetings card: %v", err)
	}
}
