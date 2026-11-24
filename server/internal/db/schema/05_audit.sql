CREATE TABLE audit_logs (
    id            TEXT NOT NULL PRIMARY KEY,
    project_id    TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id       TEXT REFERENCES tasks(id) ON DELETE CASCADE,
    report_id     TEXT REFERENCES reports(id) ON DELETE SET NULL,
    actor_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    actor_kind    TEXT NOT NULL DEFAULT 'user' CHECK (actor_kind IN ('user', 'bot', 'system')),
    action        TEXT NOT NULL, -- task.created, report.status_changed, comment.added, ...
    old_value     TEXT NOT NULL DEFAULT '',
    new_value     TEXT NOT NULL DEFAULT '',
    detail        TEXT NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_audit_project ON audit_logs(project_id, created_at);
CREATE INDEX idx_audit_task ON audit_logs(task_id, created_at);
