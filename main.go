package main

import (
	"fmt"
	"log"
	"media-server/config" // Import config package
	"media-server/handlers"
	"media-server/storage" // Import storage (where InitDB is)
)

func main() {
	// Initialize configuration (e.g., MEDIA_ROOT from .env)
	config.Init() 

	// Initialize Database
	db, err := storage.InitDB("./MediaServer.db")
	if err != nil {
		log.Fatalf("Error Initializing database: %v", err)
	}
	
	defer db.Close()

	// Sync files from filesystem to database
	err = storage.SyncFiles(db)
	if err != nil {
		log.Fatalf("Error Syncing Files: %v", err)
	}

	// Log the resolved MediaRoot for debugging
	log.Printf("Media Root Directory: %s", config.MediaRoot)

	// Pass database to handlers
	handlers.SetDB(db)

	r := setupRouter()
	r.Run(fmt.Sprintf("localhost:%v", config.AppPort))
}