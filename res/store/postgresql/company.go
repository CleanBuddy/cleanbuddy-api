package postgresql

import (
	"context"
	"fmt"

	"cleanbuddy-api/res/store"
)

type companyStore struct {
	*storeImpl
}

func NewCompanyStore(rootStore *storeImpl) *companyStore {
	return &companyStore{storeImpl: rootStore}
}

// MUTATIONS

func (cs *companyStore) Create(ctx context.Context, company *store.Company) error {
	result := cs.db.WithContext(ctx).Create(company)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create company")
	}
	return nil
}

func (cs *companyStore) Get(ctx context.Context, id string) (*store.Company, error) {
	var company store.Company
	result := cs.db.WithContext(ctx).Where("id = ?", id).First(&company)
	if result.Error != nil {
		return nil, result.Error
	}
	return &company, nil
}

func (cs *companyStore) GetByAdminUserID(ctx context.Context, adminUserID string) (*store.Company, error) {
	var company store.Company
	result := cs.db.WithContext(ctx).Where("admin_user_id = ?", adminUserID).First(&company)
	if result.Error != nil {
		return nil, result.Error
	}
	return &company, nil
}

func (cs *companyStore) Update(ctx context.Context, company *store.Company) error {
	result := cs.db.WithContext(ctx).Save(company)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("company not found (id: %s)", company.ID)
	}
	return nil
}

func (cs *companyStore) Delete(ctx context.Context, id string) error {
	result := cs.db.WithContext(ctx).Delete(&store.Company{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("company not found (id: %s)", id)
	}
	return nil
}

func (cs *companyStore) List(ctx context.Context) ([]*store.Company, error) {
	var companies []*store.Company
	result := cs.db.WithContext(ctx).Order("created_at DESC").Find(&companies)
	if result.Error != nil {
		return nil, result.Error
	}
	return companies, nil
}

func (cs *companyStore) UpdateStats(ctx context.Context, companyID string, stats store.CompanyStats) error {
	updates := make(map[string]interface{})

	if stats.TotalCleaners != nil {
		updates["total_cleaners"] = *stats.TotalCleaners
	}
	if stats.ActiveCleaners != nil {
		updates["active_cleaners"] = *stats.ActiveCleaners
	}

	if len(updates) == 0 {
		return fmt.Errorf("no stats to update")
	}

	result := cs.db.WithContext(ctx).Model(&store.Company{}).
		Where("id = ?", companyID).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("company not found (id: %s)", companyID)
	}

	return nil
}
