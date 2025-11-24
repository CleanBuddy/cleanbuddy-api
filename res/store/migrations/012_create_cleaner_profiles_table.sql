-- Migration 012: Create cleaner_profiles table
-- Extended profile for users with cleaner role, includes tier system and performance tracking

CREATE TABLE IF NOT EXISTS cleaner_profiles (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL UNIQUE,

    -- Profile Information
    bio TEXT,
    profile_picture VARCHAR(512),

    -- Tier and Performance
    tier VARCHAR(20) NOT NULL DEFAULT 'new', -- 'new', 'standard', 'premium', 'pro'
    hourly_rate INTEGER NOT NULL, -- Rate in bani (100 bani = 1 RON)
    total_bookings INTEGER NOT NULL DEFAULT 0,
    completed_bookings INTEGER NOT NULL DEFAULT 0,
    cancelled_bookings INTEGER NOT NULL DEFAULT 0,
    average_rating DECIMAL(3,2) NOT NULL DEFAULT 0.00, -- 0.00 to 5.00
    total_reviews INTEGER NOT NULL DEFAULT 0,
    total_earnings BIGINT NOT NULL DEFAULT 0, -- Total earnings in bani

    -- Availability
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_available_today BOOLEAN NOT NULL DEFAULT false,

    -- Verification
    is_verified BOOLEAN NOT NULL DEFAULT false,
    verified_at TIMESTAMP,
    background_check BOOLEAN NOT NULL DEFAULT false,
    identity_verified BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_cleaner_profiles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT check_tier CHECK (tier IN ('new', 'standard', 'premium', 'pro')),
    CONSTRAINT check_hourly_rate CHECK (hourly_rate >= 0),
    CONSTRAINT check_average_rating CHECK (average_rating >= 0 AND average_rating <= 5)
);

-- Indexes for efficient queries
CREATE INDEX idx_cleaner_profile_user ON cleaner_profiles(user_id);
CREATE INDEX idx_cleaner_tier ON cleaner_profiles(tier);
CREATE INDEX idx_cleaner_created ON cleaner_profiles(created_at);
CREATE INDEX idx_cleaner_rating ON cleaner_profiles(average_rating DESC);
CREATE INDEX idx_cleaner_active ON cleaner_profiles(is_active) WHERE is_active = true;
