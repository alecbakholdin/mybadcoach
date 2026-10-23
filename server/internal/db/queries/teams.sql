-- name: CreateTeam :one
INSERT INTO teams (id, slug, name, created_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetTeamBySlug :one
SELECT *
FROM teams
WHERE slug = ?;

-- name: GetTeamByID :one
SELECT *
FROM teams
WHERE id = ?;

-- name: UpdateTeamName :exec
UPDATE teams
SET name = ?
WHERE id = ?;

-- name: DeleteTeam :exec
DELETE FROM teams
WHERE id = ?;

-- name: TeamSlugExists :one
SELECT COUNT(*)
FROM teams
WHERE slug = ?;

-- name: ListTeamsForUser :many
SELECT t.*, m.role AS role
FROM teams t
JOIN team_memberships m ON m.team_id = t.id
WHERE m.user_id = ?
ORDER BY t.name, t.id;
