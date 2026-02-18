package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/config"
	"license-bot-go/db"
	"license-bot-go/scrapers"
	"license-bot-go/scrapers/captcha"
	"license-bot-go/email"
	"license-bot-go/tlsclient"
)

type Bot struct {
	cfg      *config.Config
	db       *db.DB
	session  *discordgo.Session
	registry *scrapers.Registry
	mailer   *email.Client
}

func New(cfg *config.Config, database *db.DB, tlsClient *tlsclient.Client) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("bot: discordgo session: %w", err)
	}

	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	var cs *captcha.CapSolver
	if cfg.CapSolverAPIKey != "" {
		cs = captcha.NewCapSolver(cfg.CapSolverAPIKey)
	}

	registry := scrapers.NewRegistry(tlsClient, cs)

	mailer := email.NewClient(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
	if mailer != nil {
		log.Println("Resend email client configured")
	}

	return &Bot{
		cfg:      cfg,
		db:       database,
		session:  session,
		registry: registry,
		mailer:   mailer,
	}, nil
}

func (b *Bot) Run(ctx context.Context) error {
	b.session.AddHandler(b.handleInteraction)
	b.session.AddHandler(b.handleMemberUpdate)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("bot: open session: %w", err)
	}

	log.Printf("Bot online as %s#%s", b.session.State.User.Username, b.session.State.User.Discriminator)

	// Start background scheduler for deadline checks + reminders
	go b.StartScheduler(ctx, b.mailer)

	// Register slash commands
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "verify",
			Description: "Verify your insurance license and get promoted to Licensed Agent",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "first_name",
					Description: "Your legal first name (as on your license)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "last_name",
					Description: "Your legal last name (as on your license)",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "state",
					Description: "Your home state (2-letter code, e.g. FL, TX, CA)",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "phone",
					Description: "Your phone number for license update texts (optional)",
					Required:    false,
				},
			},
		},
		{
			Name:        "license-history",
			Description: "View your license check history",
		},
		{
			Name:        "email-optin",
			Description: "Opt in to receive email notifications about your license verification",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "email",
					Description: "Your email address",
					Required:    true,
				},
			},
		},
		{
			Name:        "email-optout",
			Description: "Stop receiving email notifications",
		},
		{
			Name:        "npn",
			Description: "Look up an agent's NPN (National Producer Number) by name",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "first_name",
					Description: "Agent's first name",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "last_name",
					Description: "Agent's last name",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "state",
					Description: "State to search (2-letter code). Leave blank to search all 31+ states",
					Required:    false,
				},
			},
		},
	}

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, b.cfg.GuildID, cmd)
		if err != nil {
			log.Printf("Cannot register command %s: %v", cmd.Name, err)
		}
	}

	log.Printf("Slash commands registered for guild %s", b.cfg.GuildID)

	// Wait for context cancellation (SIGINT/SIGTERM)
	<-ctx.Done()
	log.Println("Shutting down bot...")
	return b.session.Close()
}

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "verify":
		b.handleVerify(s, i)
	case "license-history":
		b.handleHistory(s, i)
	case "email-optin":
		b.handleEmailOptIn(s, i)
	case "email-optout":
		b.handleEmailOptOut(s, i)
	case "npn":
		b.handleNPNLookup(s, i)
	}
}
