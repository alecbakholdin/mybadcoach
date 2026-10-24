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
	"github.com/alecbakholdin/mybadcoach/server/internal/structure"
	"github.com/alecbakholdin/mybadcoach/server/web"

	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1/authv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/events/v1/eventsv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/matches/v1/matchesv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/players/v1/playersv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/seasons/v1/seasonsv1connect"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/teams/v1/teamsv1connect"
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

	teamPath, teamHandler := teamsv1connect.NewTeamServiceHandler(
		structure.NewTeamService(cfg.DB, cfg.Now),
	)
	mux.Handle(teamPath, teamHandler)

	seasonPath, seasonHandler := seasonsv1connect.NewSeasonServiceHandler(
		structure.NewSeasonService(cfg.DB, cfg.Now),
	)
	mux.Handle(seasonPath, seasonHandler)

	eventPath, eventHandler := eventsv1connect.NewEventServiceHandler(
		structure.NewEventService(cfg.DB, cfg.Now),
	)
	mux.Handle(eventPath, eventHandler)

	playerPath, playerHandler := playersv1connect.NewPlayerServiceHandler(
		structure.NewPlayerService(cfg.DB, cfg.Now),
	)
	mux.Handle(playerPath, playerHandler)

	matchPath, matchHandler := matchesv1connect.NewMatchServiceHandler(
		structure.NewMatchService(cfg.DB, cfg.Now),
	)
	mux.Handle(matchPath, matchHandler)

	// Serve the embedded SvelteKit SPA for every path the Connect services
	// don't claim: "/" is the least-specific pattern, so RPC routes still win.
	mux.Handle("/", web.Handler())

	middleware := auth.NewMiddleware(auth.MiddlewareConfig{
		DB:                                cfg.DB,
		Now:                               cfg.Now,
		AdditionalMutatingProcedures:      extend(cfg.AdditionalMutatingProcedures, structure.MutatingProcedures()),
		AdditionalAuthenticatedProcedures: extend(cfg.AdditionalAuthenticatedProcedures, structure.AuthenticatedProcedures()),
	})
	return middleware.CSRF(middleware.Session(mux)), nil
}

// extend returns the concatenation of the extra and built-in procedure sets
// in a fresh slice, so the caller's slices are never mutated.
func extend(extra, builtIn []string) []string {
	combined := make([]string, 0, len(extra)+len(builtIn))
	combined = append(combined, extra...)
	combined = append(combined, builtIn...)
	return combined
}
