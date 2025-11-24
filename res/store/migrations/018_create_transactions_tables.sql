-- Migration 018: Create transactions and payout_batches tables
-- Financial transaction tracking for payments, payouts, and refunds

CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(50) PRIMARY KEY,
    type VARCHAR(20) NOT NULL, -- 'payment', 'payout', 'refund'
    status VARCHAR(20) NOT NULL, -- 'pending', 'processing', 'completed', 'failed', 'cancelled'

    -- Related Entities
    booking_id VARCHAR(50),

    -- Payer and Payee
    payer_id VARCHAR(50) NOT NULL,
    payee_id VARCHAR(50) NOT NULL,

    -- Amount Details (in bani)
    amount INTEGER NOT NULL,
    platform_fee INTEGER NOT NULL DEFAULT 0,
    net_amount INTEGER NOT NULL,

    -- Payment Details
    payment_method VARCHAR(30) NOT NULL, -- 'card', 'bank_transfer', 'cash'
    currency VARCHAR(10) NOT NULL DEFAULT 'RON',

    -- External Payment Provider Data (Stripe)
    stripe_payment_id VARCHAR(256) UNIQUE,
    stripe_transfer_id VARCHAR(256) UNIQUE,
    stripe_refund_id VARCHAR(256) UNIQUE,

    -- Metadata
    description TEXT,
    metadata TEXT, -- JSON for additional data

    -- Failure Information
    failure_reason TEXT,
    failure_code VARCHAR(100),

    -- Timestamps
    processed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    failed_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_transactions_booking FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT fk_transactions_payer FOREIGN KEY (payer_id) REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_transactions_payee FOREIGN KEY (payee_id) REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT check_type CHECK (type IN ('payment', 'payout', 'refund')),
    CONSTRAINT check_status CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    CONSTRAINT check_payment_method CHECK (payment_method IN ('card', 'bank_transfer', 'cash')),
    CONSTRAINT check_amounts CHECK (
        amount >= 0 AND
        platform_fee >= 0 AND
        net_amount >= 0
    )
);

CREATE TABLE IF NOT EXISTS payout_batches (
    id VARCHAR(50) PRIMARY KEY,
    status VARCHAR(20) NOT NULL, -- 'pending', 'processing', 'completed', 'failed', 'cancelled'

    -- Batch Details
    total_amount INTEGER NOT NULL, -- Total amount in batch (in bani)
    total_payouts INTEGER NOT NULL, -- Number of payouts in batch
    period_start TIMESTAMP NOT NULL, -- Start of payout period
    period_end TIMESTAMP NOT NULL, -- End of payout period

    -- Processing
    initiated_by_id VARCHAR(50) NOT NULL,
    processed_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Metadata
    notes TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_payout_batches_initiated_by FOREIGN KEY (initiated_by_id) REFERENCES users(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT check_batch_status CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    CONSTRAINT check_batch_amounts CHECK (
        total_amount >= 0 AND
        total_payouts >= 0
    )
);

-- Indexes for efficient queries
CREATE INDEX idx_transaction_type ON transactions(type);
CREATE INDEX idx_transaction_status ON transactions(status);
CREATE INDEX idx_transaction_booking ON transactions(booking_id);
CREATE INDEX idx_transaction_payer ON transactions(payer_id);
CREATE INDEX idx_transaction_payee ON transactions(payee_id);
CREATE INDEX idx_transaction_stripe_payment ON transactions(stripe_payment_id);
CREATE INDEX idx_transaction_created ON transactions(created_at);
CREATE INDEX idx_transaction_processed ON transactions(processed_at);

-- Composite indexes for common query patterns
CREATE INDEX idx_transaction_type_status ON transactions(type, status);
CREATE INDEX idx_transaction_payee_completed ON transactions(payee_id, status, completed_at) WHERE status = 'completed';
