package graphql

import (
	"context"
	"errors"
	"time"

	"cleanbuddy-api/res/store"
	"cleanbuddy-api/sys/graphql/gen"
)

// QUERY RESOLVERS
func (qr *queryResolver) Transaction(ctx context.Context, id string) (*store.Transaction, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) TransactionByStripePaymentID(ctx context.Context, stripePaymentID string) (*store.Transaction, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) TransactionsByBooking(ctx context.Context, bookingID string) ([]*store.Transaction, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) MyTransactions(ctx context.Context, filters *gen.TransactionFiltersInput, limit, offset *int, orderBy *string) (*gen.TransactionConnection, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) MyEarnings(ctx context.Context, startDate, endDate *time.Time) (*gen.CleanerEarnings, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) AllTransactions(ctx context.Context, filters *gen.TransactionFiltersInput, limit, offset *int, orderBy *string) (*gen.TransactionConnection, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) TransactionsDueForPayout(ctx context.Context, beforeDate time.Time) ([]*store.Transaction, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) PayoutBatch(ctx context.Context, id string) (*store.PayoutBatch, error) {
	return nil, errors.New("not yet implemented")
}

func (qr *queryResolver) PayoutBatches(ctx context.Context, limit, offset *int) ([]*store.PayoutBatch, error) {
	return nil, errors.New("not yet implemented")
}

// MUTATION RESOLVERS
func (mr *mutationResolver) CreatePayoutBatch(ctx context.Context, input gen.CreatePayoutBatchInput) (*store.PayoutBatch, error) {
	return nil, errors.New("not yet implemented")
}

func (mr *mutationResolver) ProcessPayoutBatch(ctx context.Context, id string) (*store.PayoutBatch, error) {
	return nil, errors.New("not yet implemented")
}
