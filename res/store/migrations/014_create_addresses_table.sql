-- Migration 014: Create addresses table
-- Physical addresses for bookings and user profiles

CREATE TABLE IF NOT EXISTS addresses (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,

    -- Address Fields
    label VARCHAR(100), -- e.g., "Home", "Office"
    street VARCHAR(200) NOT NULL,
    building VARCHAR(50),
    apartment VARCHAR(20),
    floor INTEGER,
    city VARCHAR(100) NOT NULL,
    neighborhood VARCHAR(100),
    postal_code VARCHAR(20) NOT NULL,
    county VARCHAR(100), -- Romanian: Județ
    country VARCHAR(100) NOT NULL DEFAULT 'Romania',

    -- Additional Information
    access_instructions TEXT, -- e.g., "Ring bell 3 times", "Gate code: 1234"
    is_default BOOLEAN NOT NULL DEFAULT false,

    -- Coordinates (optional, for future map integration)
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_addresses_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Indexes for efficient queries
CREATE INDEX idx_address_user ON addresses(user_id);
CREATE INDEX idx_address_city ON addresses(city);
CREATE INDEX idx_address_default ON addresses(user_id, is_default) WHERE is_default = true;
