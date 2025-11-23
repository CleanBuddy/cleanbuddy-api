-- Migration 010: Create applications table
-- Stores cleaner and company admin role applications with approval workflow

CREATE TABLE IF NOT EXISTS applications (
    id VARCHAR(50) PRIMARY KEY,
    user_id VARCHAR(50) NOT NULL,
    application_type VARCHAR(20) NOT NULL, -- 'cleaner' or 'company_admin'
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    message TEXT, -- Expandable field for future form data (JSON or structured text)
    reviewed_by_id VARCHAR(50), -- Global admin who reviewed the application
    reviewed_at TIMESTAMP, -- When the application was reviewed
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_applications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_applications_reviewed_by FOREIGN KEY (reviewed_by_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT check_application_type CHECK (application_type IN ('cleaner', 'company_admin')),
    CONSTRAINT check_application_status CHECK (status IN ('pending', 'approved', 'rejected'))
);

-- Indexes for efficient queries
CREATE INDEX idx_applications_user_id ON applications(user_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_applications_type ON applications(application_type);
CREATE INDEX idx_applications_reviewed_by_id ON applications(reviewed_by_id);

-- Composite index for finding user's pending applications
CREATE INDEX idx_applications_user_status ON applications(user_id, status);
