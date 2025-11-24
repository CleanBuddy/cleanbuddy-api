-- Migration 013: Create service_areas table
-- Geographic areas where cleaners provide services

CREATE TABLE IF NOT EXISTS service_areas (
    id VARCHAR(50) PRIMARY KEY,
    cleaner_profile_id VARCHAR(50) NOT NULL,

    -- Location Information
    city VARCHAR(100) NOT NULL,
    neighborhood VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,

    -- Travel Settings
    travel_fee INTEGER NOT NULL DEFAULT 0, -- Travel fee in bani
    is_preferred BOOLEAN NOT NULL DEFAULT false, -- Preferred area (cleaner lives here or nearby)

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_service_areas_cleaner FOREIGN KEY (cleaner_profile_id) REFERENCES cleaner_profiles(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT check_travel_fee CHECK (travel_fee >= 0)
);

-- Indexes for location-based searches
CREATE INDEX idx_service_area_cleaner ON service_areas(cleaner_profile_id);
CREATE INDEX idx_service_area_city ON service_areas(city);
CREATE INDEX idx_service_area_postal ON service_areas(postal_code);
CREATE INDEX idx_service_area_city_neighborhood ON service_areas(city, neighborhood);
