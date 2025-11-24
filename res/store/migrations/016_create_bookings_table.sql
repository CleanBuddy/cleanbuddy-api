-- Migration 016: Create bookings table
-- Core booking system for cleaning services

CREATE TABLE IF NOT EXISTS bookings (
    id VARCHAR(50) PRIMARY KEY,
    customer_id VARCHAR(50) NOT NULL,
    cleaner_id VARCHAR(50) NOT NULL,
    cleaner_profile_id VARCHAR(50) NOT NULL,

    -- Service Details
    service_type VARCHAR(20) NOT NULL, -- 'general', 'deep', 'move_in_out'
    service_frequency VARCHAR(20) NOT NULL, -- 'one_time', 'weekly', 'bi_monthly', 'monthly'
    service_add_ons TEXT, -- JSON array of add-on values

    -- Scheduling
    scheduled_date DATE NOT NULL,
    scheduled_time VARCHAR(10) NOT NULL, -- e.g., "14:00"
    duration DECIMAL(4,2) NOT NULL, -- Duration in hours

    -- Address
    address_id VARCHAR(50) NOT NULL,

    -- Pricing (in bani - stored at booking time to preserve historical pricing)
    cleaner_hourly_rate INTEGER NOT NULL,
    service_price INTEGER NOT NULL,
    add_ons_price INTEGER NOT NULL DEFAULT 0,
    travel_fee INTEGER NOT NULL DEFAULT 0,
    platform_fee INTEGER NOT NULL,
    total_price INTEGER NOT NULL,
    cleaner_payout INTEGER NOT NULL,

    -- Status and Progress
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'confirmed', 'in_progress', 'completed', 'cancelled', 'no_show'
    cancellation_reason VARCHAR(30),
    cancellation_note TEXT,
    cancelled_by_id VARCHAR(50),
    cancelled_at TIMESTAMP,

    -- Timestamps
    confirmed_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Recurring Booking Support
    is_recurring BOOLEAN NOT NULL DEFAULT false,
    parent_booking_id VARCHAR(50), -- References parent for recurring bookings
    next_booking_id VARCHAR(50), -- References next booking in series

    -- Special Instructions
    customer_notes TEXT,
    cleaner_notes TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_bookings_customer FOREIGN KEY (customer_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_bookings_cleaner FOREIGN KEY (cleaner_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_bookings_cleaner_profile FOREIGN KEY (cleaner_profile_id) REFERENCES cleaner_profiles(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_bookings_address FOREIGN KEY (address_id) REFERENCES addresses(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_bookings_cancelled_by FOREIGN KEY (cancelled_by_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT fk_bookings_parent FOREIGN KEY (parent_booking_id) REFERENCES bookings(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT fk_bookings_next FOREIGN KEY (next_booking_id) REFERENCES bookings(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT check_service_type CHECK (service_type IN ('general', 'deep', 'move_in_out')),
    CONSTRAINT check_service_frequency CHECK (service_frequency IN ('one_time', 'weekly', 'bi_monthly', 'monthly')),
    CONSTRAINT check_status CHECK (status IN ('pending', 'confirmed', 'in_progress', 'completed', 'cancelled', 'no_show')),
    CONSTRAINT check_cancellation_reason CHECK (cancellation_reason IN ('customer_request', 'cleaner_request', 'emergency', 'weather', 'other') OR cancellation_reason IS NULL),
    CONSTRAINT check_duration CHECK (duration > 0),
    CONSTRAINT check_pricing CHECK (
        cleaner_hourly_rate >= 0 AND
        service_price >= 0 AND
        add_ons_price >= 0 AND
        travel_fee >= 0 AND
        platform_fee >= 0 AND
        total_price >= 0 AND
        cleaner_payout >= 0
    )
);

-- Indexes for efficient queries
CREATE INDEX idx_booking_customer ON bookings(customer_id);
CREATE INDEX idx_booking_cleaner ON bookings(cleaner_id);
CREATE INDEX idx_booking_status ON bookings(status);
CREATE INDEX idx_booking_date ON bookings(scheduled_date);
CREATE INDEX idx_booking_created ON bookings(created_at);
CREATE INDEX idx_booking_parent ON bookings(parent_booking_id);

-- Composite indexes for common query patterns
CREATE INDEX idx_booking_customer_status ON bookings(customer_id, status);
CREATE INDEX idx_booking_cleaner_status ON bookings(cleaner_id, status);
CREATE INDEX idx_booking_cleaner_date ON bookings(cleaner_id, scheduled_date);
