package bot

import (
	"context"
	"log"
	"time"
)

// syncGHLStage moves the agent's GHL opportunity to the matching pipeline stage.
func (b *Bot) syncGHLStage(discordID int64, stage int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("syncGHLStage panic: %v", r)
		}
	}()

	if b.ghlClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	agent, err := b.db.GetAgent(ctx, discordID)
	if err != nil || agent == nil || agent.GHLContactID == "" {
		return
	}

	if err := b.ghlClient.MoveToStage(ctx, agent.GHLContactID, stage); err != nil {
		log.Printf("GHL stage sync: failed for %d (stage %d): %v", discordID, stage, err)
	}
}

// markGHLLost marks the agent's GHL opportunity as lost.
func (b *Bot) markGHLLost(discordID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("markGHLLost panic: %v", r)
		}
	}()

	if b.ghlClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	agent, err := b.db.GetAgent(ctx, discordID)
	if err != nil || agent == nil || agent.GHLContactID == "" {
		return
	}

	if err := b.ghlClient.MarkOpportunityLost(ctx, agent.GHLContactID); err != nil {
		log.Printf("GHL lost: failed for %d: %v", discordID, err)
	}
}
