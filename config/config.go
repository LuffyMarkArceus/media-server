package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var (
	// --- App Configuration ---
	AppPort int

	// --- Database Configuration ---
	DatabaseURL string

	// --- Cloudflare R2 Configuration ---
	CloudflareR2AccountID      string
	CloudflareR2AccessKeyID    string
	CloudflareR2SecretAccessKey string
	CloudflareR2BucketName     string
	CloudflarePublicDevURL     string
)

func Init() {
	// Load values from .env file.
	// It's okay if this file doesn't exist, especially in a production/container environment.
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: No .env file found, relying on OS environment variables.")
	}

	// --- Load App Port ---
	portStr := os.Getenv("APP_PORT")
	if portStr == "" {
		AppPort = 8080 // Default port
		log.Printf("APP_PORT not set, using default value: %d", AppPort)
	} else {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("FATAL: Invalid APP_PORT value: '%s'. Must be an integer.", portStr)
		}
		AppPort = port
	}

	// --- Load Database Configuration ---
	DatabaseURL = os.Getenv("DATABASE_URL")
	if DatabaseURL == "" {
		log.Fatal("FATAL: DATABASE_URL environment variable is not set.")
	}

	// --- Load Cloudflare R2 Configuration ---
	CloudflareR2AccountID = os.Getenv("CLOUDFLARE_R2_ACCOUNT_ID")
	if CloudflareR2AccountID == "" {
		log.Fatal("FATAL: CLOUDFLARE_R2_ACCOUNT_ID environment variable is not set.")
	}

	CloudflareR2AccessKeyID = os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID")
	if CloudflareR2AccessKeyID == "" {
		log.Fatal("FATAL: CLOUDFLARE_R2_ACCESS_KEY_ID environment variable is not set.")
	}

	CloudflareR2SecretAccessKey = os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY")
	if CloudflareR2SecretAccessKey == "" {
		log.Fatal("FATAL: CLOUDFLARE_R2_SECRET_ACCESS_KEY environment variable is not set.")
	}

	CloudflareR2BucketName = os.Getenv("CLOUDFLARE_R2_BUCKET_NAME")
	if CloudflareR2BucketName == "" {
		log.Fatal("FATAL: CLOUDFLARE_R2_BUCKET_NAME environment variable is not set.")
	}
	
	CloudflarePublicDevURL = os.Getenv("CF_PUBLIC_DEV_URL")
	if CloudflarePublicDevURL == "" {
		log.Fatal("FATAL: CF_PUBLIC_DEV_URL environment variable is not set.")
	}

	// The MediaRoot variable has been removed as it's no longer needed.
	log.Println("Configuration loaded successfully.")
}