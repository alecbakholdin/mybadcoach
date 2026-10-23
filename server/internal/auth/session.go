// Package auth implements password authentication, cookie-backed sessions, and
// the session and CSRF middleware shared by the Connect services.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	// CookieName is the name of the opaque session cookie.
	CookieName = "mbc_session"

	// SessionLifetime is the absolute lifetime of a session. Sessions never
	// slide or renew: past expires_at the session is invalid.
	SessionLifetime = 30 * 24 * time.Hour

	tokenBytes = 32
	idBytes    = 16
)

// sessionToken is a freshly minted session token: the raw value travels in the
// cookie, while only its hash is persisted.
type sessionToken struct {
	expiry time.Time
	raw    string
	hash   string
}

// newSessionToken returns a 256-bit random token, its SHA-256 hash, and the
// absolute expiry.
func newSessionToken(now time.Time) (sessionToken, error) {
	raw := make([]byte, tokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return sessionToken{}, fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return sessionToken{
		raw:    token,
		hash:   hashToken(token),
		expiry: now.Add(SessionLifetime),
	}, nil
}

// hashToken returns the lowercase hex SHA-256 digest of a raw session token.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// newID returns a random 32-character hex id (16 random bytes), matching the
// ids minted by the database seed.
func newID() (string, error) {
	raw := make([]byte, idBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(raw), nil
}

// createSession mints a token and stores its hash against userID.
func createSession(ctx context.Context, db *sql.DB, userID string, now time.Time) (sessionToken, error) {
	token, err := newSessionToken(now)
	if err != nil {
		return sessionToken{}, err
	}
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at, created_at)
		 VALUES (?, ?, ?, ?)`,
		token.hash,
		userID,
		token.expiry.Unix(),
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return sessionToken{}, fmt.Errorf("insert session: %w", err)
	}
	return token, nil
}

// lookupSession resolves a raw cookie token to its session and user. Expired
// sessions are deleted and reported as not found.
func lookupSession(
	ctx context.Context,
	db *sql.DB,
	raw string,
	now time.Time,
) (Session, bool, error) {
	hash := hashToken(raw)
	var (
		expiresUnix int64
		user        User
	)
	err := db.QueryRowContext(
		ctx,
		`SELECT s.expires_at, u.id, u.email, u.name
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ?`,
		hash,
	).Scan(&expiresUnix, &user.ID, &user.Email, &user.Name)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Session{}, false, nil
	case err != nil:
		return Session{}, false, fmt.Errorf("look up session: %w", err)
	}

	expiresAt := time.Unix(expiresUnix, 0).UTC()
	if !expiresAt.After(now) {
		deleteSessionHash(ctx, db, hash)
		return Session{}, false, nil
	}
	return Session{TokenHash: hash, User: user}, true, nil
}

// deleteSession removes the session identified by a raw cookie token.
func deleteSession(ctx context.Context, db *sql.DB, raw string) error {
	if _, err := db.ExecContext(
		ctx,
		"DELETE FROM sessions WHERE token_hash = ?",
		hashToken(raw),
	); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// deleteSessionHash removes a session by its stored hash. Cleanup of an
// expired session is best effort: a failed delete leaves a row that is still
// ignored on the next lookup.
func deleteSessionHash(ctx context.Context, db *sql.DB, hash string) {
	_, _ = db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash = ?", hash)
}
