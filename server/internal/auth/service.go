package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"golang.org/x/crypto/bcrypt"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	authv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1"
	"github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/auth/v1/authv1connect"
	errorsv1 "github.com/alecbakholdin/mybadcoach/server/internal/gen/mybadcoach/errors/v1"
	idgen "github.com/alecbakholdin/mybadcoach/server/internal/id"
)

const (
	// BcryptCost is the hashing cost for new password credentials. It matches
	// the development seed.
	BcryptCost = 12

	minPasswordRunes = 8
	maxPasswordBytes = 72

	// dummyBcryptHash is compared against when a login names an unknown email,
	// so the unknown-email path costs roughly the same as a wrong password. It
	// is a cost-12 hash of an arbitrary value that never matches a real login.
	dummyBcryptHash = "$2a$12$eOfKGz5uEfmxxi2V9pNvSe5jqq8DzyEivhGaS4COgbJNjpfAzkac6"
)

// Config configures an auth Service.
type Config struct {
	// DB is the open database; it must have migrations applied.
	DB *sql.DB
	// Now supplies the current time; nil means time.Now.
	Now func() time.Time
	// Secure marks the session cookie Secure. Set it in production.
	Secure bool
}

// Service implements authv1connect.AuthServiceHandler.
type Service struct {
	db     *sql.DB
	now    func() time.Time
	secure bool
}

var _ authv1connect.AuthServiceHandler = (*Service)(nil)

// NewService builds an auth Service.
func NewService(cfg Config) *Service {
	return &Service{
		db:     cfg.DB,
		secure: cfg.Secure,
		now:    nowOr(cfg.Now),
	}
}

// Register creates a password account, opens a session, and sets the session
// cookie.
func (s *Service) Register(
	ctx context.Context,
	req *connect.Request[authv1.RegisterRequest],
) (*connect.Response[authv1.RegisterResponse], error) {
	email := normalizeEmail(req.Msg.GetEmail())
	name := strings.TrimSpace(req.Msg.GetName())
	password := req.Msg.GetPassword()

	if violations := validateRegistration(email, name, password); len(violations) > 0 {
		return nil, newConnectError(
			connect.CodeInvalidArgument,
			CodeInvalidArgument,
			"invalid registration",
			violations,
		)
	}

	user, err := s.createUser(ctx, email, name, password)
	if err != nil {
		var connectErr *connect.Error
		if errors.As(err, &connectErr) {
			return nil, connectErr
		}
		return nil, internalError()
	}

	token, err := createSession(ctx, s.db, user.ID, s.now())
	if err != nil {
		return nil, internalError()
	}

	response := connect.NewResponse(&authv1.RegisterResponse{User: userMessage(user)})
	response.Header().Add("Set-Cookie", sessionCookie(token, s.secure).String())
	return response, nil
}

// createUser inserts the user, password identity, and password credential in
// one transaction. A duplicate email surfaces as CodeAlreadyExists.
func (s *Service) createUser(
	ctx context.Context,
	email string,
	name string,
	password string,
) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}
	userID, err := idgen.New()
	if err != nil {
		return User{}, err
	}
	identityID, err := idgen.New()
	if err != nil {
		return User{}, err
	}
	now := s.now().UTC().Format(time.RFC3339Nano)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin register: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO users (id, email, name, created_at) VALUES (?, ?, ?, ?)",
		userID,
		email,
		name,
		now,
	); err != nil {
		if isUniqueViolation(err) {
			return User{}, newConnectError(
				connect.CodeAlreadyExists,
				CodeEmailTaken,
				"email already registered",
				nil,
			)
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO auth_identities
		 (id, user_id, provider, provider_subject, email, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		identityID,
		userID,
		db.PasswordProvider,
		email,
		email,
		now,
	); err != nil {
		return User{}, fmt.Errorf("insert identity: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO password_credentials (user_id, password_hash, updated_at) VALUES (?, ?, ?)",
		userID,
		string(hash),
		now,
	); err != nil {
		return User{}, fmt.Errorf("insert password credential: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit register: %w", err)
	}
	return User{ID: userID, Email: email, Name: name}, nil
}

// Login verifies a password and opens a session. An unknown email and a wrong
// password produce the same error.
func (s *Service) Login(
	ctx context.Context,
	req *connect.Request[authv1.LoginRequest],
) (*connect.Response[authv1.LoginResponse], error) {
	email := normalizeEmail(req.Msg.GetEmail())
	password := req.Msg.GetPassword()

	var (
		user User
		hash string
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT u.id, u.email, u.name, p.password_hash
		 FROM users u
		 JOIN password_credentials p ON p.user_id = u.id
		 WHERE u.email = ?`,
		email,
	).Scan(&user.ID, &user.Email, &user.Name, &hash)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Burn a bcrypt comparison so an unknown email is not observably
		// faster than a wrong password.
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
		return nil, invalidCredentialsError()
	case err != nil:
		return nil, internalError()
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return nil, invalidCredentialsError()
	}

	token, err := createSession(ctx, s.db, user.ID, s.now())
	if err != nil {
		return nil, internalError()
	}

	response := connect.NewResponse(&authv1.LoginResponse{User: userMessage(user)})
	response.Header().Add("Set-Cookie", sessionCookie(token, s.secure).String())
	return response, nil
}

// Logout deletes the current session when there is one and always clears the
// cookie, so it is idempotent.
func (s *Service) Logout(
	ctx context.Context,
	req *connect.Request[authv1.LogoutRequest],
) (*connect.Response[authv1.LogoutResponse], error) {
	if raw, ok := tokenFromHeaders(req.Header()); ok {
		if err := deleteSession(ctx, s.db, raw); err != nil {
			return nil, internalError()
		}
	}

	response := connect.NewResponse(&authv1.LogoutResponse{})
	response.Header().Add("Set-Cookie", clearedSessionCookie(s.secure).String())
	return response, nil
}

// Me returns the user for the request's valid session. The session middleware
// rejects invalid sessions before this handler runs.
func (s *Service) Me(
	ctx context.Context,
	_ *connect.Request[authv1.MeRequest],
) (*connect.Response[authv1.MeResponse], error) {
	session, ok := SessionFromContext(ctx)
	if !ok {
		return nil, sessionExpiredError()
	}
	return connect.NewResponse(&authv1.MeResponse{User: userMessage(session.User)}), nil
}

func nowOr(now func() time.Time) func() time.Time {
	if now != nil {
		return now
	}
	return time.Now
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// validEmail accepts the basic address shape (local@domain) and rejects
// display-name forms like `Name <a@b>`.
func validEmail(email string) bool {
	if email == "" {
		return false
	}
	address, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return address.Address == email
}

func validateRegistration(
	email string,
	name string,
	password string,
) []*errorsv1.FieldViolation {
	var violations []*errorsv1.FieldViolation
	if !validEmail(email) {
		violations = append(violations, &errorsv1.FieldViolation{
			Field:       "email",
			Description: "must be a valid email address",
		})
	}
	if name == "" {
		violations = append(violations, &errorsv1.FieldViolation{
			Field:       "name",
			Description: "must not be empty",
		})
	}
	switch {
	case utf8.RuneCountInString(password) < minPasswordRunes:
		violations = append(violations, &errorsv1.FieldViolation{
			Field:       "password",
			Description: "must be at least 8 characters",
		})
	case len(password) > maxPasswordBytes:
		violations = append(violations, &errorsv1.FieldViolation{
			Field:       "password",
			Description: "must be at most 72 bytes",
		})
	}
	return violations
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func userMessage(user User) *authv1.User {
	return &authv1.User{
		Id:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}
}
