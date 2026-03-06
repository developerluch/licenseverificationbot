package bot

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"license-bot-go/api/websocket"
)

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

// publishEvent sends an event to the WebSocket hub if available.
func (b *Bot) publishEvent(eventType string, data interface{}) {
	if h, ok := b.hub.(*websocket.Hub); ok && h != nil {
		h.Publish(websocket.NewEvent(eventType, data))
	}
}
