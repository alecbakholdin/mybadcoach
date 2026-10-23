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

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
)

const (
	// CookieName is the name of the opaque session cookie.
	CookieName = "mbc_session"

	// SessionLifetime is the absolute lifetime of a session. Sessions never
	// slide or renew: past expires_at the session is invalid.
	SessionLifetime = 30 * 24 * time.Hour

	tokenBytes = 32
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

// createSession mints a token and stores its hash against userID.
func createSession(ctx context.Context, q *db.Queries, userID string, now time.Time) (sessionToken, error) {
	token, err := newSessionToken(now)
	if err != nil {
		return sessionToken{}, err
	}
	if err := q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: token.hash,
		UserID:    userID,
		ExpiresAt: token.expiry.Unix(),
		CreatedAt: now.UTC().Format(time.RFC3339Nano),
	}); err != nil {
		return sessionToken{}, fmt.Errorf("insert session: %w", err)
	}
	return token, nil
}

// lookupSession resolves a raw cookie token to its session and user. Expired
// sessions are deleted and reported as not found.
func lookupSession(
	ctx context.Context,
	q *db.Queries,
	raw string,
	now time.Time,
) (Session, bool, error) {
	hash := hashToken(raw)
	row, err := q.GetSessionByTokenHash(ctx, hash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Session{}, false, nil
	case err != nil:
		return Session{}, false, fmt.Errorf("look up session: %w", err)
	}

	expiresAt := time.Unix(row.ExpiresAt, 0).UTC()
	if !expiresAt.After(now) {
		deleteSessionHash(ctx, q, hash)
		return Session{}, false, nil
	}
	user := User{ID: row.ID, Email: row.Email, Name: row.Name}
	return Session{TokenHash: hash, User: user}, true, nil
}

// deleteSession removes the session identified by a raw cookie token.
func deleteSession(ctx context.Context, q *db.Queries, raw string) error {
	if err := q.DeleteSessionByTokenHash(ctx, hashToken(raw)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// deleteSessionHash removes a session by its stored hash. Cleanup of an
// expired session is best effort: a failed delete leaves a row that is still
// ignored on the next lookup.
func deleteSessionHash(ctx context.Context, q *db.Queries, hash string) {
	_ = q.DeleteSessionByTokenHash(ctx, hash)
}
