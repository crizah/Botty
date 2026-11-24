CREATE TABLE tasks (
    id                 TEXT NOT NULL PRIMARY KEY,
    project_id         TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    number             INTEGER NOT NULL, -- per-project sequence, shown as T-<number>
    title              TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    -- derived from reports (all done -> done, any todo -> todo)
    status             TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'done')),
    created_by_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (project_id, number)
);

-- one report = one person asking for the task, from one message in one group
CREATE TABLE reports (
    id                    TEXT NOT NULL PRIMARY KEY,
    task_id               TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    group_id              TEXT REFERENCES wa_groups(id) ON DELETE SET NULL,
    reporter_member_id    TEXT REFERENCES wa_group_members(id) ON DELETE SET NULL,
    message_id            TEXT NOT NULL DEFAULT '', -- WhatsApp message id
    message_text          TEXT NOT NULL DEFAULT '', -- snapshot of the message
    assigned_by_member_id TEXT REFERENCES wa_group_members(id) ON DELETE SET NULL, -- who tagged the bot
    status                TEXT NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'done')),
    closed_by_member_id   TEXT REFERENCES wa_group_members(id) ON DELETE SET NULL,
    closed_by_user_id     TEXT REFERENCES users(id) ON DELETE SET NULL,
    closed_via            TEXT CHECK (closed_via IN ('dashboard', 'tag', 'keyword')),
    closed_at             DATETIME,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_reports_task ON reports(task_id);

-- report_id NULL = comment on the task, otherwise a comment on that report
CREATE TABLE comments (
    id             TEXT NOT NULL PRIMARY KEY,
    task_id        TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    report_id      TEXT REFERENCES reports(id) ON DELETE CASCADE,
    author_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    body           TEXT NOT NULL,
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_comments_task ON comments(task_id);
