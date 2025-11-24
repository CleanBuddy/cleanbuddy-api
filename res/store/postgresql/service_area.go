package postgresql

import (
	"context"
	"fmt"

	"cleanbuddy-api/res/store"
)

type serviceAreaStore struct {
	*storeImpl
}

func NewServiceAreaStore(rootStore *storeImpl) *serviceAreaStore {
	return &serviceAreaStore{storeImpl: rootStore}
}

func (sas *serviceAreaStore) Create(ctx context.Context, area *store.ServiceArea) error {
	result := sas.db.WithContext(ctx).Create(area)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create service area")
	}
	return nil
}

func (sas *serviceAreaStore) Get(ctx context.Context, id string) (*store.ServiceArea, error) {
	var area store.ServiceArea
	result := sas.db.WithContext(ctx).Where("id = ?", id).First(&area)
	if result.Error != nil {
		return nil, result.Error
	}
	return &area, nil
}

func (sas *serviceAreaStore) GetByCleanerProfile(ctx context.Context, cleanerProfileID string) ([]*store.ServiceArea, error) {
	var areas []*store.ServiceArea
	result := sas.db.WithContext(ctx).Where("cleaner_profile_id = ?", cleanerProfileID).Find(&areas)
	if result.Error != nil {
		return nil, result.Error
	}
	return areas, nil
}

func (sas *serviceAreaStore) Update(ctx context.Context, area *store.ServiceArea) error {
	result := sas.db.WithContext(ctx).Save(area)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("service area not found (id: %s)", area.ID)
	}
	return nil
}

func (sas *serviceAreaStore) Delete(ctx context.Context, id string) error {
	result := sas.db.WithContext(ctx).Delete(&store.ServiceArea{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("service area not found (id: %s)", id)
	}
	return nil
}

func (sas *serviceAreaStore) FindCleanersInArea(ctx context.Context, city, neighborhood string) ([]*store.CleanerProfile, error) {
	var profiles []*store.CleanerProfile

	query := sas.db.WithContext(ctx).
		Joins("INNER JOIN service_areas ON service_areas.cleaner_profile_id = cleaner_profiles.id").
		Where("service_areas.city = ?", city)

	if neighborhood != "" {
		query = query.Where("service_areas.neighborhood = ?", neighborhood)
	}

	query = query.Where("cleaner_profiles.is_active = ?", true).
		Distinct().
		Order("cleaner_profiles.average_rating DESC")

	if err := query.Find(&profiles).Error; err != nil {
		return nil, err
	}

	return profiles, nil
}

func (sas *serviceAreaStore) FindCleanersByPostalCode(ctx context.Context, postalCode string) ([]*store.CleanerProfile, error) {
	var profiles []*store.CleanerProfile

	err := sas.db.WithContext(ctx).
		Joins("INNER JOIN service_areas ON service_areas.cleaner_profile_id = cleaner_profiles.id").
		Where("service_areas.postal_code = ?", postalCode).
		Where("cleaner_profiles.is_active = ?", true).
		Distinct().
		Order("cleaner_profiles.average_rating DESC").
		Find(&profiles).Error

	if err != nil {
		return nil, err
	}

	return profiles, nil
}
