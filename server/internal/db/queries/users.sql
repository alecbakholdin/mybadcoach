-- name: GetUserByEmail :one
SELECT id, email, name, created_at
FROM users
WHERE email = ?;
