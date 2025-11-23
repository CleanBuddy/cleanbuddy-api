package mail

import (
	"context"
)

// MailService defines the interface for email operations
type MailService interface {
	// RegisterUser registers a user with the email service
	RegisterUser(ctx context.Context, userID, email, displayName string) error

	// RemoveUserByEmail removes a user from the email service by email address
	RemoveUserByEmail(ctx context.Context, email string) error

	// UpdateContactProperty updates a specific custom property for a contact
	UpdateContactProperty(ctx context.Context, email, propertyName, propertyValue string) error
}
