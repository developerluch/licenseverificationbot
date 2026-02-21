package bot

import (
	"context"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/db"
)

func (b *Bot) handleEmailOptIn(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Defer failed: %v", err)
		return
	}

	opts := i.ApplicationCommandData().Options
	emailAddr := ""
	for _, opt := range opts {
		if opt.Name == "email" {
			emailAddr = strings.TrimSpace(opt.StringValue())
		}
	}

	if emailAddr == "" || !strings.Contains(emailAddr, "@") || !strings.Contains(emailAddr, ".") {
		b.followUp(s, i, "Please provide a valid email address.\nExample: `/email-optin email:you@example.com`")
		return
	}

	userIDInt, err := parseDiscordID(i.Member.User.ID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}

	optIn := true
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		Email:      &emailAddr,
		EmailOptIn: &optIn,
	})

	b.followUp(s, i, "**Email notifications enabled!**\n\n"+
		"You'll receive email reminders at **"+emailAddr+"** about:\n"+
		"- License verification deadlines\n"+
		"- Verification status updates\n\n"+
		"Use `/email-optout` anytime to unsubscribe.")
}

func (b *Bot) handleEmailOptOut(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Printf("Defer failed: %v", err)
		return
	}

	userIDInt, err := parseDiscordID(i.Member.User.ID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}
	guildIDInt, err := parseDiscordID(i.GuildID)
	if err != nil {
		b.followUp(s, i, "Internal error. Please try again.")
		return
	}

	optOut := false
	b.db.UpsertAgent(context.Background(), userIDInt, guildIDInt, db.AgentUpdate{
		EmailOptIn: &optOut,
	})

	b.followUp(s, i, "**Email notifications disabled.**\n\n"+
		"You'll no longer receive email reminders. You'll still get Discord DMs.\n"+
		"Use `/email-optin email:you@example.com` to re-enable anytime.")
}
