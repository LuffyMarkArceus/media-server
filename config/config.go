package config

import (
	"log"
	"os"
	"path/filepath" // Import for filepath.Clean
	"strconv"

	"github.com/joho/godotenv" // Import godotenv
)

var ( 
	MediaRoot string 
	AppPort int 
)

func Init() {
	err := godotenv.Load()
	if err != nil {
		// It's okay if .env is not found in production, but log a warning
		// if it's expected, or fatal if it's critical.
		log.Printf("No .env file found, using default values: %v", err)
	}

	// Get MEDIA_ROOT
	MediaRoot = os.Getenv("MEDIA_ROOT")
	if MediaRoot == "" {
		log.Fatal("MEDIA_ROOT environment variable is not set")
	}

	// Resolve Absolute Path
	absPath, err := filepath.Abs(MediaRoot)
	if err != nil {
		log.Fatalf("Error resolving MEDIA_ROOT path: %v", err)
	}

	// Get APP_PORT
	portStr := os.Getenv("APP_PORT")
	if portStr == "" {
		log.Printf("APP_PORT environment variable not set")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid APP_PORT value: %v", err)
	}

	// Get config ready
	MediaRoot = absPath
	AppPort = port
}