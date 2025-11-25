-- Add PENDING_APPLICATION role to the user role check constraint

-- Drop old check constraint
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_user_role;

-- Add new check constraint with PENDING_APPLICATION role
ALTER TABLE users ADD CONSTRAINT check_user_role
    CHECK (role IN ('CLIENT', 'PENDING_APPLICATION', 'PENDING_CLEANER', 'REJECTED_CLEANER', 'CLEANER', 'COMPANY_ADMIN', 'GLOBAL_ADMIN'));
