-- Drop old check constraint with lowercase values
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_user_role;

-- Update all user role values from lowercase to uppercase to match GraphQL schema
UPDATE users
SET role = CASE role
    WHEN 'client' THEN 'CLIENT'
    WHEN 'pending_cleaner' THEN 'PENDING_CLEANER'
    WHEN 'rejected_cleaner' THEN 'REJECTED_CLEANER'
    WHEN 'cleaner' THEN 'CLEANER'
    WHEN 'company_admin' THEN 'COMPANY_ADMIN'
    WHEN 'global_admin' THEN 'GLOBAL_ADMIN'
    ELSE role
END
WHERE role IN ('client', 'pending_cleaner', 'rejected_cleaner', 'cleaner', 'company_admin', 'global_admin');

-- Add new check constraint with uppercase values
ALTER TABLE users ADD CONSTRAINT check_user_role
    CHECK (role IN ('CLIENT', 'PENDING_CLEANER', 'REJECTED_CLEANER', 'CLEANER', 'COMPANY_ADMIN', 'GLOBAL_ADMIN'));

-- Update default value for role column
ALTER TABLE users ALTER COLUMN role SET DEFAULT 'CLIENT';
