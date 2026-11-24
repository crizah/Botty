-- name: ListProjectMembers :many
SELECT u.id AS user_id, COALESCE(u.username, '') AS username, u.email, pm.role, pm.created_at
FROM project_members pm
JOIN users u ON u.id = pm.user_id
WHERE pm.project_id = ?
ORDER BY CASE pm.role WHEN 'owner' THEN 0 WHEN 'editor' THEN 1 ELSE 2 END, pm.created_at;

-- name: GetProjectMember :one
SELECT * FROM project_members WHERE project_id = ? AND user_id = ?;

-- name: SetProjectMemberRole :execrows
UPDATE project_members SET role = ? WHERE project_id = ? AND user_id = ?;

-- name: RemoveProjectMember :execrows
DELETE FROM project_members WHERE project_id = ? AND user_id = ?;

-- name: CreateInvite :one
INSERT INTO project_invites (id, project_id, email, role, invited_by, expires_at)
VALUES (?, ?, ?, ?, ?, datetime('now', '+48 hours'))
RETURNING *;

-- name: ListPendingInvites :many
SELECT i.id, i.email, i.role, i.expires_at, i.created_at, COALESCE(u.username, u.email, '') AS invited_by_name
FROM project_invites i
JOIN users u ON u.id = i.invited_by
WHERE i.project_id = ? AND i.accepted_at IS NULL AND i.revoked_at IS NULL
ORDER BY i.created_at DESC;

-- name: GetInvite :one
SELECT * FROM project_invites WHERE id = ?;

-- name: GetOpenInvite :one
SELECT * FROM project_invites
WHERE id = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP;

-- name: RevokePendingInvitesForEmail :exec
UPDATE project_invites SET revoked_at = CURRENT_TIMESTAMP
WHERE project_id = ? AND email = ? AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: RevokeInvite :execrows
UPDATE project_invites SET revoked_at = CURRENT_TIMESTAMP
WHERE id = ? AND project_id = ? AND accepted_at IS NULL AND revoked_at IS NULL;

-- name: RefreshInvite :one
UPDATE project_invites SET expires_at = datetime('now', '+48 hours')
WHERE id = ? AND project_id = ? AND accepted_at IS NULL AND revoked_at IS NULL
RETURNING *;

-- name: AcceptInvite :execrows
UPDATE project_invites SET accepted_at = CURRENT_TIMESTAMP
WHERE id = ? AND accepted_at IS NULL AND revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP;
