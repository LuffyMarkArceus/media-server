package main

import (
	"log"
	"media-server/config"  // Import config package
	"media-server/storage" // Import storage (where InitDB is)
)

func main() {
	// Initialize configuration (e.g., MEDIA_ROOT from .env)
	//Load ENV variables:
	

	// config.init() // This is called automatically if `init` function is defined in config
	
	db, err := storage.InitDB("./MediaServer.db")
	if err != nil {
		log.Fatalf("Error Initializing database: %v", err)
	}
	defer db.Close()

	// Log the resolved MediaRoot for debugging
	log.Printf("Media Root Directory: %s", config.MediaRoot)

	r := setupRouter()
	r.Run("localhost:8080")
}