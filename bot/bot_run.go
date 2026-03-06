package bot

import (
	"context"
	"log"
)

func (b *Bot) Run(ctx context.Context) error {
	b.session.AddHandler(b.handleInteraction)
	b.session.AddHandler(b.handleMemberUpdate)
	b.session.AddHandler(b.handleMemberJoin)
	b.session.AddHandler(b.handleMessageCreate)

	if err := b.session.Open(); err != nil {
		return err
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
