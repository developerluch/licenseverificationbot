package bot

import (
	"github.com/bwmarrin/discordgo"
)

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
	case "ticket-setup":
		b.handleTicketSetup(s, i)
	case "wavv-ticket-setup":
		b.handleWAVVTicketSetup(s, i)
	case "wavv-log":
		b.handleWavvLogCommand(s, i)
	case "wavv-stats":
		b.handleWavvStatsCommand(s, i)
	}
}
