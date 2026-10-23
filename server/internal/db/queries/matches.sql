-- name: CreateMatch :one
INSERT INTO matches (
    id, event_id, our_team_id, opponent_name, slug, scheduled_at, point_cap, created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMatchBySlug :one
SELECT *
FROM matches
WHERE event_id = ? AND slug = ?;

-- name: GetMatchByID :one
SELECT *
FROM matches
WHERE id = ?;

-- name: UpdateMatchOpponentName :exec
UPDATE matches
SET opponent_name = ?
WHERE id = ?;

-- name: UpdateMatchScheduledAt :exec
UPDATE matches
SET scheduled_at = ?
WHERE id = ?;

-- name: UpdateMatchPointCap :exec
UPDATE matches
SET point_cap = ?
WHERE id = ?;

-- name: DeleteMatch :exec
DELETE FROM matches
WHERE id = ?;

-- name: MatchSlugExists :one
SELECT COUNT(*)
FROM matches
WHERE event_id = ? AND slug = ?;

-- name: ListMatches :many
SELECT *
FROM matches
WHERE event_id = ?
ORDER BY created_at, id
LIMIT ? OFFSET ?;

-- name: CountMatches :one
SELECT COUNT(*)
FROM matches
WHERE event_id = ?;
