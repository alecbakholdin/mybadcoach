package auth

import (
	"context"
)

// User is the authenticated account resolved from a session.
type User struct {
	ID    string
	Email string
	Name  string
}

// Session is a server-side session resolved from a request cookie.
type Session struct {
	TokenHash string
	User      User
}

type sessionContextKey struct{}

// WithSession returns ctx carrying session, for the session middleware to
// hand the resolved session to handlers.
func WithSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, sessionContextKey{}, session)
}

// SessionFromContext reports the session injected by the session middleware.
func SessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(sessionContextKey{}).(Session)
	return session, ok
}
