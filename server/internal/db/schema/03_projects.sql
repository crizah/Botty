CREATE TABLE projects (
    id          TEXT NOT NULL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    listen_mode TEXT NOT NULL DEFAULT 'bot_only' CHECK (listen_mode IN ('bot_only', 'all_texts')),
    created_by  TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- RBAC: owner (everything), editor (read + write tasks), viewer (read only)
CREATE TABLE project_members (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role       TEXT NOT NULL CHECK (role IN ('owner', 'editor', 'viewer')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, user_id)
);
CREATE INDEX idx_project_members_user ON project_members(user_id);

-- A group belongs to at most one project (group_id is UNIQUE) so that an
-- incoming message always maps to exactly one project.
CREATE TABLE project_groups (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    group_id   TEXT NOT NULL UNIQUE REFERENCES wa_groups(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, group_id)
);

-- who is allowed to tag the bot, per project AND group
CREATE TABLE tag_allowlist (
    project_id TEXT NOT NULL,
    group_id   TEXT NOT NULL,
    member_id  TEXT NOT NULL REFERENCES wa_group_members(id) ON DELETE CASCADE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, group_id, member_id),
    FOREIGN KEY (project_id, group_id) REFERENCES project_groups(project_id, group_id) ON DELETE CASCADE
);
