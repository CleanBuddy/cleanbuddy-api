CREATE TABLE IF NOT EXISTS invitation_codes (
    id VARCHAR(50) PRIMARY KEY,
    code VARCHAR(6) NOT NULL UNIQUE,
    owner_id VARCHAR(50) NOT NULL,
    redeemed_at TIMESTAMP,
    redeemed_by_id VARCHAR(50),
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_invitation_codes_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT fk_invitation_codes_redeemed_by FOREIGN KEY (redeemed_by_id) REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE
);

CREATE INDEX idx_invitation_codes_code ON invitation_codes(code);
CREATE INDEX idx_invitation_codes_owner_id ON invitation_codes(owner_id);
CREATE INDEX idx_invitation_codes_redeemed_by_id ON invitation_codes(redeemed_by_id);
