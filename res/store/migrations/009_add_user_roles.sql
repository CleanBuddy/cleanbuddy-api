-- Migration 009: Add user roles and remove status field
-- This migration transitions from status-based (pending/active/suspended) to role-based access control

-- Add role column with default 'client'
ALTER TABLE users ADD COLUMN role VARCHAR(20) NOT NULL DEFAULT 'client';

-- Create index for efficient role-based queries
CREATE INDEX idx_users_role ON users(role);

-- Remove old status column (replaced by role system)
ALTER TABLE users DROP COLUMN status;

-- Add check constraint to ensure valid roles
ALTER TABLE users ADD CONSTRAINT check_user_role
    CHECK (role IN ('client', 'cleaner', 'company_admin', 'global_admin'));
