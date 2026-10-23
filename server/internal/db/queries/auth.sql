-- name: CreateUser :exec
INSERT INTO users (id, email, name, created_at)
VALUES (?, ?, ?, ?);

-- name: CreateAuthIdentity :exec
INSERT INTO auth_identities (id, user_id, provider, provider_subject, email, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: CreatePasswordCredential :exec
INSERT INTO password_credentials (user_id, password_hash, updated_at)
VALUES (?, ?, ?);

-- name: GetUserWithPasswordHash :one
SELECT u.id, u.email, u.name, p.password_hash
FROM users u
JOIN password_credentials p ON p.user_id = u.id
WHERE u.email = ?;

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at, created_at)
VALUES (?, ?, ?, ?);

-- name: GetSessionByTokenHash :one
SELECT s.expires_at, u.id, u.email, u.name
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = ?;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM sessions
WHERE token_hash = ?;
