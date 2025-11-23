package notification

import "context"

// NotificationService defines the interface for notification operations
type NotificationService interface {
	// NotifyNewUserSignup sends a notification when a new user signs up
	NotifyNewUserSignup(ctx context.Context, email, displayName, userID string) error
	// SendFeedback sends a user feedback message to Slack
	SendFeedback(ctx context.Context, message, userID, userEmail string) error
}
