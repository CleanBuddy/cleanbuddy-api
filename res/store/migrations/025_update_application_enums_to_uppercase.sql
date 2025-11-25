-- Migration: Update application enums to UPPERCASE
-- This aligns application_type and status with the UserRole enum convention

-- Update existing application_type values to UPPERCASE
UPDATE applications
SET application_type = CASE application_type
    WHEN 'cleaner' THEN 'CLEANER'
    WHEN 'company_admin' THEN 'COMPANY_ADMIN'
    ELSE UPPER(application_type)
END;

-- Update existing status values to UPPERCASE
UPDATE applications
SET status = CASE status
    WHEN 'pending' THEN 'PENDING'
    WHEN 'approved' THEN 'APPROVED'
    WHEN 'rejected' THEN 'REJECTED'
    ELSE UPPER(status)
END;

-- Drop old constraints
ALTER TABLE applications DROP CONSTRAINT IF EXISTS check_application_type;
ALTER TABLE applications DROP CONSTRAINT IF EXISTS check_application_status;

-- Add new constraints with UPPERCASE values
ALTER TABLE applications ADD CONSTRAINT check_application_type
    CHECK (application_type IN ('CLEANER', 'COMPANY_ADMIN'));

ALTER TABLE applications ADD CONSTRAINT check_application_status
    CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED'));
