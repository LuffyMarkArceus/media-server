package main

import (
	"fmt"
	"log"
	"os"

	"media-server/config"   // Loads env vars like MEDIA_ROOT, etc.
	"media-server/handlers" // HTTP handlers
	"media-server/storage"  // DB init + sync logic
)

func main() {
	// Load config from .env
	config.Init()

	// Get NeonDB URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	// Initialize PostgreSQL database (NeonDB)
	db, err := storage.InitDB(databaseURL)
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}
	defer db.Close()

	// Sync local files to NeonDB
	if err := storage.SyncFiles(db); err != nil {
		log.Fatalf("Error syncing files: %v", err)
	}

	log.Printf("Media Root Directory: %s", config.MediaRoot)

	// Pass database connection to handlers
	handlers.SetDB(db)

	// Start HTTP server
	r := setupRouter()
	r.Run(fmt.Sprintf("localhost:%v", config.AppPort))
}
