package main

import (
	"fmt"
	"log"
	"media-server/config"
	"media-server/handlers"
	"media-server/r2" // Import the new r2 package
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

	// Initialize R2 Client
	r2Client, err := r2.NewR2Client()
	if err != nil {
		log.Fatalf("Error Initializing R2 client: %v", err)
	}
	log.Println("Cloudflare R2 Client Initialized.")

	// Pass database and R2 client to handlers
	handlers.SetDB(db)
	handlers.SetR2Client(r2Client) // New function to set the R2 client

	// Sync files from R2 to database at startup
	// This replaces the old filesystem walk.
	err = storage.SyncFilesWithR2(db, r2Client, config.CloudflareR2BucketName)
	if err != nil {
		log.Fatalf("Error Syncing Files from R2: %v", err)
	}

	// Setup and run the router
	r := setupRouter()
	r.Run(fmt.Sprintf(":%v", config.AppPort)) // Listen on all interfaces for container deployment
}
