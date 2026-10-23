-- Gameplay event log. Event ids are client-generated so offline replays are
-- idempotent; the payload column holds protojson for the event's oneof member
-- and type holds the short enum name (see internal/gameplay).

-- name: GetMatchEventByID :one
SELECT *
FROM match_events
WHERE id = ?;

-- name: CreateMatchEvent :exec
INSERT INTO match_events (
    id, match_id, point_number, effective_ts, type, payload, deleted, created_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateMatchEventDeleted :exec
UPDATE match_events
SET deleted = ?
WHERE id = ?;

-- name: ListMatchEvents :many
SELECT *
FROM match_events
WHERE match_id = ?
ORDER BY effective_ts, id
LIMIT ? OFFSET ?;

-- name: ListMatchEventsByPoint :many
SELECT *
FROM match_events
WHERE match_id = ? AND point_number = ?
ORDER BY effective_ts, id
LIMIT ? OFFSET ?;

-- name: CountMatchEvents :one
SELECT COUNT(*)
FROM match_events
WHERE match_id = ?;

-- name: CountMatchEventsByPoint :one
SELECT COUNT(*)
FROM match_events
WHERE match_id = ? AND point_number = ?;

-- name: ListAllMatchEvents :many
SELECT *
FROM match_events
WHERE match_id = ?
ORDER BY effective_ts, id;

-- name: ListTeamMatchScores :many
SELECT m.id AS match_id,
       (
           SELECT COUNT(*)
           FROM match_events e
           WHERE e.match_id = m.id AND e.type = 'SCORE' AND e.deleted = 0
       ) AS goals
FROM matches m
JOIN events ev ON ev.id = m.event_id
JOIN seasons s ON s.id = ev.season_id
WHERE m.our_team_id = ?
  AND (sqlc.narg('season_id') IS NULL OR s.id = sqlc.narg('season_id'))
ORDER BY m.id;
