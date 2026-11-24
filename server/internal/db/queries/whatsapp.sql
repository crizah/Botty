-- name: CreateGroup :one
INSERT INTO wa_groups (id, jid, name, owner_user_id)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetGroup :one
SELECT * FROM wa_groups WHERE id = ?;

-- name: ListGroupsForOwner :many
SELECT g.id, g.jid, g.name, g.created_at, pg.project_id,
       (SELECT COUNT(*) FROM wa_group_members m WHERE m.group_id = g.id) AS member_count
FROM wa_groups g
LEFT JOIN project_groups pg ON pg.group_id = g.id
WHERE g.owner_user_id = ?
ORDER BY g.created_at DESC;

-- name: UpsertMember :one
INSERT INTO wa_group_members (id, group_id, wa_id, phone, push_name)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT (group_id, wa_id) DO UPDATE SET
    push_name = COALESCE(excluded.push_name, wa_group_members.push_name),
    phone     = COALESCE(excluded.phone, wa_group_members.phone)
RETURNING *;

-- name: GetMember :one
SELECT * FROM wa_group_members WHERE id = ?;

-- name: ListMembersByGroup :many
SELECT * FROM wa_group_members WHERE group_id = ? ORDER BY push_name, wa_id;

-- name: SetMemberPhone :exec
UPDATE wa_group_members SET phone = ? WHERE id = ?;

-- name: ListMembersWithoutPhone :many
SELECT * FROM wa_group_members WHERE phone IS NULL LIMIT ?;
