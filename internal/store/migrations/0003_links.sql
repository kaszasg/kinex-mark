CREATE TABLE icons (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    sha256       TEXT    NOT NULL UNIQUE CHECK (length(sha256) = 64),
    content_type TEXT    NOT NULL CHECK (content_type IN (
                     'image/png', 'image/x-icon', 'image/gif', 'image/jpeg', 'image/webp', 'image/svg+xml')),
    data         BLOB    NOT NULL CHECK (length(data) BETWEEN 1 AND 65536),
    created_at   TEXT    NOT NULL
);

CREATE TABLE links (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id       INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    url            TEXT    NOT NULL CHECK (length(url) BETWEEN 9 AND 2048),
    host           TEXT    NOT NULL CHECK (length(host) BETWEEN 1 AND 253),
    title          TEXT    NOT NULL DEFAULT '' CHECK (length(title) <= 500),
    title_source   TEXT    NOT NULL DEFAULT 'none' CHECK (title_source IN ('none', 'fetched', 'manual')),
    description    TEXT    NOT NULL DEFAULT '' CHECK (length(description) <= 500),
    search_text    TEXT    NOT NULL DEFAULT '',
    sort_label     TEXT    NOT NULL DEFAULT '',
    icon_id        INTEGER REFERENCES icons (id) ON DELETE SET NULL,
    fetch_status   TEXT    NOT NULL DEFAULT 'pending' CHECK (fetch_status IN ('pending', 'ok', 'failed', 'blocked')),
    fetch_error    TEXT    CHECK (fetch_error IS NULL OR length(fetch_error) <= 40),
    fetch_attempts INTEGER NOT NULL DEFAULT 0 CHECK (fetch_attempts >= 0),
    next_fetch_at  TEXT,
    fetched_at     TEXT,
    created_at     TEXT    NOT NULL,
    updated_at     TEXT    NOT NULL,
    UNIQUE (owner_id, url)
);

CREATE INDEX idx_links_owner_created ON links (owner_id, created_at DESC);
CREATE INDEX idx_links_fetch_due ON links (next_fetch_at) WHERE fetch_status = 'pending';
CREATE INDEX idx_links_icon ON links (icon_id);

CREATE TABLE tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_id   INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name       TEXT    NOT NULL CHECK (length(name) BETWEEN 1 AND 40),
    name_key   TEXT    NOT NULL,
    color      TEXT    NOT NULL DEFAULT 'neutral' CHECK (color IN (
                   'blue', 'ochre', 'green', 'violet', 'rose', 'neutral', 'contrast')),
    created_at TEXT    NOT NULL,
    updated_at TEXT    NOT NULL,
    UNIQUE (owner_id, name_key)
);

CREATE TABLE link_tags (
    link_id    INTEGER NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    tag_id     INTEGER NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    created_at TEXT    NOT NULL,
    PRIMARY KEY (link_id, tag_id)
) WITHOUT ROWID;

CREATE INDEX idx_link_tags_tag ON link_tags (tag_id, link_id);

CREATE TABLE link_shares (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    link_id        INTEGER NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    target_type    TEXT    NOT NULL CHECK (target_type IN ('all', 'role', 'user')),
    target_role    TEXT    CHECK (target_role IN ('admin', 'developer', 'customer')),
    target_user_id INTEGER REFERENCES users (id) ON DELETE CASCADE,
    created_at     TEXT    NOT NULL,
    CHECK (
        (target_type = 'all'  AND target_role IS NULL     AND target_user_id IS NULL) OR
        (target_type = 'role' AND target_role IS NOT NULL AND target_user_id IS NULL) OR
        (target_type = 'user' AND target_role IS NULL     AND target_user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX ux_link_shares_target
    ON link_shares (link_id, target_type, coalesce(target_role, ''), coalesce(target_user_id, 0));
CREATE INDEX idx_link_shares_role ON link_shares (target_type, target_role);
CREATE INDEX idx_link_shares_user ON link_shares (target_user_id) WHERE target_user_id IS NOT NULL;

CREATE TABLE link_user_state (
    user_id        INTEGER NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    link_id        INTEGER NOT NULL REFERENCES links (id) ON DELETE CASCADE,
    pinned         INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1)),
    use_count      INTEGER NOT NULL DEFAULT 0 CHECK (use_count >= 0),
    last_used_at   TEXT,
    baseline_at    TEXT    NOT NULL,
    archived_at    TEXT,
    archive_reason TEXT    CHECK (archive_reason IN ('auto', 'manual')),
    updated_at     TEXT    NOT NULL,
    PRIMARY KEY (user_id, link_id),
    CHECK ((archived_at IS NULL) = (archive_reason IS NULL)),
    CHECK (NOT (pinned = 1 AND archived_at IS NOT NULL))
) WITHOUT ROWID;

CREATE INDEX idx_link_user_state_link ON link_user_state (link_id);
