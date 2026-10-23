// Package api assembles the Connect services and HTTP middleware into the
// server's http.Handler. It is the "database + config to http.Handler" seam
// that later tickets extend with more services.
package api

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/alecbakholdin/mybadcoach/server/internal/auth"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1/authv1connect"
)

// Config is the assembly input: the open database plus deployment settings.
type Config struct {
	// DB is the open database; it must have migrations applied.
	DB *sql.DB
	// Now supplies the current time; nil means time.Now.
	Now func() time.Time
	// AdditionalMutatingProcedures extends the CSRF-gated procedure set for
	// later services.
	AdditionalMutatingProcedures []string
	// AdditionalAuthenticatedProcedures extends the session-gated procedure set
	// for later services.
	AdditionalAuthenticatedProcedures []string
	// SecureCookies marks the session cookie Secure. Set it in production.
	SecureCookies bool
}

// NewHandler mounts every Connect service on a mux and wraps it in the CSRF
// and session middleware, outermost first.
func NewHandler(cfg Config) (http.Handler, error) {
	if cfg.DB == nil {
		return nil, errors.New("api: database is required")
	}

	mux := http.NewServeMux()

	authService := auth.NewService(auth.Config{
		DB:     cfg.DB,
		Secure: cfg.SecureCookies,
		Now:    cfg.Now,
	})
	authPath, authHandler := authv1connect.NewAuthServiceHandler(authService)
	mux.Handle(authPath, authHandler)

	middleware := auth.NewMiddleware(auth.MiddlewareConfig{
		DB:                                cfg.DB,
		Now:                               cfg.Now,
		AdditionalMutatingProcedures:      cfg.AdditionalMutatingProcedures,
		AdditionalAuthenticatedProcedures: cfg.AdditionalAuthenticatedProcedures,
	})
	return middleware.CSRF(middleware.Session(mux)), nil
}
