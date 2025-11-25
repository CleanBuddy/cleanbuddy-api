package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"cleanbuddy-api/res/auth"
	"cleanbuddy-api/res/mail"
	"cleanbuddy-api/res/mail/sidemail"
	"cleanbuddy-api/res/notification"
	"cleanbuddy-api/res/notification/slack"
	"cleanbuddy-api/res/storage"
	"cleanbuddy-api/res/store"
	"cleanbuddy-api/res/store/postgresql"
	"cleanbuddy-api/sys/graphql"
	"cleanbuddy-api/sys/http/middleware"
)

var logger = log.New(os.Stdout, "", log.LstdFlags|log.LUTC|log.Llongfile)

// CONFIGURATION CONVENTION:
// All environment variable configuration is centralized in this file (api/index.go).
// This provides a single location to view all configuration requirements and ensures
// consistent handling of environment variables across the application.
//
// REQUIRED Environment Variables (minimum to run):
// - DATABASE_POSTGRES_URL: PostgreSQL connection string
// - AUTH_JWT_SECRET: JWT signing secret
// - AUTH_GOOGLE_CLIENT_ID: Google OAuth client ID
// - AUTH_GOOGLE_SECRET: Google OAuth client secret
// - AUTH_GOOGLE_REDIRECT_URL: Google OAuth redirect URL
//
// OPTIONAL Environment Variables (with graceful degradation):
// - SIDEMAIL_API_KEY: Sidemail API key for email operations (optional)
// - SIDEMAIL_API_URL: Sidemail API base URL (default: https://api.sidemail.io/v1)
// - SIDEMAIL_SIGNUPS_GROUP_ID: Sidemail group ID for user signups (optional)
// - SLACK_WEBHOOK_URL: Slack webhook URL for notifications (optional)
// - SLACK_TIMEOUT_SECONDS: Timeout for notification API requests in seconds (default: 5)
// - GCS_BUCKET_NAME: Google Cloud Storage bucket name for document uploads (optional)
// - GCS_PROJECT_ID: Google Cloud project ID (optional)
// - GOOGLE_APPLICATION_CREDENTIALS: Path to GCS service account credentials file (optional)

// Global service instances initialized once
var (
	storeInstance               store.Store
	authInstance                auth.Auth
	mailServiceInstance         mail.MailService
	notificationServiceInstance notification.NotificationService
	storageServiceInstance      *storage.GCSService
	initOnce                    sync.Once
	initError                   error
)

func Handler(w http.ResponseWriter, r *http.Request) {
	// Initialize services only once using sync.Once
	initOnce.Do(func() {
		storeInstance, initError = configStore()
		if initError != nil {
			return
		}

		authInstance = configAuth()
		mailServiceInstance = configMail()
		notificationServiceInstance = configNotification()
		storageServiceInstance = configStorage()
	})

	if initError != nil {
		logger.Fatalf("Failed to initialize services: %v", initError)
	}

	graphqlServerHandler := graphql.New(&graphql.Config{
		Logger:              logger,
		Store:               storeInstance,
		Auth:                authInstance,
		MailService:         mailServiceInstance,
		NotificationService: notificationServiceInstance,
		StorageService:      storageServiceInstance,
	})

	// GraphQL endpoint with middleware stack
	middleware.CSPMiddleware()(
		middleware.CORSMiddleware()(
			middleware.AuthMiddleware(logger, storeInstance, authInstance)(graphqlServerHandler),
		),
	).ServeHTTP(w, r)
}

func readRequiredEnvVar(name string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		logger.Fatalf("Env variable not set: %s", name)
	}
	return val
}

func readOptionalEnvVar(name, defaultValue string) string {
	val, ok := os.LookupEnv(name)
	if !ok {
		return defaultValue
	}
	return val
}

func configStore() (store.Store, error) {
	rawStore, err := postgresql.Connect(readRequiredEnvVar("DATABASE_POSTGRES_URL"))
	if err != nil {
		return nil, err
	}
	return rawStore, nil
}

func configAuth() auth.Auth {
	return auth.New(
		readRequiredEnvVar("AUTH_JWT_SECRET"),
		readRequiredEnvVar("AUTH_GOOGLE_CLIENT_ID"),
		readRequiredEnvVar("AUTH_GOOGLE_SECRET"),
		readRequiredEnvVar("AUTH_GOOGLE_REDIRECT_URL"),
	)
}

func configMail() mail.MailService {
	apiKey := readOptionalEnvVar("SIDEMAIL_API_KEY", "")
	if apiKey == "" {
		logger.Printf("SIDEMAIL_API_KEY not set, email service disabled")
		return nil
	}

	apiURL := readOptionalEnvVar("SIDEMAIL_API_URL", "https://api.sidemail.io/v1")
	signUpsGroupId := readOptionalEnvVar("SIDEMAIL_SIGNUPS_GROUP_ID", "")
	timeout := 10 * time.Second

	return sidemail.New(apiKey, apiURL, signUpsGroupId, timeout, logger)
}

func configNotification() notification.NotificationService {
	webhookURL := readOptionalEnvVar("SLACK_WEBHOOK_URL", "")
	if webhookURL == "" {
		logger.Printf("SLACK_WEBHOOK_URL not set, notifications disabled")
		return nil
	}

	timeoutSeconds := readOptionalEnvVar("SLACK_TIMEOUT_SECONDS", "5")
	timeout, _ := time.ParseDuration(timeoutSeconds + "s")

	return slack.New(webhookURL, timeout, logger)
}

func configStorage() *storage.GCSService {
	bucketName := readOptionalEnvVar("GCS_BUCKET_NAME", "")
	if bucketName == "" {
		logger.Printf("GCS_BUCKET_NAME not set, storage service disabled")
		return nil
	}

	projectID := readOptionalEnvVar("GCS_PROJECT_ID", "")
	if projectID == "" {
		logger.Printf("GCS_PROJECT_ID not set, storage service disabled")
		return nil
	}

	credentialsPath := readOptionalEnvVar("GOOGLE_APPLICATION_CREDENTIALS", "")

	ctx := context.Background()
	gcsService, err := storage.NewGCSService(ctx, bucketName, projectID, credentialsPath)
	if err != nil {
		logger.Printf("Failed to initialize GCS storage service: %v. Storage service disabled.", err)
		return nil
	}

	logger.Printf("GCS storage service initialized successfully (bucket: %s)", bucketName)
	return gcsService
}
