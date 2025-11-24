-- Migration 015: Create service_definitions and service_add_on_definitions tables
-- Service types and add-ons with pricing modifiers

CREATE TABLE IF NOT EXISTS service_definitions (
    id VARCHAR(50) PRIMARY KEY,
    type VARCHAR(20) NOT NULL UNIQUE, -- 'general', 'deep', 'move_in_out'

    -- Service Details
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Duration and Pricing Modifiers
    base_hours DECIMAL(4,2) NOT NULL, -- Base hours for the service
    price_multiplier DECIMAL(4,2) NOT NULL DEFAULT 1.0, -- Multiplier for cleaner's hourly rate

    -- Availability
    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_service_type CHECK (type IN ('general', 'deep', 'move_in_out')),
    CONSTRAINT check_base_hours CHECK (base_hours > 0),
    CONSTRAINT check_price_multiplier CHECK (price_multiplier > 0)
);

CREATE TABLE IF NOT EXISTS service_add_on_definitions (
    id VARCHAR(50) PRIMARY KEY,
    add_on VARCHAR(20) NOT NULL UNIQUE, -- 'oven', 'windows', 'fridge', 'garage'

    -- Add-On Details
    name VARCHAR(100) NOT NULL,
    description TEXT,

    -- Pricing
    fixed_price INTEGER NOT NULL, -- Fixed price in bani (can be 0 if time-based)
    estimated_hours DECIMAL(4,2) NOT NULL DEFAULT 0, -- Additional hours needed

    -- Availability
    is_active BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT check_addon_type CHECK (add_on IN ('oven', 'windows', 'fridge', 'garage')),
    CONSTRAINT check_fixed_price CHECK (fixed_price >= 0),
    CONSTRAINT check_estimated_hours CHECK (estimated_hours >= 0)
);

-- Indexes
CREATE INDEX idx_service_type ON service_definitions(type);
CREATE INDEX idx_addon_type ON service_add_on_definitions(add_on);
