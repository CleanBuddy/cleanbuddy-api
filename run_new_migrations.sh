#!/bin/bash

# Script to run the new CleanBuddy marketplace migrations (012-021)
# These migrations add cleaner profiles, bookings, services, and related tables

set -e  # Exit on any error

# Load environment variables
source .env

# Check if DATABASE_POSTGRES_URL is set
if [ -z "$DATABASE_POSTGRES_URL" ]; then
    echo "Error: DATABASE_POSTGRES_URL not set in .env"
    exit 1
fi

echo "Running CleanBuddy marketplace migrations..."
echo "=========================================="

# Array of migration files in order
migrations=(
    "012_create_cleaner_profiles_table.sql"
    "013_create_service_areas_table.sql"
    "014_create_addresses_table.sql"
    "015_create_service_definitions_tables.sql"
    "016_create_bookings_table.sql"
    "017_create_reviews_table.sql"
    "018_create_transactions_tables.sql"
    "019_create_availability_table.sql"
    "020_seed_service_definitions.sql"
    "021_add_google_place_id_to_addresses.sql"
)

# Run each migration
for migration in "${migrations[@]}"; do
    echo ""
    echo "Running migration: $migration"
    echo "----------------------------------------"

    psql "$DATABASE_POSTGRES_URL" -f "res/store/migrations/$migration"

    if [ $? -eq 0 ]; then
        echo "✓ Successfully applied: $migration"
    else
        echo "✗ Failed to apply: $migration"
        exit 1
    fi
done

echo ""
echo "=========================================="
echo "All migrations completed successfully!"
echo ""
echo "New tables created:"
echo "  - cleaner_profiles"
echo "  - service_areas"
echo "  - addresses"
echo "  - service_definitions"
echo "  - service_add_ons"
echo "  - bookings"
echo "  - reviews"
echo "  - transactions"
echo "  - payout_batches"
echo "  - availabilities"
echo ""
echo "Seed data added:"
echo "  - 3 service types (General Cleaning, Deep Cleaning, Move-in/out)"
echo "  - 4 add-ons (Oven, Windows, Fridge, Garage)"
echo ""
