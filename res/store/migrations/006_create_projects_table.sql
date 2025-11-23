CREATE TABLE IF NOT EXISTS projects (
    id VARCHAR(50) PRIMARY KEY,
    display_name VARCHAR(50) NOT NULL,
    subdomain VARCHAR(63) UNIQUE NOT NULL,
    team_id VARCHAR(50) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_projects_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX idx_projects_team_id ON projects(team_id);
CREATE INDEX idx_projects_subdomain ON projects(subdomain);
CREATE INDEX idx_projects_created_at ON projects(created_at);
