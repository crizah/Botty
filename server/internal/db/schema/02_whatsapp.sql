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
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (group_id, wa_id)
);
