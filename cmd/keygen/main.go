package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hackclub/geocoder/internal/config"
	"github.com/hackclub/geocoder/internal/database"
)

func main() {
	var (
		rateLimitPerSecond = flag.Int("rate-limit", 10, "Rate limit per second")
	)
	flag.Parse()

	// Interactive prompts for required information
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Print("Hack Club staff member username (e.g., zrl): ")
	owner, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read owner: %v", err)
	}
	owner = strings.TrimSpace(owner)
	if owner == "" {
		log.Fatal("Owner username is required")
	}

	fmt.Print("App name the API key will be used in (e.g., spotcheck): ")
	appName, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read app name: %v", err)
	}
	appName = strings.TrimSpace(appName)
	if appName == "" {
		log.Fatal("App name is required")
	}

	fmt.Print("Environment (dev for local laptop, prod for deployed app): ")
	environment, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read environment: %v", err)
	}
	environment = strings.TrimSpace(environment)
	if environment == "" {
		log.Fatal("Environment is required")
	}

	cfg := config.Load()

	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Generate prefix from user input
	prefix := fmt.Sprintf("%s_%s_%s", owner, environment, appName)
	
	// Generate a secure API key
	apiKey := generateAPIKey(prefix)
	keyHash := database.HashAPIKey(apiKey)

	// Create name in format: owner_appname_environment_randomsuffix
	randomSuffix := generateRandomString(8)
	name := fmt.Sprintf("%s_%s_%s_%s", owner, appName, environment, randomSuffix)

	// Save to database
	dbKey, err := db.CreateAPIKey(keyHash, name, owner, appName, environment, *rateLimitPerSecond)
	if err != nil {
		log.Fatalf("Failed to create API key: %v", err)
	}

	fmt.Printf("\nAPI Key created successfully!\n")
	fmt.Printf("ID: %s\n", dbKey.ID)
	fmt.Printf("Name: %s\n", dbKey.Name)
	fmt.Printf("Owner: %s\n", dbKey.Owner)
	fmt.Printf("App: %s\n", dbKey.AppName)
	fmt.Printf("Environment: %s\n", dbKey.Environment)
	fmt.Printf("Key: %s\n", apiKey)
	fmt.Printf("Rate Limit: %d requests/second\n", dbKey.RateLimitPerSecond)
	fmt.Printf("Created: %s\n", dbKey.CreatedAt.Format(time.RFC3339))
	fmt.Printf("\nPlease save this key securely as it won't be shown again.\n")
}

func generateAPIKey(prefix string) string {
	// Generate 20 random bytes
	randomBytes := make([]byte, 20)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}

	randomString := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%s_%s", prefix, randomString)
}

func generateRandomString(length int) string {
	randomBytes := make([]byte, length/2)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatalf("Failed to generate random bytes: %v", err)
	}
	return hex.EncodeToString(randomBytes)[:length]
}
