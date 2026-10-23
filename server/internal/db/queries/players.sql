-- name: CreatePlayer :one
INSERT INTO players (id, team_id, names, slug, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPlayerBySlug :one
SELECT *
FROM players
WHERE team_id = ? AND slug = ?;

-- name: GetPlayerByID :one
SELECT *
FROM players
WHERE id = ?;

-- name: UpdatePlayerNames :exec
UPDATE players
SET names = ?
WHERE id = ?;

-- name: DeletePlayer :exec
DELETE FROM players
WHERE id = ?;

-- name: PlayerSlugExists :one
SELECT COUNT(*)
FROM players
WHERE team_id = ? AND slug = ?;

-- name: ListPlayers :many
SELECT *
FROM players
WHERE team_id = ?
ORDER BY created_at, id
LIMIT ? OFFSET ?;

-- name: CountPlayers :one
SELECT COUNT(*)
FROM players
WHERE team_id = ?;
