CREATE TABLE IF NOT EXISTS tasks (
    id SERIAL PRIMARY KEY,
    payload JSONB NOT NULL,
    type TEXT NOT NULL,
    progress INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3
);

CREATE INDEX idx_tasks_type ON tasks(type);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);
CREATE INDEX idx_tasks_type_status ON tasks(type, status);
CREATE INDEX idx_tasks_payload_project_id ON tasks((payload->>'project_id'));
