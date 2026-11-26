-- name: NextTaskNumber :one
SELECT CAST(COALESCE(MAX(number), 0) + 1 AS INTEGER) FROM tasks WHERE project_id = ?;

-- name: CreateTask :one
INSERT INTO tasks (id, project_id, number, title, description, created_by_user_id)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetTask :one
SELECT * FROM tasks WHERE id = ?;

-- name: ListTasksByProject :many
SELECT t.id, t.number, t.title, t.description, t.status, t.created_at, t.updated_at,
       (SELECT COUNT(*) FROM reports r WHERE r.task_id = t.id) AS report_count,
       (SELECT COUNT(*) FROM reports r WHERE r.task_id = t.id AND r.status = 'done') AS done_count,
       (SELECT COUNT(*) FROM comments c WHERE c.task_id = t.id) AS comment_count
FROM tasks t
WHERE t.project_id = ?
ORDER BY t.number DESC;

-- name: UpdateTask :one
UPDATE tasks SET
    title       = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    updated_at  = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: SetTaskStatus :exec
UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: CreateReport :one
INSERT INTO reports (id, task_id, group_id, reporter_member_id, message_id, message_text, assigned_by_member_id, tag_message_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetReport :one
SELECT * FROM reports WHERE id = ?;

-- name: ListReportsByTask :many
SELECT r.id, r.task_id, r.group_id, r.reporter_member_id, r.message_id, r.message_text,
       r.assigned_by_member_id, r.status, r.closed_by_member_id, r.closed_by_user_id,
       r.closed_via, r.closed_at, r.created_at,
       g.name AS group_name,
       COALESCE(rm.push_name, rm.phone, rm.wa_id, '') AS reporter_name,
       COALESCE(am.push_name, am.phone, am.wa_id, '') AS assigned_by_name,
       COALESCE(cm.push_name, cm.phone, cm.wa_id, '') AS closed_by_member_name,
       COALESCE(cu.username, cu.email, '') AS closed_by_user_name
FROM reports r
LEFT JOIN wa_groups g ON g.id = r.group_id
LEFT JOIN wa_group_members rm ON rm.id = r.reporter_member_id
LEFT JOIN wa_group_members am ON am.id = r.assigned_by_member_id
LEFT JOIN wa_group_members cm ON cm.id = r.closed_by_member_id
LEFT JOIN users cu ON cu.id = r.closed_by_user_id
WHERE r.task_id = ?
ORDER BY r.created_at, r.id;

-- name: SetReportStatus :one
UPDATE reports SET
    status              = sqlc.arg('status'),
    closed_by_member_id = sqlc.narg('closed_by_member_id'),
    closed_by_user_id   = sqlc.narg('closed_by_user_id'),
    closed_via          = sqlc.narg('closed_via'),
    closed_at           = sqlc.narg('closed_at'),
    updated_at          = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: CountReports :one
SELECT COUNT(*) FROM reports WHERE task_id = ?;

-- name: CountOpenReports :one
SELECT COUNT(*) FROM reports WHERE task_id = ? AND status = 'todo';

-- name: CreateComment :one
INSERT INTO comments (id, task_id, report_id, author_user_id, body)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListCommentsByTask :many
SELECT c.id, c.task_id, c.report_id, c.author_user_id, c.body, c.created_at,
       COALESCE(u.username, u.email, '') AS author_name
FROM comments c
LEFT JOIN users u ON u.id = c.author_user_id
WHERE c.task_id = ?
ORDER BY c.created_at, c.id;

-- name: ListOpenTasks :many
SELECT id, number, title, description FROM tasks
WHERE project_id = ? AND status = 'todo'
ORDER BY updated_at DESC, number DESC
LIMIT ?;

-- name: GetReportByMessage :one
SELECT r.* FROM reports r
JOIN tasks t ON t.id = r.task_id
WHERE t.project_id = sqlc.arg('project_id') AND r.group_id = sqlc.arg('group_id')
  AND (r.message_id = sqlc.arg('message_id') OR r.tag_message_id = sqlc.arg('message_id')
       OR EXISTS (SELECT 1 FROM report_duplicates d WHERE d.report_id = r.id
                  AND (d.message_id = sqlc.arg('message_id') OR d.tag_message_id = sqlc.arg('message_id'))))
LIMIT 1;

-- name: ListOpenReportsByReporter :many
SELECT id, message_text FROM reports
WHERE task_id = ? AND reporter_member_id = ? AND status = 'todo'
ORDER BY created_at DESC
LIMIT ?;

-- name: GetReportInProject :one
SELECT r.* FROM reports r JOIN tasks t ON t.id = r.task_id
WHERE r.id = ? AND t.project_id = ?;

-- name: CreateDuplicate :one
INSERT INTO report_duplicates (id, report_id, message_id, tag_message_id, message_text)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListDuplicatesByTask :many
SELECT d.* FROM report_duplicates d
JOIN reports r ON r.id = d.report_id
WHERE r.task_id = ?
ORDER BY d.created_at, d.id;

-- name: CreateMedia :exec
INSERT INTO report_media (id, report_id, message_id, file, mime, size) VALUES (?, ?, ?, ?, ?, ?);

-- name: ListMediaByTask :many
SELECT m.id, m.report_id, m.message_id, m.mime, m.size, m.created_at
FROM report_media m
JOIN reports r ON r.id = m.report_id
WHERE r.task_id = ?
ORDER BY m.created_at, m.id;

-- name: GetMediaFile :one
SELECT m.file, m.mime, t.project_id
FROM report_media m
JOIN reports r ON r.id = m.report_id
JOIN tasks t ON t.id = r.task_id
WHERE m.id = ?;
