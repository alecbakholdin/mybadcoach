-- name: AddSeasonRosterEntry :exec
INSERT OR IGNORE INTO season_roster_entries (id, season_id, player_id, created_at)
VALUES (?, ?, ?, ?);

-- name: RemoveSeasonRosterEntry :exec
DELETE FROM season_roster_entries
WHERE season_id = ? AND player_id = ?;

-- name: ListSeasonRoster :many
SELECT p.*
FROM players p
JOIN season_roster_entries r ON r.player_id = p.id
WHERE r.season_id = ?
ORDER BY p.created_at, p.id
LIMIT ? OFFSET ?;

-- name: CountSeasonRoster :one
SELECT COUNT(*)
FROM season_roster_entries
WHERE season_id = ?;

-- name: ListSeasonRosterPlayersNotInTarget :many
SELECT p.*
FROM players p
JOIN season_roster_entries r ON r.player_id = p.id
WHERE r.season_id = ?
  AND p.id NOT IN (
      SELECT target.player_id
      FROM season_roster_entries target
      WHERE target.season_id = ?
  )
ORDER BY p.created_at, p.id;

-- name: AddEventRosterEntry :exec
INSERT OR IGNORE INTO event_roster_entries (id, event_id, player_id, created_at)
VALUES (?, ?, ?, ?);

-- name: RemoveEventRosterEntry :exec
DELETE FROM event_roster_entries
WHERE event_id = ? AND player_id = ?;

-- name: ListEventRoster :many
SELECT p.*
FROM players p
JOIN event_roster_entries r ON r.player_id = p.id
WHERE r.event_id = ?
ORDER BY p.created_at, p.id
LIMIT ? OFFSET ?;

-- name: CountEventRoster :one
SELECT COUNT(*)
FROM event_roster_entries
WHERE event_id = ?;
