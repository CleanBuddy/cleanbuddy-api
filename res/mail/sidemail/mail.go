package sidemail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	netmail "net/mail"
	"net/url"
	"strings"
	"time"

	"saas-starter-api/res/mail"
)

// SidemailService implements the MailService interface using Sidemail API
type SidemailService struct {
	apiKey         string
	apiBaseURL     string
	signUpsGroupId string
	logger         *log.Logger
	httpClient     *http.Client
}

// SidemailResponse represents a generic response from Sidemail API
type SidemailResponse struct {
	ID        string `json:"id,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	Status    string `json:"status,omitempty"`
	Error     string `json:"error,omitempty"`
	Message   string `json:"message,omitempty"`
	Success   bool   `json:"success,omitempty"`
}

// New creates a new Sidemail service instance
func New(apiKey, apiURL, signUpsGroupId string, timeout time.Duration, logger *log.Logger) mail.MailService {
	return &SidemailService{
		apiKey:         apiKey,
		apiBaseURL:     apiURL,
		signUpsGroupId: signUpsGroupId,
		logger:         logger,
		httpClient:     &http.Client{Timeout: timeout},
	}
}

// SidemailContactPayload represents the payload for creating/updating contacts via Sidemail API
type SidemailContactPayload struct {
	EmailAddress string                 `json:"emailAddress"`
	Identifier   string                 `json:"identifier"`
	IsSubscribed bool                   `json:"isSubscribed"`
	Groups       []string               `json:"groups,omitempty"`
	CustomProps  map[string]interface{} `json:"customProps,omitempty"`
}

// SidemailContactResponse represents the response from Sidemail contacts API
type SidemailContactResponse struct {
	Status string `json:"status"`
}

// validateEmail validates an email address format using Go's built-in mail parser.
// Returns an error if the email address is malformed or empty.
func (s *SidemailService) validateEmail(email string) error {
	_, err := netmail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email address: %w", err)
	}
	return nil
}

// sanitizeInput sanitizes user input to prevent injection attacks by removing
// control characters, null bytes, and trimming whitespace.
func (s *SidemailService) sanitizeInput(input string) string {
	// Remove null bytes and control characters
	cleaned := strings.ReplaceAll(input, "\x00", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	return strings.TrimSpace(cleaned)
}

// sanitizeResponseBody sanitizes response body for safe inclusion in error messages
func (s *SidemailService) sanitizeResponseBody(body string) string {
	// Limit length to prevent log injection and excessive logging
	const maxLength = 200
	sanitized := s.sanitizeInput(body)

	if len(sanitized) > maxLength {
		return sanitized[:maxLength] + "..."
	}
	return sanitized
}

// RegisterUser registers a user with Sidemail using the contacts API.
// It validates the email address and sanitizes inputs before making the API call.
// If no API key is configured, this method returns nil (graceful degradation).
func (s *SidemailService) RegisterUser(ctx context.Context, userID, email, displayName string) error {
	if s.apiKey == "" {
		s.logger.Printf("Sidemail API key not configured, skipping user registration")
		return nil
	}

	// Validate email address
	if err := s.validateEmail(email); err != nil {
		return fmt.Errorf("user registration failed: %w", err)
	}

	// Sanitize inputs
	userID = s.sanitizeInput(userID)
	email = s.sanitizeInput(email)
	displayName = s.sanitizeInput(displayName)

	// Create contact using Sidemail contacts API
	payload := SidemailContactPayload{
		EmailAddress: email,
		Identifier:   userID,
		IsSubscribed: true,
		Groups:       []string{s.signUpsGroupId},
		CustomProps: map[string]interface{}{
			"name":   displayName,
			"userID": userID,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling contact data: %w", err)
	}

	url := fmt.Sprintf("%s/contacts", s.apiBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	return s.handleSidemailContactResponse(resp, fmt.Sprintf("user registration for %s", userID))
}

// RemoveUserByEmail removes a user from Sidemail using the contacts API.
// It validates the email address before making the deletion request.
// If no API key is configured, this method returns nil (graceful degradation).
func (s *SidemailService) RemoveUserByEmail(ctx context.Context, email string) error {
	if s.apiKey == "" {
		s.logger.Printf("Sidemail API key not configured, skipping user removal")
		return nil
	}

	// Validate email address
	if err := s.validateEmail(email); err != nil {
		return fmt.Errorf("user removal failed: %w", err)
	}

	// Sanitize email
	email = s.sanitizeInput(email)

	// URL encode the email address to handle special characters safely
	encodedEmail := url.QueryEscape(email)
	requestURL := fmt.Sprintf("%s/contacts/%s", s.apiBaseURL, encodedEmail)
	req, err := http.NewRequestWithContext(ctx, "DELETE", requestURL, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	return s.handleSidemailContactResponse(resp, fmt.Sprintf("user removal for %s", email))
}

// UpdateContactProperty updates a specific custom property for a contact.
// It validates the email address and sanitizes inputs before making the API call.
// If no API key is configured, this method returns nil (graceful degradation).
func (s *SidemailService) UpdateContactProperty(ctx context.Context, email, propertyName, propertyValue string) error {
	if s.apiKey == "" {
		s.logger.Printf("Sidemail API key not configured, skipping contact property update")
		return nil
	}

	// Validate email address
	if err := s.validateEmail(email); err != nil {
		return fmt.Errorf("contact property update failed: %w", err)
	}

	// Sanitize inputs
	email = s.sanitizeInput(email)
	propertyName = s.sanitizeInput(propertyName)
	propertyValue = s.sanitizeInput(propertyValue)

	// Create/update contact with the specific property using Sidemail contacts API
	payload := SidemailContactPayload{
		EmailAddress: email,
		Identifier:   email, // Use email as identifier for property updates
		IsSubscribed: true,
		CustomProps: map[string]interface{}{
			propertyName: propertyValue,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling contact property data: %w", err)
	}

	url := fmt.Sprintf("%s/contacts", s.apiBaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	return s.handleSidemailContactResponse(resp, fmt.Sprintf("property update %s=%s for %s", propertyName, propertyValue, email))
}

// handleSidemailContactResponse handles and validates responses from the Sidemail contacts API.
// It parses the JSON response, checks for errors, and logs the outcome with structured logging.
func (s *SidemailService) handleSidemailContactResponse(resp *http.Response, operation string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	// Log the raw response for debugging
	s.logger.Printf("[SIDEMAIL_CONTACTS_RESPONSE] status=%d operation=%s body_length=%d", resp.StatusCode, operation, len(body))

	// Parse the response to check for any errors
	var response SidemailContactResponse
	if err := json.Unmarshal(body, &response); err != nil {
		s.logger.Printf("Warning: Could not parse Sidemail contacts response: %v", err)
		// If we can't parse the response, check if it contains error indicators
		bodyStr := string(body)
		if strings.Contains(strings.ToLower(bodyStr), "error") ||
			strings.Contains(strings.ToLower(bodyStr), "unauthorized") ||
			strings.Contains(strings.ToLower(bodyStr), "invalid") {
			return fmt.Errorf("sidemail contacts API error (unparseable response): %s", s.sanitizeResponseBody(bodyStr))
		}
	}

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("sidemail contacts API returned status %d: %s", resp.StatusCode, s.sanitizeResponseBody(string(body)))
	}

	s.logger.Printf("[SIDEMAIL_CONTACTS_SUCCESS] operation=%s status=%s", operation, response.Status)
	return nil
}
