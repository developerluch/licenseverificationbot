package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken          string
	GuildID               string
	DatabaseURL           string
	CapSolverAPIKey       string
	LicenseCheckChannelID      string
	LicenseVerifyLogChannelID string
	HiringLogChannelID        string
	StudentRoleID         string
	LicensedAgentRoleID   string
	LogLevel              string

	// Resend Email
	ResendAPIKey  string
	EmailFrom     string
	EmailFromName string

	// Admin notifications
	AdminNotificationChannelID string

	// Onboarding channels
	StartHereChannelID string
	GreetingsChannelID string
	RulesChannelID     string
	NewAgentChannelID  string

	// Onboarding roles
	ActiveAgentRoleID string

	// Agency roles
	TFCRoleID          string
	RadiantRoleID      string
	GBURoleID          string
	TruLightRoleID     string
	ThriveRoleID       string
	ThePointRoleID     string
	SynergyRoleID      string
	IlluminateRoleID   string
	EliteOneRoleID     string
	UnassignedRoleID   string

	// Staff roles (comma-separated)
	StaffRoleIDs string

	// Scheduler config
	InactivityKickWeeks int
	CheckinDay          int // 0=Sunday, 1=Monday, ..., 6=Saturday
	CheckinHour         int // Hour in ET (0-23)

	// Tracker
	TrackerChannelID string
	NudgeAfterDays   int

	// Agency owner Discord IDs (for approval flow)
	AgencyOwnerTFC       string
	AgencyOwnerRadiant   string
	AgencyOwnerGBU       string
	AgencyOwnerTruLight  string
	AgencyOwnerThrive    string
	AgencyOwnerThePoint  string
	AgencyOwnerSynergy   string
	AgencyOwnerIlluminate string
	AgencyOwnerEliteOne  string
	PendingRoleID        string

	// GoHighLevel CRM
	GHLAPIKey       string
	GHLLocationID   string
	GHLPipelineID   string
	GHLStageWelcome string
	GHLStageForm    string
	GHLStageSorted  string
	GHLStageStudent string
	GHLStageVerified    string
	GHLStageContracting string
	GHLStageSetup       string
	GHLStageActive      string
	GHLCFDiscordID  string
	GHLCFAgency     string
	GHLCFState      string

	// API Server
	APIToken      string
	APIPort       string
	AllowedOrigin string
}

func MustLoad() *Config {
	_ = godotenv.Load() // .env is optional (Railway injects env directly)

	cfg := &Config{
		DiscordToken:          os.Getenv("DISCORD_TOKEN"),
		GuildID:               os.Getenv("GUILD_ID"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		CapSolverAPIKey:       os.Getenv("CAPSOLVER_API_KEY"),
		LicenseCheckChannelID:      os.Getenv("LICENSE_CHECK_CHANNEL_ID"),
		LicenseVerifyLogChannelID: os.Getenv("LICENSE_VERIFY_LOG_CHANNEL_ID"),
		HiringLogChannelID:        os.Getenv("HIRING_LOG_CHANNEL_ID"),
		StudentRoleID:         os.Getenv("STUDENT_ROLE_ID"),
		LicensedAgentRoleID:   os.Getenv("LICENSED_AGENT_ROLE_ID"),
		LogLevel:              os.Getenv("LOG_LEVEL"),

		ResendAPIKey:  os.Getenv("RESEND_API_KEY"),
		EmailFrom:     os.Getenv("EMAIL_FROM"),
		EmailFromName: os.Getenv("EMAIL_FROM_NAME"),

		AdminNotificationChannelID: os.Getenv("ADMIN_NOTIFICATION_CHANNEL_ID"),

		StartHereChannelID: os.Getenv("START_HERE_CHANNEL_ID"),
		GreetingsChannelID: os.Getenv("GREETINGS_CHANNEL_ID"),
		RulesChannelID:     os.Getenv("RULES_CHANNEL_ID"),
		NewAgentChannelID:  os.Getenv("NEW_AGENT_CHANNEL_ID"),

		ActiveAgentRoleID: os.Getenv("ACTIVE_AGENT_ROLE_ID"),
		TFCRoleID:         os.Getenv("TFC_ROLE_ID"),
		RadiantRoleID:     os.Getenv("RADIANT_ROLE_ID"),
		GBURoleID:         os.Getenv("GBU_ROLE_ID"),
		TruLightRoleID:    os.Getenv("TRULIGHT_ROLE_ID"),
		ThriveRoleID:      os.Getenv("THRIVE_ROLE_ID"),
		ThePointRoleID:    os.Getenv("THE_POINT_ROLE_ID"),
		SynergyRoleID:     os.Getenv("SYNERGY_ROLE_ID"),
		IlluminateRoleID:  os.Getenv("ILLUMINATE_ROLE_ID"),
		EliteOneRoleID:    os.Getenv("ELITE_ONE_ROLE_ID"),
		UnassignedRoleID:  os.Getenv("UNASSIGNED_ROLE_ID"),

		StaffRoleIDs: os.Getenv("STAFF_ROLE_IDS"),

		TrackerChannelID: os.Getenv("TRACKER_CHANNEL_ID"),

		AgencyOwnerTFC:       os.Getenv("AGENCY_OWNER_TFC"),
		AgencyOwnerRadiant:   os.Getenv("AGENCY_OWNER_RADIANT"),
		AgencyOwnerGBU:       os.Getenv("AGENCY_OWNER_GBU"),
		AgencyOwnerTruLight:  os.Getenv("AGENCY_OWNER_TRULIGHT"),
		AgencyOwnerThrive:    os.Getenv("AGENCY_OWNER_THRIVE"),
		AgencyOwnerThePoint:  os.Getenv("AGENCY_OWNER_THE_POINT"),
		AgencyOwnerSynergy:   os.Getenv("AGENCY_OWNER_SYNERGY"),
		AgencyOwnerIlluminate: os.Getenv("AGENCY_OWNER_ILLUMINATE"),
		AgencyOwnerEliteOne:  os.Getenv("AGENCY_OWNER_ELITE_ONE"),
		PendingRoleID:        os.Getenv("PENDING_ROLE_ID"),

		GHLAPIKey:           os.Getenv("GHL_API_KEY"),
		GHLLocationID:       os.Getenv("GHL_LOCATION_ID"),
		GHLPipelineID:       os.Getenv("GHL_PIPELINE_ID"),
		GHLStageWelcome:     os.Getenv("GHL_STAGE_WELCOME"),
		GHLStageForm:        os.Getenv("GHL_STAGE_FORM"),
		GHLStageSorted:      os.Getenv("GHL_STAGE_SORTED"),
		GHLStageStudent:     os.Getenv("GHL_STAGE_STUDENT"),
		GHLStageVerified:    os.Getenv("GHL_STAGE_VERIFIED"),
		GHLStageContracting: os.Getenv("GHL_STAGE_CONTRACTING"),
		GHLStageSetup:       os.Getenv("GHL_STAGE_SETUP"),
		GHLStageActive:      os.Getenv("GHL_STAGE_ACTIVE"),
		GHLCFDiscordID:      os.Getenv("GHL_CF_DISCORD_ID"),
		GHLCFAgency:         os.Getenv("GHL_CF_AGENCY"),
		GHLCFState:          os.Getenv("GHL_CF_STATE"),

		APIToken:      os.Getenv("API_TOKEN"),
		APIPort:       os.Getenv("API_PORT"),
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "INFO"
	}
	if cfg.APIPort == "" {
		cfg.APIPort = "8080"
	}

	cfg.InactivityKickWeeks = getEnvInt("INACTIVITY_KICK_WEEKS", 4)
	cfg.CheckinDay = getEnvInt("CHECKIN_DAY", 1) // 1 = Monday
	cfg.CheckinHour = getEnvInt("CHECKIN_HOUR", 9)
	cfg.NudgeAfterDays = getEnvInt("NUDGE_AFTER_DAYS", 30)

	// Validate required
	if cfg.DiscordToken == "" {
		log.Fatal("DISCORD_TOKEN is required")
	}
	if cfg.GuildID == "" {
		log.Fatal("GUILD_ID is required")
	}
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	return cfg
}

// GuildIDInt returns the guild ID as int64 for discordgo (which uses string, but we might need int for DB).
func (c *Config) GuildIDInt() int64 {
	v, _ := strconv.ParseInt(c.GuildID, 10, 64)
	return v
}

// StaffRoleIDList returns the staff role IDs as a slice.
func (c *Config) StaffRoleIDList() []string {
	if c.StaffRoleIDs == "" {
		return nil
	}
	parts := strings.Split(c.StaffRoleIDs, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// IsStaff returns true if any of the given role IDs match a staff role.
func (c *Config) IsStaff(memberRoles []string) bool {
	staffIDs := c.StaffRoleIDList()
	for _, roleID := range memberRoles {
		for _, staffID := range staffIDs {
			if roleID == staffID {
				return true
			}
		}
	}
	return false
}

// GetAgencyRoleID returns the Discord role ID for the given agency name.
func (c *Config) GetAgencyRoleID(agency string) string {
	switch strings.ToLower(strings.TrimSpace(agency)) {
	case "tfc", "topfloorclosers", "top floor closers":
		return c.TFCRoleID
	case "radiant", "radiant financial":
		return c.RadiantRoleID
	case "gbu":
		return c.GBURoleID
	case "trulight", "tru light", "ffl trulight":
		return c.TruLightRoleID
	case "thrive", "ffl thrive":
		return c.ThriveRoleID
	case "the point", "thepoint", "ffl the point":
		return c.ThePointRoleID
	case "synergy", "ffl synergy":
		return c.SynergyRoleID
	case "illuminate", "ffl illuminate":
		return c.IlluminateRoleID
	case "elite one", "eliteone", "elite 1", "ffl elite one":
		return c.EliteOneRoleID
	default:
		return c.UnassignedRoleID
	}
}

// GHLStageMap returns the mapping of bot stages (1-8) to GHL stage IDs.
func (c *Config) GHLStageMap() map[int]string {
	return map[int]string{
		1: c.GHLStageWelcome,
		2: c.GHLStageForm,
		3: c.GHLStageSorted,
		4: c.GHLStageStudent,
		5: c.GHLStageVerified,
		6: c.GHLStageContracting,
		7: c.GHLStageSetup,
		8: c.GHLStageActive,
	}
}

// GetAgencyOwnerID returns the Discord user ID of the agency owner for approval flow.
func (c *Config) GetAgencyOwnerID(agency string) string {
	switch strings.ToLower(strings.TrimSpace(agency)) {
	case "tfc", "topfloorclosers", "top floor closers":
		return c.AgencyOwnerTFC
	case "radiant", "radiant financial":
		return c.AgencyOwnerRadiant
	case "gbu":
		return c.AgencyOwnerGBU
	case "trulight", "tru light":
		return c.AgencyOwnerTruLight
	case "thrive":
		return c.AgencyOwnerThrive
	case "the point", "thepoint":
		return c.AgencyOwnerThePoint
	case "synergy":
		return c.AgencyOwnerSynergy
	case "illuminate":
		return c.AgencyOwnerIlluminate
	case "elite one", "eliteone", "elite 1":
		return c.AgencyOwnerEliteOne
	default:
		return ""
	}
}

func getEnvInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}
