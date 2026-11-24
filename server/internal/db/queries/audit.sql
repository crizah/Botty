-- name: CreateAuditLog :exec
INSERT INTO audit_logs (id, project_id, task_id, report_id, actor_user_id, actor_kind, action, old_value, new_value, detail)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ListAuditByProject :many
SELECT a.id, a.task_id, a.report_id, a.actor_kind, a.action, a.old_value, a.new_value, a.detail, a.created_at,
       COALESCE(u.username, u.email, '') AS actor_name,
       t.number AS task_number, t.title AS task_title
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_user_id
LEFT JOIN tasks t ON t.id = a.task_id
WHERE a.project_id = ?
ORDER BY a.created_at DESC, a.id DESC
LIMIT ?;

-- name: ListAuditByTask :many
SELECT a.id, a.task_id, a.report_id, a.actor_kind, a.action, a.old_value, a.new_value, a.detail, a.created_at,
       COALESCE(u.username, u.email, '') AS actor_name
FROM audit_logs a
LEFT JOIN users u ON u.id = a.actor_user_id
WHERE a.task_id = ?
ORDER BY a.created_at DESC, a.id DESC;
