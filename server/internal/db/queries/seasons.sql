-- name: CreateSeason :one
INSERT INTO seasons (id, team_id, name, slug, start_date, end_date, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetSeasonBySlug :one
SELECT *
FROM seasons
WHERE team_id = ? AND slug = ?;

-- name: GetSeasonByID :one
SELECT *
FROM seasons
WHERE id = ?;

-- name: UpdateSeasonName :exec
UPDATE seasons
SET name = ?
WHERE id = ?;

-- name: UpdateSeasonStartDate :exec
UPDATE seasons
SET start_date = ?
WHERE id = ?;

-- name: UpdateSeasonEndDate :exec
UPDATE seasons
SET end_date = ?
WHERE id = ?;

-- name: DeleteSeason :exec
DELETE FROM seasons
WHERE id = ?;

-- name: SeasonSlugExists :one
SELECT COUNT(*)
FROM seasons
WHERE team_id = ? AND slug = ?;

-- name: ListSeasons :many
SELECT *
FROM seasons
WHERE team_id = ?
ORDER BY created_at, id
LIMIT ? OFFSET ?;

-- name: CountSeasons :one
SELECT COUNT(*)
FROM seasons
WHERE team_id = ?;
