package postgresql

import (
	"context"
	"fmt"
	"time"

	"cleanbuddy-api/res/store"

	"gorm.io/gorm"
)

type transactionStore struct {
	*storeImpl
}

func NewTransactionStore(rootStore *storeImpl) *transactionStore {
	return &transactionStore{storeImpl: rootStore}
}

func (ts *transactionStore) Create(ctx context.Context, transaction *store.Transaction) error {
	result := ts.db.WithContext(ctx).Create(transaction)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create transaction")
	}
	return nil
}

func (ts *transactionStore) Get(ctx context.Context, id string) (*store.Transaction, error) {
	var transaction store.Transaction
	result := ts.db.WithContext(ctx).Where("id = ?", id).First(&transaction)
	if result.Error != nil {
		return nil, result.Error
	}
	return &transaction, nil
}

func (ts *transactionStore) GetByStripePaymentID(ctx context.Context, stripePaymentID string) (*store.Transaction, error) {
	var transaction store.Transaction
	result := ts.db.WithContext(ctx).Where("stripe_payment_id = ?", stripePaymentID).First(&transaction)
	if result.Error != nil {
		return nil, result.Error
	}
	return &transaction, nil
}

func (ts *transactionStore) Update(ctx context.Context, transaction *store.Transaction) error {
	result := ts.db.WithContext(ctx).Save(transaction)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("transaction not found (id: %s)", transaction.ID)
	}
	return nil
}

func (ts *transactionStore) UpdateStatus(ctx context.Context, transactionID string, status store.TransactionStatus) error {
	result := ts.db.WithContext(ctx).Model(&store.Transaction{}).
		Where("id = ?", transactionID).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("transaction not found (id: %s)", transactionID)
	}
	return nil
}

func (ts *transactionStore) GetByBooking(ctx context.Context, bookingID string) ([]*store.Transaction, error) {
	var transactions []*store.Transaction
	result := ts.db.WithContext(ctx).
		Where("booking_id = ?", bookingID).
		Order("created_at DESC").
		Find(&transactions)
	if result.Error != nil {
		return nil, result.Error
	}
	return transactions, nil
}

func (ts *transactionStore) GetByUser(ctx context.Context, userID string, filters store.TransactionFilters) ([]*store.Transaction, error) {
	query := ts.db.WithContext(ctx).
		Where("payer_id = ? OR payee_id = ?", userID, userID)
	query = ts.applyFilters(query, filters)

	var transactions []*store.Transaction
	if err := query.Find(&transactions).Error; err != nil {
		return nil, err
	}
	return transactions, nil
}

func (ts *transactionStore) GetPayoutsDue(ctx context.Context, beforeDate time.Time) ([]*store.Transaction, error) {
	var transactions []*store.Transaction
	err := ts.db.WithContext(ctx).
		Where("type = ? AND status = ? AND processed_at < ?",
			store.TransactionTypePayout,
			store.TransactionStatusPending,
			beforeDate).
		Order("processed_at ASC").
		Find(&transactions).Error

	if err != nil {
		return nil, err
	}
	return transactions, nil
}

func (ts *transactionStore) ListAll(ctx context.Context, filters store.TransactionFilters) ([]*store.Transaction, error) {
	query := ts.db.WithContext(ctx)
	query = ts.applyFilters(query, filters)

	var transactions []*store.Transaction
	if err := query.Find(&transactions).Error; err != nil {
		return nil, err
	}
	return transactions, nil
}

func (ts *transactionStore) CreatePayoutBatch(ctx context.Context, batch *store.PayoutBatch) error {
	result := ts.db.WithContext(ctx).Create(batch)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("failed to create payout batch")
	}
	return nil
}

func (ts *transactionStore) GetPayoutBatch(ctx context.Context, id string) (*store.PayoutBatch, error) {
	var batch store.PayoutBatch
	result := ts.db.WithContext(ctx).Where("id = ?", id).First(&batch)
	if result.Error != nil {
		return nil, result.Error
	}
	return &batch, nil
}

func (ts *transactionStore) ListPayoutBatches(ctx context.Context, limit, offset int) ([]*store.PayoutBatch, error) {
	var batches []*store.PayoutBatch
	query := ts.db.WithContext(ctx).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&batches).Error; err != nil {
		return nil, err
	}
	return batches, nil
}

func (ts *transactionStore) GetCleanerEarnings(ctx context.Context, cleanerID string, startDate, endDate time.Time) (int64, error) {
	var result struct {
		TotalEarnings int64
	}

	query := ts.db.WithContext(ctx).
		Model(&store.Transaction{}).
		Select("COALESCE(SUM(net_amount), 0) as total_earnings").
		Where("payee_id = ? AND type = ? AND status = ?",
			cleanerID,
			store.TransactionTypePayout,
			store.TransactionStatusCompleted)

	if !startDate.IsZero() {
		query = query.Where("completed_at >= ?", startDate)
	}
	if !endDate.IsZero() {
		query = query.Where("completed_at <= ?", endDate)
	}

	err := query.Scan(&result).Error
	if err != nil {
		return 0, err
	}

	return result.TotalEarnings, nil
}

// Helper method to apply filters
func (ts *transactionStore) applyFilters(query *gorm.DB, filters store.TransactionFilters) *gorm.DB {
	if filters.Type != nil {
		query = query.Where("type = ?", *filters.Type)
	}
	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}
	if filters.PaymentMethod != nil {
		query = query.Where("payment_method = ?", *filters.PaymentMethod)
	}
	if filters.StartDate != nil {
		query = query.Where("created_at >= ?", *filters.StartDate)
	}
	if filters.EndDate != nil {
		query = query.Where("created_at <= ?", *filters.EndDate)
	}
	if filters.MinAmount != nil {
		query = query.Where("amount >= ?", *filters.MinAmount)
	}
	if filters.MaxAmount != nil {
		query = query.Where("amount <= ?", *filters.MaxAmount)
	}

	if filters.OrderBy != "" {
		query = query.Order(filters.OrderBy)
	} else {
		query = query.Order("created_at DESC")
	}

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	return query
}
