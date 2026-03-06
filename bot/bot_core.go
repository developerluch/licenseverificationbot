package bot

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

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
	hub              interface{} // websocket.Hub
	modalState       sync.Map // userID (string) -> *ModalTempData
	welcomeMessages  sync.Map // userID (string) -> welcomeMsgRef{channelID, messageID}
}

// welcomeMsgRef stores the channel and message ID for a user's welcome message in #start-here.
type welcomeMsgRef struct {
	ChannelID string
	MessageID string
}

func New(cfg *config.Config, database *db.DB, tlsClient *tlsclient.Client, hub interface{}) (*Bot, error) {
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
