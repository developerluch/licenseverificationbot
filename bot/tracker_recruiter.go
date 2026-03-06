package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleTrackerRecruiter(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	var agency string
	for _, opt := range opts {
		if opt.Name == "agency" {
			agency = opt.StringValue()
		}
	}

	if agency == "" {
		b.followUp(s, i, "Please specify an agency name.")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	recruiters, err := b.db.GetRecruiterTrackerStats(ctx, agency)
	if err != nil {
		log.Printf("tracker recruiter: %v", err)
		b.followUp(s, i, "An error occurred fetching recruiter stats. Please try again later.")
		return
	}

	if len(recruiters) == 0 {
		b.followUp(s, i, fmt.Sprintf("No recruiters found for agency **%s**.", agency))
		return
	}

	var fields []*discordgo.MessageEmbedField
	for _, r := range recruiters {
		name := r.RecruiterName
		if r.RecruiterDiscordID != 0 {
			name = fmt.Sprintf("%s (<@%d>)", r.RecruiterName, r.RecruiterDiscordID)
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   name,
			Value:  fmt.Sprintf("Licensed: %d/%d", r.LicensedRecruits, r.TotalRecruits),
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:     fmt.Sprintf("License Tracker — %s Recruiters", agency),
		Color:     0x3498DB,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}
