CREATE TABLE users (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    username                TEXT    NOT NULL CHECK (length(username) BETWEEN 3 AND 64),
    username_key            TEXT    NOT NULL UNIQUE,
    display_name            TEXT    NOT NULL DEFAULT '' CHECK (length(display_name) <= 100),
    role                    TEXT    NOT NULL CHECK (role IN ('admin', 'developer', 'customer')),
    password_hash           TEXT    NOT NULL,
    must_change_password    INTEGER NOT NULL DEFAULT 0 CHECK (must_change_password IN (0, 1)),
    is_active               INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    totp_secret             TEXT,
    totp_enabled_at         TEXT,
    totp_last_step          INTEGER NOT NULL DEFAULT 0,
    totp_pending_secret     TEXT,
    totp_pending_expires_at TEXT,
    password_changed_at     TEXT    NOT NULL,
    last_login_at           TEXT,
    created_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK ((totp_secret IS NULL) = (totp_enabled_at IS NULL)),
    CHECK ((totp_pending_secret IS NULL) = (totp_pending_expires_at IS NULL))
);

CREATE INDEX idx_users_role_active ON users (role, is_active);

CREATE TABLE user_preferences (
    user_id            INTEGER PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    locale             TEXT    NOT NULL DEFAULT 'hu'    CHECK (locale IN ('hu', 'en', 'de')),
    theme              TEXT    NOT NULL DEFAULT 'dark'  CHECK (theme IN ('dark', 'light', 'system')),
    view_mode          TEXT    NOT NULL DEFAULT 'list'  CHECK (view_mode IN ('list', 'tiles')),
    show_icon          INTEGER NOT NULL DEFAULT 1 CHECK (show_icon IN (0, 1)),
    show_url           INTEGER NOT NULL DEFAULT 1 CHECK (show_url IN (0, 1)),
    show_title         INTEGER NOT NULL DEFAULT 1 CHECK (show_title IN (0, 1)),
    show_description   INTEGER NOT NULL DEFAULT 1 CHECK (show_description IN (0, 1)),
    show_archived      INTEGER NOT NULL DEFAULT 0 CHECK (show_archived IN (0, 1)),
    sort               TEXT    NOT NULL DEFAULT 'usage' CHECK (sort IN ('usage', 'created', 'title')),
    tag_match          TEXT    NOT NULL DEFAULT 'any'   CHECK (tag_match IN ('any', 'all')),
    archive_after_days INTEGER CHECK (archive_after_days IS NULL OR archive_after_days BETWEEN 0 AND 3650),
    updated_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
