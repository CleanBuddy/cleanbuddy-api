-- Migration 021: Add google_place_id to addresses table
-- This stores the Google Maps Place ID for verification and future lookups

ALTER TABLE addresses
ADD COLUMN IF NOT EXISTS google_place_id VARCHAR(256);

-- Add index for place ID lookups
CREATE INDEX IF NOT EXISTS idx_address_place_id ON addresses(google_place_id);
