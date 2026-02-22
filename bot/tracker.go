package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleTrackerCommand routes /tracker subcommands (Staff only).
func (b *Bot) handleTrackerCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.cfg.IsStaff(i.Member.Roles) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "This command is restricted to staff.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Defer ephemeral response
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	opts := i.ApplicationCommandData().Options
	if len(opts) == 0 {
		b.followUp(s, i, "Please specify a subcommand: overview, agency, or recruiter.")
		return
	}

	sub := opts[0]
	switch sub.Name {
	case "overview":
		b.handleTrackerOverview(s, i)
	case "agency":
		b.handleTrackerAgency(s, i)
	case "recruiter":
		b.handleTrackerRecruiter(s, i, sub.Options)
	}
}

func (b *Bot) handleTrackerOverview(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := b.db.GetOverallTrackerStats(ctx)
	if err != nil {
		log.Printf("tracker overview: %v", err)
		b.followUp(s, i, "An error occurred fetching stats. Please try again later.")
		return
	}

	bar := progressBar(stats.Percentage)
	description := fmt.Sprintf("Licensed Agents: **%d/%d** (%.1f%%)\n\n%s",
		stats.LicensedAgents, stats.TotalAgents, stats.Percentage, bar)

	embed := &discordgo.MessageEmbed{
		Title:       "License Tracker",
		Description: description,
		Color:       0x3498DB,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

func (b *Bot) handleTrackerAgency(s *discordgo.Session, i *discordgo.InteractionCreate) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	agencies, err := b.db.GetAgencyTrackerStats(ctx)
	if err != nil {
		log.Printf("tracker agency: %v", err)
		b.followUp(s, i, "An error occurred fetching agency stats. Please try again later.")
		return
	}

	if len(agencies) == 0 {
		b.followUp(s, i, "No agencies found with assigned agents.")
		return
	}

	var fields []*discordgo.MessageEmbedField
	for _, a := range agencies {
		bar := progressBar(a.Percentage)
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   a.Agency,
			Value:  fmt.Sprintf("%d/%d (%.1f%%) %s", a.LicensedAgents, a.TotalAgents, a.Percentage, bar),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:     "License Tracker \u2014 By Agency",
		Color:     0x3498DB,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

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
		Title:     fmt.Sprintf("License Tracker \u2014 %s Recruiters", agency),
		Color:     0x3498DB,
		Fields:    fields,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
		Flags:  discordgo.MessageFlagsEphemeral,
	})
}

// progressBar builds a 10-segment visual progress bar.
func progressBar(percentage float64) string {
	filled := int(percentage / 10)
	if filled < 0 {
		filled = 0
	}
	if filled > 10 {
		filled = 10
	}
	return strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", 10-filled)
}
