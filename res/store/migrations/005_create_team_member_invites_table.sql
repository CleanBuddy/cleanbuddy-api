CREATE TABLE IF NOT EXISTS team_member_invites (
    id VARCHAR(50) PRIMARY KEY,
    team_id VARCHAR(50) NOT NULL,
    invited_by_id VARCHAR(50) NOT NULL,
    invitee_email VARCHAR(255) NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_team_member_invites_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_team_member_invites_invited_by FOREIGN KEY (invited_by_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE SET NULL
);

CREATE INDEX idx_team_member_invites_team_id ON team_member_invites(team_id);
CREATE INDEX idx_team_member_invites_invitee_email ON team_member_invites(invitee_email);
CREATE INDEX idx_team_member_invites_created_at ON team_member_invites(created_at);
