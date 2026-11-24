-- name: CreateProject :one
INSERT INTO projects (id, name, description, created_by)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: AddProjectMember :exec
INSERT INTO project_members (project_id, user_id, role) VALUES (?, ?, ?);

-- name: GetProjectRole :one
SELECT role FROM project_members WHERE project_id = ? AND user_id = ?;

-- name: ListProjectsForUser :many
SELECT p.id, p.name, p.description, p.listen_mode, p.created_at, pm.role,
       (SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id) AS task_count,
       (SELECT COUNT(*) FROM tasks t WHERE t.project_id = p.id AND t.status = 'done') AS done_count,
       (SELECT COUNT(*) FROM project_groups g WHERE g.project_id = p.id) AS group_count
FROM projects p
JOIN project_members pm ON pm.project_id = p.id
WHERE pm.user_id = ?
ORDER BY p.created_at DESC;

-- name: GetProject :one
SELECT * FROM projects WHERE id = ?;

-- name: UpdateProject :one
UPDATE projects SET
    name        = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    listen_mode = COALESCE(sqlc.narg('listen_mode'), listen_mode),
    updated_at  = CURRENT_TIMESTAMP
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: AttachGroup :exec
INSERT INTO project_groups (project_id, group_id) VALUES (?, ?);

-- name: DetachGroup :exec
DELETE FROM project_groups WHERE project_id = ? AND group_id = ?;

-- name: ListProjectGroups :many
SELECT g.id, g.jid, g.name,
       (SELECT COUNT(*) FROM wa_group_members m WHERE m.group_id = g.id) AS member_count,
       (SELECT COUNT(*) FROM tag_allowlist a WHERE a.project_id = pg.project_id AND a.group_id = g.id) AS allowlist_count
FROM project_groups pg
JOIN wa_groups g ON g.id = pg.group_id
WHERE pg.project_id = ?
ORDER BY g.name;

-- name: IsGroupInProject :one
SELECT COUNT(*) FROM project_groups WHERE project_id = ? AND group_id = ?;

-- name: AddAllowlist :exec
INSERT OR IGNORE INTO tag_allowlist (project_id, group_id, member_id) VALUES (?, ?, ?);

-- name: RemoveAllowlist :exec
DELETE FROM tag_allowlist WHERE project_id = ? AND group_id = ? AND member_id = ?;

-- name: ListAllowlist :many
SELECT m.id, m.wa_id, m.phone, m.push_name
FROM tag_allowlist a
JOIN wa_group_members m ON m.id = a.member_id
WHERE a.project_id = ? AND a.group_id = ?
ORDER BY m.push_name, m.wa_id;
