CREATE TABLE app_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
) WITHOUT ROWID;

CREATE TABLE system_settings (
    key        TEXT    PRIMARY KEY,
    value      TEXT    NOT NULL CHECK (json_valid(value)),
    updated_at TEXT    NOT NULL,
    updated_by INTEGER REFERENCES users (id) ON DELETE SET NULL
) WITHOUT ROWID;

CREATE TABLE audit_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    at          TEXT    NOT NULL,
    event       TEXT    NOT NULL CHECK (length(event) <= 64),
    success     INTEGER NOT NULL DEFAULT 1 CHECK (success IN (0, 1)),
    actor_id    INTEGER REFERENCES users (id) ON DELETE SET NULL,
    actor_name  TEXT    NOT NULL DEFAULT '' CHECK (length(actor_name) <= 64),
    target_type TEXT    CHECK (target_type IN ('user', 'link', 'session', 'settings', 'tag')),
    target_id   INTEGER,
    target_name TEXT    NOT NULL DEFAULT '' CHECK (length(target_name) <= 64),
    ip          TEXT    NOT NULL DEFAULT '' CHECK (length(ip) <= 45),
    details     TEXT    NOT NULL DEFAULT '{}' CHECK (json_valid(details))
);

CREATE INDEX idx_audit_at ON audit_log (at);
CREATE INDEX idx_audit_event ON audit_log (event, id DESC);
CREATE INDEX idx_audit_actor ON audit_log (actor_id, id DESC);

CREATE TABLE consent_records (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    kind        TEXT    NOT NULL CHECK (kind IN ('privacy_notice')),
    version     INTEGER NOT NULL CHECK (version >= 1),
    accepted_at TEXT    NOT NULL,
    UNIQUE (user_id, kind, version)
);
