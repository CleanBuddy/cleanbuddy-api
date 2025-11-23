package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"saas-starter-api/api"
	"saas-starter-api/api/playground"
	"saas-starter-api/res/store"
	"saas-starter-api/res/store/postgresql"

	"github.com/joho/godotenv"
)

var logger = log.New(os.Stdout, "(cmd/main.go)", log.LstdFlags|log.LUTC|log.Llongfile)

func main() {
	// Load .env file in development
	// Try multiple locations: current dir, saas-starter-api/, parent dir
	err := godotenv.Load()
	if err != nil {
		err = godotenv.Load("saas-starter-api/.env")
	}
	if err != nil {
		err = godotenv.Load(".env")
	}
	if err != nil {
		logger.Printf("Note: .env file not found, using system environment variables")
	}

	port := readRequiredEnvVar("PORT")
	environment := readRequiredEnvVar("ENVIRONMENT")

	// Bootstrap global admin if GLOBAL_ADMIN_EMAIL is set
	if globalAdminEmail := os.Getenv("GLOBAL_ADMIN_EMAIL"); globalAdminEmail != "" {
		if err := bootstrapGlobalAdmin(globalAdminEmail); err != nil {
			logger.Printf("Warning: Failed to bootstrap global admin: %v", err)
		} else {
			logger.Printf("Successfully checked/updated global admin: %s", globalAdminEmail)
		}
	}

	// Main GraphQL API endpoint
	http.HandleFunc("/api", api.Handler)

	// GraphQL playground (disabled in production)
	if environment != "production" {
		http.HandleFunc("/api/playground", playground.Handler)
		logger.Printf("GraphQL playground enabled at /api/playground")
	} else {
		logger.Printf("GraphQL playground disabled (production mode)")
	}

	logger.Printf("Starting server on :%s (environment: %s)\n", port, environment)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		logger.Fatalf("Server failed to start: %v", err)
	}
}

func readRequiredEnvVar(name string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		logger.Fatalf("Env variable not set: %s", name)
	}
	return val
}

func bootstrapGlobalAdmin(email string) error {
	// Connect to database
	dbURL := readRequiredEnvVar("DATABASE_POSTGRES_URL")
	storeInstance, err := postgresql.Connect(dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	ctx := context.Background()

	// Find user by email
	user, err := storeInstance.Users().GetByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("failed to find user with email %s: %w", email, err)
	}

	// If user already has global admin role, nothing to do
	if user.Role == store.UserRoleGlobalAdmin {
		logger.Printf("User %s already has global admin role", email)
		return nil
	}

	// Update user role to global admin
	globalAdminRole := store.UserRoleGlobalAdmin
	_, err = storeInstance.Users().Update(ctx, user.ID, nil, &globalAdminRole)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	logger.Printf("Successfully promoted user %s to global admin", email)
	return nil
}
