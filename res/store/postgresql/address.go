package postgresql

import (
	"context"
	"fmt"

	"cleanbuddy-api/res/store"
)

type addressStore struct {
	*storeImpl
}

func NewAddressStore(rootStore *storeImpl) *addressStore {
	return &addressStore{storeImpl: rootStore}
}

func (as *addressStore) Create(ctx context.Context, address *store.Address) error {
	result := as.db.WithContext(ctx).Create(address)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create address")
	}
	return nil
}

func (as *addressStore) Get(ctx context.Context, id string) (*store.Address, error) {
	var address store.Address
	result := as.db.WithContext(ctx).Where("id = ?", id).First(&address)
	if result.Error != nil {
		return nil, result.Error
	}
	return &address, nil
}

func (as *addressStore) GetByUser(ctx context.Context, userID string) ([]*store.Address, error) {
	var addresses []*store.Address
	result := as.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, created_at DESC").
		Find(&addresses)
	if result.Error != nil {
		return nil, result.Error
	}
	return addresses, nil
}

func (as *addressStore) GetDefaultByUser(ctx context.Context, userID string) (*store.Address, error) {
	var address store.Address
	result := as.db.WithContext(ctx).
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&address)
	if result.Error != nil {
		return nil, result.Error
	}
	return &address, nil
}

func (as *addressStore) Update(ctx context.Context, address *store.Address) error {
	result := as.db.WithContext(ctx).Save(address)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("address not found (id: %s)", address.ID)
	}
	return nil
}

func (as *addressStore) Delete(ctx context.Context, id string) error {
	result := as.db.WithContext(ctx).Delete(&store.Address{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("address not found (id: %s)", id)
	}
	return nil
}

func (as *addressStore) SetDefault(ctx context.Context, addressID, userID string) error {
	// Start a transaction
	tx := as.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Unset all other defaults for this user
	if err := tx.Model(&store.Address{}).
		Where("user_id = ? AND id != ?", userID, addressID).
		Update("is_default", false).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Set the new default
	result := tx.Model(&store.Address{}).
		Where("id = ? AND user_id = ?", addressID, userID).
		Update("is_default", true)

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}
	if result.RowsAffected != 1 {
		tx.Rollback()
		return fmt.Errorf("address not found or does not belong to user")
	}

	return tx.Commit().Error
}
