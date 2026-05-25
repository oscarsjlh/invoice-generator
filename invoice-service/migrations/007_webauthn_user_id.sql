ALTER TABLE users ADD COLUMN webauthn_user_id TEXT;

UPDATE users SET webauthn_user_id = CAST(id AS TEXT) WHERE webauthn_user_id IS NULL OR webauthn_user_id = '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_webauthn_user_id ON users(webauthn_user_id);

