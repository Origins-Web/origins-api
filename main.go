package main

import (
	"log"
	"os"

	"origins-api/api"
	"origins-api/config"
	"origins-api/store"
)

func main() {
	cfg := config.Load()

	// Ensure Redis URL is set
	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL environment variable is required")
	}

	// Initialize Redis Store
	st, err := store.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	server := api.NewServer(cfg, st)

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}