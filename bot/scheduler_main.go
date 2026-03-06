package bot

import (
	"context"
	"log"
	"time"

	"license-bot-go/email"
)

// StartScheduler runs a background loop that checks deadlines every 24 hours.
// It sends reminders at day 7, 14, and 21, and notifies admin when deadlines expire.
func (b *Bot) StartScheduler(ctx context.Context, mailer *email.Client) {
	log.Println("Scheduler started: checking deadlines every 24 hours")

	// Run immediately on startup, then every 24h
	b.runSchedulerCycle(mailer)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Scheduler shutting down")
			return
		case <-ticker.C:
			b.runSchedulerCycle(mailer)
		}
	}
}

func (b *Bot) runSchedulerCycle(mailer *email.Client) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Scheduler panic: %v", r)
		}
	}()

	log.Println("Scheduler: running deadline check cycle")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. Re-attempt verification for pending deadlines
	b.retryVerifications(ctx, mailer)

	// 2. Send reminders for approaching deadlines (every 7 days)
	b.sendReminders(ctx, mailer)

	// 3. Handle expired deadlines — notify admin
	b.handleExpiredDeadlines(ctx, mailer)

	// 4. Weekly check-ins (only on configured day)
	loc, _ := time.LoadLocation("America/New_York")
	nowET := time.Now().In(loc)
	if int(nowET.Weekday()) == b.cfg.CheckinDay {
		b.sendWeeklyCheckins(ctx)
	}

	// 5. Inactivity check (daily)
	b.checkInactivity(ctx)

	// 6. Recruiter nudges for unlicensed agents past threshold
	b.sendRecruiterNudges(ctx)

	// 7. Unlicensed agent 60-day kick check (daily)
	b.checkUnlicensedKicks(ctx)

	// 8. Daily tracker auto-post (only once per day, early afternoon ET)
	if nowET.Hour() >= 13 && nowET.Hour() < 14 {
		b.postDailyTracker(ctx)
	}
}
