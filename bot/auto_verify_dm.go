package bot

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/scrapers"
)

func (b *Bot) dmUser(s *discordgo.Session, userID, content string) {
	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		log.Printf("Cannot create DM channel for %s: %v", userID, err)
		return
	}
	_, err = s.ChannelMessageSend(channel.ID, content)
	if err != nil {
		log.Printf("Cannot send DM to %s: %v", userID, err)
	}
}

func (b *Bot) dmVerificationSuccess(s *discordgo.Session, e *discordgo.GuildMemberUpdate, match *scrapers.LicenseResult, state string) {
	channel, err := s.UserChannelCreate(e.User.ID)
	if err != nil {
		log.Printf("Cannot create DM channel for auto-verify: %v", err)
		return
	}

	var fields []*discordgo.MessageEmbedField
	fields = append(fields, &discordgo.MessageEmbedField{Name: "Full Name", Value: nvl(match.FullName, "N/A"), Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "State", Value: state, Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "Status", Value: nvl(match.Status, "N/A"), Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "License #", Value: nvl(match.LicenseNumber, "N/A"), Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "NPN", Value: nvl(match.NPN, "N/A"), Inline: true})
	fields = append(fields, &discordgo.MessageEmbedField{Name: "License Type", Value: nvl(match.LicenseType, "N/A"), Inline: true})

	if match.ExpirationDate != "" {
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Expiration Date", Value: match.ExpirationDate, Inline: true})
	}
	if match.LOAs != "" {
		loas := match.LOAs
		if len(loas) > 900 {
			loas = loas[:900] + "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{Name: "Lines of Authority", Value: loas, Inline: false})
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name: "\u200b\nNext Step: Contracting",
		Value: "Use `/contract` in the server to book your contracting appointment.\n\n" +
			"**What to Prepare:**\n" +
			"- Government-issued photo ID\n" +
			"- Social Security number\n" +
			"- E&O insurance info\n" +
			"- Bank info for direct deposit\n" +
			"- Resident state license number",
		Inline: false,
	})

	embed := &discordgo.MessageEmbed{
		Title:       "License Automatically Verified!",
		Description: fmt.Sprintf("Welcome **%s**! Your license was verified automatically. Here are your details:", e.User.Username),
		Color:       0x2ECC71,
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
		Footer:      &discordgo.MessageEmbedFooter{Text: "VIPA License Verification"},
	}

	s.ChannelMessageSendEmbed(channel.ID, embed)
}
