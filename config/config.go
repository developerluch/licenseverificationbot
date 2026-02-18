package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken          string
	GuildID               string
	DatabaseURL           string
	CapSolverAPIKey       string
	LicenseCheckChannelID string
	HiringLogChannelID    string
	StudentRoleID         string
	LicensedAgentRoleID   string
	LogLevel              string

	// Twilio SMS
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string

	// Admin notifications
	AdminNotificationChannelID string
}

func MustLoad() *Config {
	_ = godotenv.Load() // .env is optional (Railway injects env directly)

	cfg := &Config{
		DiscordToken:          os.Getenv("DISCORD_TOKEN"),
		GuildID:               os.Getenv("GUILD_ID"),
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		CapSolverAPIKey:       os.Getenv("CAPSOLVER_API_KEY"),
		LicenseCheckChannelID: os.Getenv("LICENSE_CHECK_CHANNEL_ID"),
		HiringLogChannelID:    os.Getenv("HIRING_LOG_CHANNEL_ID"),
		StudentRoleID:         os.Getenv("STUDENT_ROLE_ID"),
		LicensedAgentRoleID:   os.Getenv("LICENSED_AGENT_ROLE_ID"),
		LogLevel:              os.Getenv("LOG_LEVEL"),

		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFromNumber: os.Getenv("TWILIO_FROM_NUMBER"),

		AdminNotificationChannelID: os.Getenv("ADMIN_NOTIFICATION_CHANNEL_ID"),
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = "INFO"
	}

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
