package postgresql

import (
	"context"
	"fmt"

	"cleanbuddy-api/res/store"
)

type serviceStore struct {
	*storeImpl
}

func NewServiceStore(rootStore *storeImpl) *serviceStore {
	return &serviceStore{storeImpl: rootStore}
}

// Service Definitions

func (ss *serviceStore) GetServiceDefinition(ctx context.Context, serviceType store.ServiceType) (*store.ServiceDefinition, error) {
	var service store.ServiceDefinition
	result := ss.db.WithContext(ctx).Where("type = ?", serviceType).First(&service)
	if result.Error != nil {
		return nil, result.Error
	}
	return &service, nil
}

func (ss *serviceStore) ListServiceDefinitions(ctx context.Context, activeOnly bool) ([]*store.ServiceDefinition, error) {
	var services []*store.ServiceDefinition
	query := ss.db.WithContext(ctx)

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Order("type ASC").Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}

func (ss *serviceStore) CreateServiceDefinition(ctx context.Context, service *store.ServiceDefinition) error {
	result := ss.db.WithContext(ctx).Create(service)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create service definition")
	}
	return nil
}

func (ss *serviceStore) UpdateServiceDefinition(ctx context.Context, service *store.ServiceDefinition) error {
	result := ss.db.WithContext(ctx).Save(service)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("service definition not found (id: %s)", service.ID)
	}
	return nil
}

// Add-On Definitions

func (ss *serviceStore) GetAddOnDefinition(ctx context.Context, addOn store.ServiceAddOn) (*store.ServiceAddOnDefinition, error) {
	var addOnDef store.ServiceAddOnDefinition
	result := ss.db.WithContext(ctx).Where("add_on = ?", addOn).First(&addOnDef)
	if result.Error != nil {
		return nil, result.Error
	}
	return &addOnDef, nil
}

func (ss *serviceStore) ListAddOnDefinitions(ctx context.Context, activeOnly bool) ([]*store.ServiceAddOnDefinition, error) {
	var addOns []*store.ServiceAddOnDefinition
	query := ss.db.WithContext(ctx)

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	if err := query.Order("add_on ASC").Find(&addOns).Error; err != nil {
		return nil, err
	}
	return addOns, nil
}

func (ss *serviceStore) CreateAddOnDefinition(ctx context.Context, addOn *store.ServiceAddOnDefinition) error {
	result := ss.db.WithContext(ctx).Create(addOn)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create add-on definition")
	}
	return nil
}

func (ss *serviceStore) UpdateAddOnDefinition(ctx context.Context, addOn *store.ServiceAddOnDefinition) error {
	result := ss.db.WithContext(ctx).Save(addOn)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("add-on definition not found (id: %s)", addOn.ID)
	}
	return nil
}
