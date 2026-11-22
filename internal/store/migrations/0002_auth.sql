CREATE TABLE sessions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash   BLOB    NOT NULL UNIQUE CHECK (length(token_hash) = 32),
    user_id      INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at   TEXT    NOT NULL,
    last_seen_at TEXT    NOT NULL,
    expires_at   TEXT    NOT NULL,
    ip           TEXT    NOT NULL DEFAULT '' CHECK (length(ip) <= 45),
    user_agent   TEXT    NOT NULL DEFAULT '' CHECK (length(user_agent) <= 255)
);

CREATE INDEX idx_sessions_user ON sessions (user_id);
CREATE INDEX idx_sessions_expires ON sessions (expires_at);
CREATE INDEX idx_sessions_last_seen ON sessions (last_seen_at);

CREATE TABLE login_challenges (
    token_hash BLOB    PRIMARY KEY CHECK (length(token_hash) = 32),
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    attempts   INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    ip         TEXT    NOT NULL DEFAULT '' CHECK (length(ip) <= 45),
    created_at TEXT    NOT NULL,
    expires_at TEXT    NOT NULL
) WITHOUT ROWID;

CREATE INDEX idx_login_challenges_expires ON login_challenges (expires_at);

CREATE TABLE recovery_codes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash  TEXT    NOT NULL CHECK (length(code_hash) = 64),
    used_at    TEXT,
    created_at TEXT    NOT NULL,
    UNIQUE (user_id, code_hash)
);

CREATE TABLE login_throttle (
    key               TEXT    PRIMARY KEY CHECK (length(key) <= 100),
    user_id           INTEGER REFERENCES users (id) ON DELETE CASCADE,
    failures          INTEGER NOT NULL DEFAULT 0 CHECK (failures >= 0),
    window_started_at TEXT    NOT NULL,
    locked_until      TEXT,
    lockouts          INTEGER NOT NULL DEFAULT 0 CHECK (lockouts >= 0),
    updated_at        TEXT    NOT NULL
) WITHOUT ROWID;

CREATE INDEX idx_login_throttle_updated ON login_throttle (updated_at);
CREATE INDEX idx_login_throttle_user ON login_throttle (user_id) WHERE user_id IS NOT NULL;
