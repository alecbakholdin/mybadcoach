package api_test

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"github.com/alecbakholdin/mybadcoach/server/internal/api"
	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	authv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1/authv1connect"
	errorsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/errors/v1"
)

const (
	testEmail    = "player@example.com"
	testPassword = "correct-horse"
	testName     = "Test Player"
)

// harness wires the production handler assembly against a real temp-file
// SQLite database and a real HTTP server, per docs/testing.md's backend
// integration seam. It keeps two Connect clients over the same cookie jar:
// client sends the CSRF header, bareClient does not.
type harness struct {
	t          *testing.T
	db         *sql.DB
	httpClient *http.Client
	client     authv1connect.AuthServiceClient
	bareClient authv1connect.AuthServiceClient
	url        string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return newHarnessWithConfig(t, api.Config{})
}

func newHarnessWithConfig(t *testing.T, cfg api.Config) *harness {
	t.Helper()

	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	cfg.DB = sqlDB
	handler, err := api.NewHandler(cfg)
	if err != nil {
		t.Fatalf("assemble handler: %v", err)
	}

	server := httptest.NewServer(handler)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("create cookie jar: %v", err)
	}
	httpClient := &http.Client{Jar: jar}

	t.Cleanup(func() {
		server.Close()
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("close test database: %v", closeErr)
		}
	})

	return &harness{
		t:          t,
		db:         sqlDB,
		url:        server.URL,
		httpClient: httpClient,
		client: authv1connect.NewAuthServiceClient(
			httpClient,
			server.URL,
			connect.WithInterceptors(browserHeaders("")),
		),
		bareClient: authv1connect.NewAuthServiceClient(httpClient, server.URL),
	}
}

// browserHeaders adds the header the CSRF middleware requires on mutating
// RPCs, standing in for the browser client, and an Origin header when origin
// is non-empty.
func browserHeaders(origin string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("X-Requested-With", "XMLHttpRequest")
			if origin != "" {
				req.Header().Set("Origin", origin)
			}
			return next(ctx, req)
		}
	}
}

// clientWithHeaders builds a client over the harness cookie jar that sends the
// CSRF header and the given Origin (when non-empty).
func (h *harness) clientWithHeaders(origin string) authv1connect.AuthServiceClient {
	h.t.Helper()
	return authv1connect.NewAuthServiceClient(
		h.httpClient,
		h.url,
		connect.WithInterceptors(browserHeaders(origin)),
	)
}

func (h *harness) register(email, password, name string) (*connect.Response[authv1.RegisterResponse], error) {
	h.t.Helper()
	return h.client.Register(context.Background(), connect.NewRequest(&authv1.RegisterRequest{
		Email:    email,
		Password: password,
		Name:     name,
	}))
}

// assertConnectError requires err to be a Connect error with the given code
// and machine-readable ErrorDetail code.
func assertConnectError(t *testing.T, err error, code connect.Code, machineCode string) {
	t.Helper()

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("error = %v, want a *connect.Error", err)
	}
	if connectErr.Code() != code {
		t.Fatalf("connect code = %v, want %v", connectErr.Code(), code)
	}
	if got := errorMachineCode(t, connectErr); got != machineCode {
		t.Fatalf("machine code = %q, want %q", got, machineCode)
	}
}

func errorFieldViolations(t *testing.T, err error) []*errorsv1.FieldViolation {
	t.Helper()

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("error = %v, want a *connect.Error", err)
	}
	for _, detail := range connectErr.Details() {
		if detail.Type() != "mybadcoach.errors.v1.ErrorDetail" {
			continue
		}
		message, valueErr := detail.Value()
		if valueErr != nil {
			t.Fatalf("decode error detail: %v", valueErr)
		}
		decoded, ok := message.(*errorsv1.ErrorDetail)
		if !ok {
			t.Fatalf("error detail type = %T, want *errorsv1.ErrorDetail", message)
		}
		return decoded.GetFieldViolations()
	}
	return nil
}

func assertFieldViolation(t *testing.T, err error, field string) {
	t.Helper()

	violations := errorFieldViolations(t, err)
	for _, violation := range violations {
		if violation.GetField() == field {
			return
		}
	}
	t.Fatalf("field violations = %v, want a violation for %q", violations, field)
}

func errorMachineCode(t *testing.T, err *connect.Error) string {
	t.Helper()

	for _, detail := range err.Details() {
		if detail.Type() != "mybadcoach.errors.v1.ErrorDetail" {
			continue
		}
		message, valueErr := detail.Value()
		if valueErr != nil {
			t.Fatalf("decode error detail: %v", valueErr)
		}
		decoded, ok := message.(*errorsv1.ErrorDetail)
		if !ok {
			t.Fatalf("error detail type = %T, want *errorsv1.ErrorDetail", message)
		}
		return decoded.GetCode()
	}
	return ""
}

func TestRegisterMeLogoutMe(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	registered, err := h.register(testEmail, testPassword, testName)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	user := registered.Msg.GetUser()
	if user.GetId() == "" {
		t.Fatal("Register returned a user without an id")
	}
	if user.GetEmail() != testEmail {
		t.Fatalf("user email = %q, want %q", user.GetEmail(), testEmail)
	}
	if user.GetName() != testName {
		t.Fatalf("user name = %q, want %q", user.GetName(), testName)
	}
	if got := registered.Header().Get("Set-Cookie"); got == "" {
		t.Fatal("Register set no session cookie")
	}

	me, err := h.bareClient.Me(ctx, connect.NewRequest(&authv1.MeRequest{}))
	if err != nil {
		t.Fatalf("Me after Register: %v", err)
	}
	if me.Msg.GetUser().GetId() != user.GetId() {
		t.Fatalf("Me user id = %q, want %q", me.Msg.GetUser().GetId(), user.GetId())
	}

	loggedOut, err := h.client.Logout(ctx, connect.NewRequest(&authv1.LogoutRequest{}))
	if err != nil {
		t.Fatalf("Logout: %v", err)
	}
	logoutResponse := http.Response{Header: loggedOut.Header()}
	cleared := findCookie(t, logoutResponse.Cookies(), "mbc_session")
	if cleared.MaxAge >= 0 {
		t.Fatalf("Logout cookie MaxAge = %d, want a deletion (negative)", cleared.MaxAge)
	}
	if cleared.Value != "" {
		t.Fatalf("Logout cookie value = %q, want empty", cleared.Value)
	}

	if _, err := h.bareClient.Me(ctx, connect.NewRequest(&authv1.MeRequest{})); err == nil {
		t.Fatal("Me after Logout succeeded, want session_expired")
	} else {
		assertConnectError(t, err, connect.CodeUnauthenticated, "session_expired")
	}
}

func TestLoginSymmetry(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	if _, err := h.register(testEmail, testPassword, testName); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, unknownErr := h.client.Login(ctx, connect.NewRequest(&authv1.LoginRequest{
		Email:    "nobody@example.com",
		Password: testPassword,
	}))
	if unknownErr == nil {
		t.Fatal("Login with an unknown email succeeded, want invalid_credentials")
	}
	assertConnectError(t, unknownErr, connect.CodeUnauthenticated, "invalid_credentials")

	_, wrongErr := h.client.Login(ctx, connect.NewRequest(&authv1.LoginRequest{
		Email:    testEmail,
		Password: "not-the-password",
	}))
	if wrongErr == nil {
		t.Fatal("Login with a wrong password succeeded, want invalid_credentials")
	}
	assertConnectError(t, wrongErr, connect.CodeUnauthenticated, "invalid_credentials")

	var unknownConnectErr, wrongConnectErr *connect.Error
	if !errors.As(unknownErr, &unknownConnectErr) {
		t.Fatalf("unknown-email error = %v, want *connect.Error", unknownErr)
	}
	if !errors.As(wrongErr, &wrongConnectErr) {
		t.Fatalf("wrong-password error = %v, want *connect.Error", wrongErr)
	}
	if unknownConnectErr.Code() != wrongConnectErr.Code() ||
		unknownConnectErr.Message() != wrongConnectErr.Message() {
		t.Fatalf(
			"unknown-email error = (%v, %q), wrong-password error = (%v, %q); want identical",
			unknownConnectErr.Code(),
			unknownConnectErr.Message(),
			wrongConnectErr.Code(),
			wrongConnectErr.Message(),
		)
	}

	if _, err := h.client.Login(ctx, connect.NewRequest(&authv1.LoginRequest{
		Email:    testEmail,
		Password: testPassword,
	})); err != nil {
		t.Fatalf("Login with valid credentials: %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	cases := []struct {
		name      string
		request   *authv1.RegisterRequest
		wantField string
	}{
		{
			name: "bad email",
			request: &authv1.RegisterRequest{
				Email:    "not-an-email",
				Password: testPassword,
				Name:     testName,
			},
			wantField: "email",
		},
		{
			name: "short password",
			request: &authv1.RegisterRequest{
				Email:    testEmail,
				Password: "short",
				Name:     testName,
			},
			wantField: "password",
		},
		{
			name: "password over 72 bytes",
			request: &authv1.RegisterRequest{
				Email:    testEmail,
				Password: strings.Repeat("a", 73),
				Name:     testName,
			},
			wantField: "password",
		},
		{
			name: "empty name",
			request: &authv1.RegisterRequest{
				Email:    testEmail,
				Password: testPassword,
				Name:     "",
			},
			wantField: "name",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := h.client.Register(ctx, connect.NewRequest(testCase.request))
			if err == nil {
				t.Fatal("Register succeeded, want invalid_argument")
			}
			assertConnectError(t, err, connect.CodeInvalidArgument, "invalid_argument")
			assertFieldViolation(t, err, testCase.wantField)
		})
	}
}

func TestRegisterDuplicate(t *testing.T) {
	h := newHarness(t)

	if _, err := h.register(testEmail, testPassword, testName); err != nil {
		t.Fatalf("first Register: %v", err)
	}

	// Case-insensitive: emails are normalized to lowercase.
	if _, err := h.register(strings.ToUpper(testEmail), testPassword, "Other Name"); err == nil {
		t.Fatal("duplicate Register succeeded, want email_taken")
	} else {
		assertConnectError(t, err, connect.CodeAlreadyExists, "email_taken")
	}
}

func TestExpiredSessionRejected(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	registered, err := h.register(testEmail, testPassword, testName)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	past := time.Now().Add(-time.Minute).Unix()
	if _, err := h.db.Exec(
		"UPDATE sessions SET expires_at = ? WHERE user_id = ?",
		past,
		registered.Msg.GetUser().GetId(),
	); err != nil {
		t.Fatalf("expire session: %v", err)
	}

	if _, err := h.bareClient.Me(ctx, connect.NewRequest(&authv1.MeRequest{})); err == nil {
		t.Fatal("Me with an expired session succeeded, want session_expired")
	} else {
		assertConnectError(t, err, connect.CodeUnauthenticated, "session_expired")
	}
}

func TestSessionCookiePosture(t *testing.T) {
	cases := []struct {
		name   string
		secure bool
	}{
		{name: "development", secure: false},
		{name: "production", secure: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			h := newHarnessWithConfig(t, api.Config{SecureCookies: testCase.secure})
			registered, err := h.register(testEmail, testPassword, testName)
			if err != nil {
				t.Fatalf("Register: %v", err)
			}

			response := http.Response{Header: registered.Header()}
			cookie := findCookie(t, response.Cookies(), "mbc_session")
			if !cookie.HttpOnly {
				t.Fatal("session cookie is not HttpOnly")
			}
			if cookie.SameSite != http.SameSiteLaxMode {
				t.Fatalf("session cookie SameSite = %v, want Lax", cookie.SameSite)
			}
			if cookie.Path != "/" {
				t.Fatalf("session cookie Path = %q, want %q", cookie.Path, "/")
			}
			if cookie.Secure != testCase.secure {
				t.Fatalf("session cookie Secure = %v, want %v", cookie.Secure, testCase.secure)
			}
			if cookie.MaxAge != int((30 * 24 * time.Hour).Seconds()) {
				t.Fatalf("session cookie MaxAge = %d, want %d", cookie.MaxAge, 30*24*60*60)
			}
			if cookie.Value == "" {
				t.Fatal("session cookie has an empty value")
			}
		})
	}
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("no %q cookie in %v", name, cookies)
	return nil
}

func TestCSRFPosture(t *testing.T) {
	register := func(client authv1connect.AuthServiceClient) error {
		_, err := client.Register(context.Background(), connect.NewRequest(&authv1.RegisterRequest{
			Email:    testEmail,
			Password: testPassword,
			Name:     testName,
		}))
		return err
	}

	t.Run("mutating RPC without the header is rejected", func(t *testing.T) {
		h := newHarness(t)
		err := register(h.bareClient)
		assertConnectError(t, err, connect.CodePermissionDenied, "csrf_rejected")

		var users int
		if scanErr := h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&users); scanErr != nil {
			t.Fatalf("count users: %v", scanErr)
		}
		if users != 0 {
			t.Fatalf("users = %d, want 0 when CSRF rejects the request", users)
		}
	})

	t.Run("cross-origin request is rejected", func(t *testing.T) {
		h := newHarness(t)
		err := register(h.clientWithHeaders("https://evil.example"))
		assertConnectError(t, err, connect.CodePermissionDenied, "csrf_rejected")

		var users int
		if scanErr := h.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&users); scanErr != nil {
			t.Fatalf("count users: %v", scanErr)
		}
		if users != 0 {
			t.Fatalf("users = %d, want 0 when CSRF rejects the request", users)
		}
	})

	t.Run("same-origin request with the header succeeds", func(t *testing.T) {
		h := newHarness(t)
		if err := register(h.clientWithHeaders(h.url)); err != nil {
			t.Fatalf("Register with frontend headers: %v", err)
		}
	})

	t.Run("read RPC needs no CSRF header", func(t *testing.T) {
		h := newHarness(t)
		if _, err := h.register(testEmail, testPassword, testName); err != nil {
			t.Fatalf("Register: %v", err)
		}
		if _, err := h.bareClient.Me(
			context.Background(),
			connect.NewRequest(&authv1.MeRequest{}),
		); err != nil {
			t.Fatalf("Me without the CSRF header: %v", err)
		}
	})
}
