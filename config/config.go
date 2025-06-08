package config

import (
	"log"
	"os"
	"path/filepath" // Import for filepath.Clean

	"github.com/joho/godotenv" // Import godotenv
)

var MediaRoot string

func init() {
	err := godotenv.Load()
	if err != nil {
		// It's okay if .env is not found in production, but log a warning
		// if it's expected, or fatal if it's critical.
		log.Printf("Warning: Error loading .env file (might not exist in production): %v", err)
	}

	MediaRoot = os.Getenv("MEDIA_ROOT")
	log.Printf("MEDIA ROOT : %v", MediaRoot)
	if MediaRoot == "" {
		// Provide a sensible default or log a warning if MEDIA_ROOT is not set
		MediaRoot = "./media"
		// You might want to log this if it's not expected to be empty
		log.Printf("MEDIA_ROOT environment variable not set, defaulting to %s", MediaRoot)
	}
	// Clean the path to handle potential issues with trailing slashes or relative components
	// Also converts to OS-specific path separators
	MediaRoot = filepath.Clean(MediaRoot)
}