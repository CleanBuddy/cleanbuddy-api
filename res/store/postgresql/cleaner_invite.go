package postgresql

import (
	"context"
	"fmt"
	"time"

	"cleanbuddy-api/res/store"
)

type cleanerInviteStore struct {
	*storeImpl
}

func NewCleanerInviteStore(rootStore *storeImpl) *cleanerInviteStore {
	return &cleanerInviteStore{storeImpl: rootStore}
}

func (cis *cleanerInviteStore) Create(ctx context.Context, invite *store.CleanerInvite) error {
	result := cis.db.WithContext(ctx).Create(invite)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create cleaner invite")
	}
	return nil
}

func (cis *cleanerInviteStore) Get(ctx context.Context, id string) (*store.CleanerInvite, error) {
	var invite store.CleanerInvite
	result := cis.db.WithContext(ctx).Where("id = ?", id).First(&invite)
	if result.Error != nil {
		return nil, result.Error
	}
	return &invite, nil
}

func (cis *cleanerInviteStore) GetByToken(ctx context.Context, token string) (*store.CleanerInvite, error) {
	var invite store.CleanerInvite
	result := cis.db.WithContext(ctx).Where("token = ?", token).First(&invite)
	if result.Error != nil {
		return nil, result.Error
	}
	return &invite, nil
}

func (cis *cleanerInviteStore) GetByCompany(ctx context.Context, companyID string) ([]*store.CleanerInvite, error) {
	var invites []*store.CleanerInvite
	result := cis.db.WithContext(ctx).
		Where("company_id = ?", companyID).
		Order("created_at DESC").
		Find(&invites)
	if result.Error != nil {
		return nil, result.Error
	}
	return invites, nil
}

func (cis *cleanerInviteStore) GetPendingByCompany(ctx context.Context, companyID string) ([]*store.CleanerInvite, error) {
	var invites []*store.CleanerInvite
	result := cis.db.WithContext(ctx).
		Where("company_id = ? AND status = ? AND expires_at > ?",
			companyID,
			store.CleanerInviteStatusPending,
			time.Now()).
		Order("created_at DESC").
		Find(&invites)
	if result.Error != nil {
		return nil, result.Error
	}
	return invites, nil
}

func (cis *cleanerInviteStore) GetByAcceptedUserID(ctx context.Context, userID string) (*store.CleanerInvite, error) {
	var invite store.CleanerInvite
	result := cis.db.WithContext(ctx).
		Where("accepted_by_id = ? AND status = ?", userID, store.CleanerInviteStatusAccepted).
		First(&invite)
	if result.Error != nil {
		return nil, result.Error
	}
	return &invite, nil
}

func (cis *cleanerInviteStore) Update(ctx context.Context, invite *store.CleanerInvite) error {
	result := cis.db.WithContext(ctx).Save(invite)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("cleaner invite not found (id: %s)", invite.ID)
	}
	return nil
}

func (cis *cleanerInviteStore) MarkAsAccepted(ctx context.Context, id string, acceptedByID string) error {
	now := time.Now()
	result := cis.db.WithContext(ctx).
		Model(&store.CleanerInvite{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         store.CleanerInviteStatusAccepted,
			"accepted_by_id": acceptedByID,
			"accepted_at":    now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("cleaner invite not found (id: %s)", id)
	}
	return nil
}

func (cis *cleanerInviteStore) MarkAsRevoked(ctx context.Context, id string) error {
	result := cis.db.WithContext(ctx).
		Model(&store.CleanerInvite{}).
		Where("id = ?", id).
		Update("status", store.CleanerInviteStatusRevoked)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("cleaner invite not found (id: %s)", id)
	}
	return nil
}

func (cis *cleanerInviteStore) Delete(ctx context.Context, id string) error {
	result := cis.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&store.CleanerInvite{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (cis *cleanerInviteStore) DeleteExpired(ctx context.Context, olderThan time.Time) error {
	result := cis.db.WithContext(ctx).
		Where("expires_at < ? AND status = ?", olderThan, store.CleanerInviteStatusPending).
		Delete(&store.CleanerInvite{})
	return result.Error
}
