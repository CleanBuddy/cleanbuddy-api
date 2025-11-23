CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(50) PRIMARY KEY,
    display_name VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    google_identity VARCHAR(256) UNIQUE,
    email VARCHAR(256) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_google_identity ON users(google_identity);
CREATE INDEX idx_users_status ON users(status);
