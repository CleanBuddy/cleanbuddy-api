package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// GCSService handles file uploads to Google Cloud Storage
type GCSService struct {
	client     *storage.Client
	bucketName string
	projectID  string
}

// NewGCSService creates a new Google Cloud Storage service
func NewGCSService(ctx context.Context, bucketName, projectID, credentialsPath string) (*GCSService, error) {
	var client *storage.Client
	var err error

	if credentialsPath != "" {
		client, err = storage.NewClient(ctx, option.WithCredentialsFile(credentialsPath))
	} else {
		// Use default credentials (for GCE, Cloud Run, etc.)
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &GCSService{
		client:     client,
		bucketName: bucketName,
		projectID:  projectID,
	}, nil
}

// Close closes the GCS client
func (s *GCSService) Close() error {
	return s.client.Close()
}

// UploadFile uploads a file to Google Cloud Storage
func (s *GCSService) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader, objectPath string) (string, error) {
	// Validate file size (10MB max)
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if header.Size > maxFileSize {
		return "", fmt.Errorf("file size %d exceeds maximum allowed size of %d bytes", header.Size, maxFileSize)
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExtensions := map[string]bool{
		".pdf":  true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	if !allowedExtensions[ext] {
		return "", fmt.Errorf("file type %s not allowed. Allowed types: PDF, JPG, PNG", ext)
	}

	// Create object writer
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	writer := obj.NewWriter(ctx)

	// Set content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	writer.ContentType = contentType

	// Copy file to GCS
	if _, err := io.Copy(writer, file); err != nil {
		writer.Close()
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Return the public URL (will use signed URL for access)
	return fmt.Sprintf("gs://%s/%s", s.bucketName, objectPath), nil
}

// UploadFromReader uploads a file to Google Cloud Storage from io.Reader (for GraphQL Upload)
func (s *GCSService) UploadFromReader(ctx context.Context, reader io.Reader, filename string, fileSize int64, contentType string, objectPath string) (string, error) {
	// Validate file size (10MB max)
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if fileSize > maxFileSize {
		return "", fmt.Errorf("file size %d exceeds maximum allowed size of %d bytes", fileSize, maxFileSize)
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(filename))
	allowedExtensions := map[string]bool{
		".pdf":  true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	if !allowedExtensions[ext] {
		return "", fmt.Errorf("file type %s not allowed. Allowed types: PDF, JPG, PNG", ext)
	}

	// Create object writer
	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	writer := obj.NewWriter(ctx)

	// Set content type
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	writer.ContentType = contentType

	// Copy file to GCS
	if _, err := io.Copy(writer, reader); err != nil {
		writer.Close()
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Return the GCS URL
	return fmt.Sprintf("gs://%s/%s", s.bucketName, objectPath), nil
}

// GenerateSignedURL generates a signed URL for accessing a private file
func (s *GCSService) GenerateSignedURL(ctx context.Context, objectPath string, expiration time.Duration) (string, error) {
	// Remove gs:// prefix if present
	objectPath = strings.TrimPrefix(objectPath, fmt.Sprintf("gs://%s/", s.bucketName))

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	}

	url, err := s.client.Bucket(s.bucketName).SignedURL(objectPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// DeleteFile deletes a file from Google Cloud Storage
func (s *GCSService) DeleteFile(ctx context.Context, objectPath string) error {
	// Remove gs:// prefix if present
	objectPath = strings.TrimPrefix(objectPath, fmt.Sprintf("gs://%s/", s.bucketName))

	obj := s.client.Bucket(s.bucketName).Object(objectPath)
	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// BuildApplicationDocumentPath builds a path for an application document
func BuildApplicationDocumentPath(applicationID, documentType, filename string) string {
	ext := filepath.Ext(filename)
	timestamp := time.Now().Unix()

	// Sanitize document type
	docType := strings.ToLower(strings.ReplaceAll(documentType, " ", "-"))

	return fmt.Sprintf("applications/%s/%s-%d%s", applicationID, docType, timestamp, ext)
}
