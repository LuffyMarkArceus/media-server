package main

import (
	"fmt"
	"log"
	"media-server/config"
	"media-server/handlers"
	"media-server/r2"
	"media-server/storage"
)

func main() {
	// Initialize configuration from .env
	config.Init()

	// Initialize Database (Neon)
	db, err := storage.InitDB(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Error Initializing database: %v", err)
	}
	defer db.Close()

	// Initialize Cloudflare R2 Client
	r2Client, err := r2.NewR2Client()
	if err != nil {
		log.Fatalf("Error Initializing R2 client: %v", err)
	}
	log.Println("Cloudflare R2 Client Initialized.")

	// Pass database and R2 client to handlers
	handlers.SetDB(db)
	handlers.SetR2Client(r2Client)

	// Step 1: Sync files from R2 to DB (inserts any new files)
	err = storage.SyncFilesWithR2(db, r2Client, config.CloudflareR2BucketName)
	if err != nil {
		log.Fatalf("Error Syncing Files from R2: %v", err)
	}

	// Step 2: For already-synced files, generate missing thumbnails/subtitles
	err = storage.GenerateMissingAssetsForExistingFiles(db, r2Client, config.CloudflareR2BucketName)
	if err != nil {
		log.Fatalf("Error Generating Thumbnails/Subtitles: %v", err)
	}

	// Step 3: Start the HTTP server
	r := setupRouter()
	r.Run(fmt.Sprintf(":%v", config.AppPort))
}
