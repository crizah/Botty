-- Groups the bot has been added to.
CREATE TABLE wa_groups (
    id            TEXT NOT NULL PRIMARY KEY,
    jid           TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    -- the dashboard user this group is attributed to.
    -- TODO: change - the ingest worker needs a way to attribute a group to a user (claim code?)
    owner_user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_wa_groups_owner ON wa_groups(owner_user_id);

-- People inside a group. wa_id is whatever WhatsApp gives us (JID or LID);
-- phone is filled in by the identity-resolution pipeline when it can.
CREATE TABLE wa_group_members (
    id         TEXT NOT NULL PRIMARY KEY,
    group_id   TEXT NOT NULL REFERENCES wa_groups(id) ON DELETE CASCADE,
    wa_id      TEXT NOT NULL,
    phone      TEXT,
    push_name  TEXT,
    -- set when the member leaves the group; the row stays so past reports/allowlist history still resolve
    left_at    DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (group_id, wa_id)
);

-- One-time codes a dashboard user generates, then sends in a group as "@botty link <code>".
-- A group only exists in wa_groups once a claim succeeds, so ownership is always explicit.
CREATE TABLE group_claims (
    id         TEXT NOT NULL PRIMARY KEY,
    code       TEXT NOT NULL UNIQUE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    used_at    DATETIME,
    group_id   TEXT REFERENCES wa_groups(id) ON DELETE SET NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_group_claims_user ON group_claims(user_id);
