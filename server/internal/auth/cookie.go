package auth

import (
	"net/http"
	"time"
)

// sessionCookie builds the session cookie for a freshly minted token.
func sessionCookie(token sessionToken, secure bool) *http.Cookie {
	// Secure is true in production and false only when serving plain-HTTP
	// development, so gosec's literal-true check does not apply here.
	//nolint:gosec // Secure is configurable by deployment.
	return &http.Cookie{
		Name:     CookieName,
		Value:    token.raw,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  token.expiry,
		MaxAge:   int(SessionLifetime / time.Second),
	}
}

// clearedSessionCookie builds the cookie that deletes the session cookie
// client-side (Max-Age=0).
func clearedSessionCookie(secure bool) *http.Cookie {
	//nolint:gosec // Secure is configurable by deployment.
	return &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
	}
}

// tokenFromRequest returns the raw session token from the request cookie.
func tokenFromRequest(request *http.Request) (string, bool) {
	cookie, err := request.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

// tokenFromHeaders returns the raw session token from a Connect request's
// headers.
func tokenFromHeaders(header http.Header) (string, bool) {
	request := http.Request{Header: header}
	return tokenFromRequest(&request)
}
