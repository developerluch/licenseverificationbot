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

	"license-bot-go/api/websocket"
	"license-bot-go/config"
	"license-bot-go/db"
	"license-bot-go/email"
	"license-bot-go/ghl"
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
	CourseEnrolled  bool
	ExpiresAt       time.Time
}

type Bot struct {
	cfg              *config.Config
	db               *db.DB
	session          *discordgo.Session
	registry         *scrapers.Registry
	mailer           *email.Client
	ghlClient        *ghl.Client
	hub              *websocket.Hub
	modalState       sync.Map // userID (string) -> *ModalTempData
	welcomeMessages  sync.Map // userID (string) -> welcomeMsgRef{channelID, messageID}
}

// welcomeMsgRef stores the channel and message ID for a user's welcome message in #start-here.
type welcomeMsgRef struct {
	ChannelID string
	MessageID string
}

func New(cfg *config.Config, database *db.DB, tlsClient *tlsclient.Client, hub *websocket.Hub) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("bot: discordgo session: %w", err)
	}

	session.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers | discordgo.IntentsDirectMessages | discordgo.IntentsGuildMessages

	var cs *captcha.CapSolver
	if cfg.CapSolverAPIKey != "" {
		cs = captcha.NewCapSolver(cfg.CapSolverAPIKey)
	}

	registry := scrapers.NewRegistry(tlsClient, cs)

	mailer := email.NewClient(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
	if mailer != nil {
		log.Println("Resend email client configured")
	}

	ghlClient := ghl.NewClient(ghl.Config{
		APIKey:      cfg.GHLAPIKey,
		LocationID:  cfg.GHLLocationID,
		PipelineID:  cfg.GHLPipelineID,
		StageMap:    cfg.GHLStageMap(),
		CFDiscordID: cfg.GHLCFDiscordID,
		CFAgency:    cfg.GHLCFAgency,
		CFState:     cfg.GHLCFState,
	})
	if ghlClient != nil {
		log.Println("GoHighLevel CRM client configured")
	}

	return &Bot{
		cfg:       cfg,
		db:        database,
		session:   session,
		registry:  registry,
		mailer:    mailer,
		ghlClient: ghlClient,
		hub:       hub,
	}, nil
}

// Session returns the underlying Discord session for use by other components.
func (b *Bot) Session() *discordgo.Session {
	return b.session
}

// Registry returns the scraper registry for use by other components (e.g. API server).
func (b *Bot) Registry() *scrapers.Registry {
	return b.registry
}

func (b *Bot) Run(ctx context.Context) error {
	b.session.AddHandler(b.handleInteraction)
	b.session.AddHandler(b.handleMemberUpdate)
	b.session.AddHandler(b.handleMemberJoin)
	b.session.AddHandler(b.handleMessageCreate)

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
	case "tracker":
		b.handleTrackerCommand(s, i)
	case "log":
		b.handleLogCommand(s, i)
	case "leaderboard":
		b.handleLeaderboardCommand(s, i)
	case "start":
		b.handleStart(s, i)
	case "zoom":
		b.handleZoomCommand(s, i)
	case "role-audit":
		b.handleRoleAudit(s, i)
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
	case customID == "vipa:course_enrolled_yes":
		b.handleCourseEnrolledYes(s, i)
	case customID == "vipa:course_enrolled_no":
		b.handleCourseEnrolledNo(s, i)
	case customID == "vipa:step2_continue":
		b.handleStep2Continue(s, i)
	case customID == "vipa:step2b_continue":
		b.handleStep2bContinue(s, i)

	// Check-in buttons (vipa:checkin:{action}:{week_start})
	case strings.HasPrefix(customID, "vipa:checkin:"):
		b.handleCheckinResponse(s, i)

	// Approval buttons (vipa:approve:{id} or vipa:deny:{id})
	case strings.HasPrefix(customID, "vipa:approve:"), strings.HasPrefix(customID, "vipa:deny:"):
		b.handleApprovalButton(s, i)

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

	switch {
	case customID == "vipa:modal_step1":
		b.handleStep1Submit(s, i)
	case customID == "vipa:modal_step2":
		b.handleStep2Submit(s, i)
	case customID == "vipa:modal_step2b":
		b.handleStep2bSubmit(s, i)
	case strings.HasPrefix(customID, "vipa:deny_reason:"):
		b.handleDenyReasonModal(s, i)
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

// handleMessageCreate auto-deletes user messages in #start-here to keep it clean.
func (b *Bot) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Only act on #start-here channel
	if b.cfg.StartHereChannelID == "" || m.ChannelID != b.cfg.StartHereChannelID {
		return
	}
	// Don't delete bot's own messages
	if m.Author.ID == s.State.User.ID {
		return
	}
	// Don't delete other bot messages
	if m.Author.Bot {
		return
	}
	// Delete the user's message
	if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
		log.Printf("start-here: failed to delete message from %s: %v", m.Author.ID, err)
	}
}

// publishEvent sends an event to the WebSocket hub if available.
func (b *Bot) publishEvent(eventType string, data interface{}) {
	if b.hub != nil {
		b.hub.Publish(websocket.NewEvent(eventType, data))
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
