-- Migration 019: Create availability table
-- Cleaner availability schedule management

CREATE TABLE IF NOT EXISTS availabilities (
    id VARCHAR(50) PRIMARY KEY,
    cleaner_profile_id VARCHAR(50) NOT NULL,

    -- Type
    type VARCHAR(20) NOT NULL, -- 'available', 'unavailable'

    -- Date and Time
    date DATE NOT NULL,
    start_time VARCHAR(10) NOT NULL, -- e.g., "09:00"
    end_time VARCHAR(10) NOT NULL, -- e.g., "17:00"

    -- Recurrence
    is_recurring BOOLEAN NOT NULL DEFAULT false,
    recurrence_pattern VARCHAR(20), -- 'none', 'weekly'
    recurrence_end TIMESTAMP, -- When recurrence ends (null = indefinite)

    -- Notes
    notes TEXT, -- e.g., "Vacation", "Personal appointment"

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_availabilities_cleaner FOREIGN KEY (cleaner_profile_id) REFERENCES cleaner_profiles(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT check_type CHECK (type IN ('available', 'unavailable')),
    CONSTRAINT check_recurrence_pattern CHECK (recurrence_pattern IN ('none', 'weekly') OR recurrence_pattern IS NULL)
);

-- Indexes for efficient availability queries
CREATE INDEX idx_availability_cleaner ON availabilities(cleaner_profile_id);
CREATE INDEX idx_availability_date ON availabilities(date);
CREATE INDEX idx_availability_type ON availabilities(type);

-- Composite index for finding cleaner availability on specific dates
CREATE INDEX idx_availability_cleaner_date ON availabilities(cleaner_profile_id, date);
