-- name: CreateEvent :one
INSERT INTO events (id, season_id, name, slug, start_date, end_date, location, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetEventBySlug :one
SELECT *
FROM events
WHERE season_id = ? AND slug = ?;

-- name: GetEventByID :one
SELECT *
FROM events
WHERE id = ?;

-- name: UpdateEventName :exec
UPDATE events
SET name = ?
WHERE id = ?;

-- name: UpdateEventStartDate :exec
UPDATE events
SET start_date = ?
WHERE id = ?;

-- name: UpdateEventEndDate :exec
UPDATE events
SET end_date = ?
WHERE id = ?;

-- name: UpdateEventLocation :exec
UPDATE events
SET location = ?
WHERE id = ?;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = ?;

-- name: EventSlugExists :one
SELECT COUNT(*)
FROM events
WHERE season_id = ? AND slug = ?;

-- name: ListEvents :many
SELECT *
FROM events
WHERE season_id = ?
ORDER BY created_at, id
LIMIT ? OFFSET ?;

-- name: CountEvents :one
SELECT COUNT(*)
FROM events
WHERE season_id = ?;
