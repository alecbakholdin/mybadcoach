-- name: CreateMembership :one
INSERT INTO team_memberships (id, user_id, team_id, role, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMembership :one
SELECT *
FROM team_memberships
WHERE team_id = ? AND user_id = ?;

-- name: GetMembershipWithUser :one
SELECT m.role AS role, u.id AS user_id, u.email AS email, u.name AS name
FROM team_memberships m
JOIN users u ON u.id = m.user_id
WHERE m.team_id = ? AND m.user_id = ?;

-- name: UpdateMembershipRole :exec
UPDATE team_memberships
SET role = ?
WHERE team_id = ? AND user_id = ?;

-- name: DeleteMembership :exec
DELETE FROM team_memberships
WHERE team_id = ? AND user_id = ?;

-- name: CountOwners :one
SELECT COUNT(*)
FROM team_memberships
WHERE team_id = ? AND role = 'owner';
