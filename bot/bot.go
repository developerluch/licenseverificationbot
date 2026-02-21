package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"license-bot-go/config"
	"license-bot-go/db"
	"license-bot-go/email"
	"license-bot-go/scrapers"
	"license-bot-go/scrapers/captcha"
	"license-bot-go/tlsclient"
)

// ModalTempData holds temporary form data between Step 1 and Step 2 modals.
type ModalTempData struct {
	FullName        string
	Agency          string
	UplineManager   string
	ExperienceLevel string
	LicenseStatus   string
	ExpiresAt       time.Time
}

type Bot struct {
	cfg        *config.Config
	db         *db.DB
	session    *discordgo.Session
	registry   *scrapers.Registry
	mailer     *email.Client
	modalState sync.Map // userID (string) -> *ModalTempData
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
	b.session.AddHandler(b.handleMemberJoin)

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("bot: open session: %w", err)
	}

	log.Printf("Bot online as %s#%s", b.session.State.User.Username, b.session.State.User.Discriminator)

	// Start background scheduler for deadline checks + reminders + checkins
	go b.StartScheduler(ctx, b.mailer)

	// Start modal state TTL cleanup
	go b.cleanupModalState(ctx)

	// Register slash commands
	b.registerCommands()

	// Wait for context cancellation (SIGINT/SIGTERM)
	<-ctx.Done()
	log.Println("Shutting down bot...")
	return b.session.Close()
}

// handleInteraction routes all Discord interactions by type.
func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleCommand(s, i)
	case discordgo.InteractionMessageComponent:
		b.handleComponent(s, i)
	case discordgo.InteractionModalSubmit:
		b.handleModalSubmit(s, i)
	}
}

// handleCommand routes slash commands.
func (b *Bot) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Name {
	// Existing commands (DO NOT TOUCH)
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
	// Onboarding commands
	case "contract":
		b.handleContract(s, i)
	case "setup":
		b.handleSetup(s, i)
	// Admin/staff commands
	case "agent":
		b.handleAgentCommand(s, i)
	case "contracting":
		b.handleContractingCommand(s, i)
	case "restart":
		b.handleRestart(s, i)
	case "onboarding-setup":
		b.handleOnboardingSetup(s, i)
	case "setup-rules":
		b.handleSetupRules(s, i)
	}
}

// handleComponent routes button clicks and other component interactions.
func (b *Bot) handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.MessageComponentData().CustomID

	switch {
	// Onboarding buttons
	case customID == "vipa:onboarding_get_started":
		b.handleGetStarted(s, i)
	case customID == "vipa:step2_continue":
		b.handleStep2Continue(s, i)
	case customID == "vipa:step2b_continue":
		b.handleStep2bContinue(s, i)

	// Check-in buttons (vipa:checkin:{action}:{week_start})
	case strings.HasPrefix(customID, "vipa:checkin:"):
		b.handleCheckinResponse(s, i)

	// Setup checklist buttons (vipa:setup:{item_key} or vipa:setup_complete_all)
	case customID == "vipa:setup_complete_all":
		b.handleSetupCompleteAll(s, i)
	case strings.HasPrefix(customID, "vipa:setup:"):
		b.handleSetupItem(s, i)
	}
}

// handleModalSubmit routes modal form submissions.
func (b *Bot) handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	customID := i.ModalSubmitData().CustomID

	switch customID {
	case "vipa:modal_step1":
		b.handleStep1Submit(s, i)
	case "vipa:modal_step2":
		b.handleStep2Submit(s, i)
	case "vipa:modal_step2b":
		b.handleStep2bSubmit(s, i)
	}
}

// cleanupModalState removes expired modal temp data every 5 minutes.
func (b *Bot) cleanupModalState(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.modalState.Range(func(key, value interface{}) bool {
				data, ok := value.(*ModalTempData)
				if ok && time.Now().After(data.ExpiresAt) {
					b.modalState.Delete(key)
				}
				return true
			})
		}
	}
}

// parseDiscordID converts a Discord snowflake ID string to int64.
func parseDiscordID(id string) (int64, error) {
	n, err := strconv.ParseInt(id, 10, 64)
	if err != nil || n == 0 {
		return 0, fmt.Errorf("invalid discord ID: %s", id)
	}
	return n, nil
}
