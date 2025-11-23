package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"saas-starter-api/res/notification"
)

// notificationService implements the NotificationService interface
type notificationService struct {
	webhookURL string
	httpClient *http.Client
	logger     *log.Logger
}

// slackMessage represents the structure of a Slack message
type slackMessage struct {
	Text string `json:"text"`
}

// New creates a new NotificationService instance
func New(webhookURL string, timeout time.Duration, logger *log.Logger) notification.NotificationService {
	return &notificationService{
		webhookURL: webhookURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// NotifyNewUserSignup sends a notification to Slack when a new user signs up
func (s *notificationService) NotifyNewUserSignup(ctx context.Context, email, displayName, userID string) error {
	// If webhook URL is not configured, skip notification silently
	if s.webhookURL == "" {
		s.logger.Printf("Slack webhook URL not configured, skipping notification")
		return nil
	}

	message := slackMessage{
		Text: fmt.Sprintf(":tada: New user signup: %s (%s) - User ID: %s", email, displayName, userID),
	}

	return s.sendToSlack(ctx, message)
}

// SendFeedback sends a user feedback message to Slack
func (s *notificationService) SendFeedback(ctx context.Context, message, userID, userEmail string) error {
	// If webhook URL is not configured, skip notification silently
	if s.webhookURL == "" {
		s.logger.Printf("Slack webhook URL not configured, skipping feedback")
		return nil
	}

	slackMsg := slackMessage{
		Text: fmt.Sprintf(":speech_balloon: User Feedback\n*From:* %s (%s)\n*Message:* %s", userEmail, userID, message),
	}

	return s.sendToSlack(ctx, slackMsg)
}

// sendToSlack is a helper method to send messages to Slack
func (s *notificationService) sendToSlack(ctx context.Context, message slackMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal slack message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack API returned non-OK status %d: %s", resp.StatusCode, string(body))
	}

	s.logger.Printf("Successfully sent Slack message")
	return nil
}
