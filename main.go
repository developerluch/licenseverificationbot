package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"license-bot-go/api"
	"license-bot-go/api/websocket"
	"license-bot-go/bot"
	"license-bot-go/config"
	"license-bot-go/db"
	"license-bot-go/tlsclient"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Starting VIPA License Bot (Go)...")

	cfg := config.MustLoad()

	log.Printf("Config loaded — Guild: %s", cfg.GuildID)

	database, err := db.New(cfg)
	if err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer database.Close()
	log.Println("Database ready")

	tlsClient := tlsclient.New()
	log.Println("TLS client ready")

	// Create WebSocket hub for real-time event broadcasting
	hub := websocket.NewHub()

	b, err := bot.New(cfg, database, tlsClient, hub)
	if err != nil {
		log.Fatalf("Bot init failed: %v", err)
	}
	log.Println("Bot created, connecting to Discord...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start WebSocket hub
	go hub.Run(ctx)

	// Start API server if configured
	if cfg.APIToken != "" {
		apiServer := api.NewServer(cfg, database, b.Session(), hub, b.Registry())
		go apiServer.Start(ctx)
	}

	if err := b.Run(ctx); err != nil {
		log.Fatalf("Bot error: %v", err)
	}
}
