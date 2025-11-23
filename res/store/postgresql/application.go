package postgresql

import (
	"context"
	"fmt"
	"time"

	"saas-starter-api/res/store"
)

type applicationStore struct {
	parent *storeImpl
}

func NewApplicationStore(parent *storeImpl) *applicationStore {
	return &applicationStore{parent: parent}
}

func (s *applicationStore) Create(ctx context.Context, application *store.Application) error {
	if application.ID == "" {
		return fmt.Errorf("application ID is required")
	}
	if application.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	result := s.parent.db.WithContext(ctx).Create(application)
	if result.Error != nil {
		return fmt.Errorf("failed to create application: %w", result.Error)
	}

	return nil
}

func (s *applicationStore) Get(ctx context.Context, id string) (*store.Application, error) {
	var application store.Application
	result := s.parent.db.WithContext(ctx).
		Preload("User").
		Preload("ReviewedBy").
		First(&application, "id = ?", id)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get application: %w", result.Error)
	}

	return &application, nil
}

func (s *applicationStore) GetByUser(ctx context.Context, userID string) ([]*store.Application, error) {
	var applications []*store.Application
	result := s.parent.db.WithContext(ctx).
		Preload("User").
		Preload("ReviewedBy").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&applications)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get applications by user: %w", result.Error)
	}

	return applications, nil
}

func (s *applicationStore) GetPending(ctx context.Context) ([]*store.Application, error) {
	var applications []*store.Application
	result := s.parent.db.WithContext(ctx).
		Preload("User").
		Preload("ReviewedBy").
		Where("status = ?", store.ApplicationStatusPending).
		Order("created_at ASC").
		Find(&applications)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get pending applications: %w", result.Error)
	}

	return applications, nil
}

func (s *applicationStore) UpdateStatus(ctx context.Context, id string, status store.ApplicationStatus, reviewerID string) error {
	now := time.Now()
	result := s.parent.db.WithContext(ctx).
		Model(&store.Application{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         status,
			"reviewed_by_id": reviewerID,
			"reviewed_at":    now,
			"updated_at":     now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update application status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("application not found: %s", id)
	}

	return nil
}

func (s *applicationStore) GetByUserAndType(ctx context.Context, userID string, appType store.ApplicationType) ([]*store.Application, error) {
	var applications []*store.Application
	result := s.parent.db.WithContext(ctx).
		Preload("User").
		Preload("ReviewedBy").
		Where("user_id = ? AND application_type = ?", userID, appType).
		Order("created_at DESC").
		Find(&applications)

	if result.Error != nil {
		return nil, fmt.Errorf("failed to get applications by user and type: %w", result.Error)
	}

	return applications, nil
}
