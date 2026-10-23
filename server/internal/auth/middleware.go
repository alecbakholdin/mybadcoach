package auth

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1/authv1connect"
)

// CSRFHeader is the custom header browser clients send on mutating RPCs. Its
// presence proves the request came from script that could read the response,
// which a cross-site form post cannot do.
const CSRFHeader = "X-Requested-With"

// DefaultMutatingProcedures returns the RPC procedures gated by the CSRF
// check out of the box: the auth service's state-changing calls.
func DefaultMutatingProcedures() []string {
	return []string{
		authv1connect.AuthServiceRegisterProcedure,
		authv1connect.AuthServiceLoginProcedure,
		authv1connect.AuthServiceLogoutProcedure,
	}
}

// DefaultAuthenticatedProcedures returns the RPC procedures that require a
// valid session. Logout is intentionally absent: it is idempotent and clears
// the cookie even when the session has already expired.
func DefaultAuthenticatedProcedures() []string {
	return []string{authv1connect.AuthServiceMeProcedure}
}

// MiddlewareConfig configures the session and CSRF middleware. Later services
// extend the gates by adding their procedures to the Additional sets; the
// auth service's defaults always apply.
type MiddlewareConfig struct {
	// DB is the open database; it must have migrations applied.
	DB *sql.DB
	// Now supplies the current time; nil means time.Now.
	Now func() time.Time
	// AdditionalMutatingProcedures extends the CSRF-gated procedure set.
	AdditionalMutatingProcedures []string
	// AdditionalAuthenticatedProcedures extends the session-gated procedure set.
	AdditionalAuthenticatedProcedures []string
}

// Middleware parses session cookies, injects sessions into request contexts,
// rejects unauthenticated access to protected procedures, and enforces the
// CSRF posture on mutating procedures.
type Middleware struct {
	queries       *db.Queries
	now           func() time.Time
	mutating      map[string]struct{}
	authenticated map[string]struct{}
	errorWriter   *connect.ErrorWriter
}

// NewMiddleware builds the middleware. Mutating and authenticated procedure
// sets are matched against the request path, which is the Connect procedure
// name for RPC requests.
func NewMiddleware(cfg MiddlewareConfig) *Middleware {
	return &Middleware{
		queries:       db.New(cfg.DB),
		now:           nowOr(cfg.Now),
		mutating:      procedureSet(DefaultMutatingProcedures(), cfg.AdditionalMutatingProcedures),
		authenticated: procedureSet(DefaultAuthenticatedProcedures(), cfg.AdditionalAuthenticatedProcedures),
		errorWriter:   connect.NewErrorWriter(),
	}
}

// Session resolves the session cookie (when present) onto the request context
// and rejects protected procedures that have no valid session.
func (m *Middleware) Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if raw, ok := tokenFromRequest(request); ok {
			session, found, err := lookupSession(request.Context(), m.queries, raw, m.now())
			if err != nil {
				m.writeError(writer, request, internalError())
				return
			}
			if found {
				request = request.WithContext(WithSession(request.Context(), session))
			}
		}

		if _, protected := m.authenticated[request.URL.Path]; protected {
			if _, ok := SessionFromContext(request.Context()); !ok {
				m.writeError(writer, request, sessionExpiredError())
				return
			}
		}

		next.ServeHTTP(writer, request)
	})
}

// CSRF requires the CSRF header on mutating procedures and rejects a present,
// cross-origin Origin header. Requests without an Origin header (non-browser
// clients) are allowed through the origin check.
func (m *Middleware) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if _, mutating := m.mutating[request.URL.Path]; mutating {
			if request.Header.Get(CSRFHeader) == "" {
				m.writeError(writer, request, csrfRejectedError())
				return
			}
			if origin := request.Header.Get("Origin"); origin != "" && !sameOrigin(request, origin) {
				m.writeError(writer, request, csrfRejectedError())
				return
			}
		}
		next.ServeHTTP(writer, request)
	})
}

// sameOrigin reports whether origin names the request's own scheme and host.
func sameOrigin(request *http.Request, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	return strings.EqualFold(parsed.Scheme, scheme) &&
		strings.EqualFold(parsed.Host, request.Host)
}

func procedureSet(sets ...[]string) map[string]struct{} {
	result := make(map[string]struct{})
	for _, set := range sets {
		for _, procedure := range set {
			result[procedure] = struct{}{}
		}
	}
	return result
}

func (m *Middleware) writeError(writer http.ResponseWriter, request *http.Request, err error) {
	if writeErr := m.errorWriter.Write(writer, request, err); writeErr != nil {
		http.Error(writer, "internal server error", http.StatusInternalServerError)
	}
}
