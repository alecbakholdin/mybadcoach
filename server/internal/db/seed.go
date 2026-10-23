package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"

	idgen "github.com/alecbakholdin/mybadcoach/server/internal/id"
)

const (
	// SeedEnvVar gates the development seed: Seed runs only when this
	// environment variable has the value "1". Leave it unset in normal builds.
	SeedEnvVar = "MBC_DEV_SEED"

	// DevAdminEmail is the email of the seeded development admin account.
	DevAdminEmail = "admin@mybadcoach.dev"

	// DevAdminPassword is the plaintext password of the seeded development
	// admin account. It is bcrypt-hashed before storage.
	DevAdminPassword = "password"

	// PasswordProvider is the auth_identities.provider value for password
	// authentication. Password identities use the account email as
	// provider_subject; ticket 03's Register must follow the same convention.
	PasswordProvider = "password"

	devAdminName = "Dev Admin"
	bcryptCost   = 12
)

// Seed creates the development admin account when SeedEnvVar is "1" and no
// account with DevAdminEmail exists yet. It is idempotent: calling it on every
// startup is safe.
//
// Seed checks only SeedEnvVar. Callers that can run in production must also
// refuse to seed there (cmd/server gates on MBC_ENV), so the well-known dev
// credentials can never be created in a production deployment.
//
// The seeded identity uses PasswordProvider and the account email as
// provider_subject. The password credential holds a bcrypt hash (cost 12) of
// DevAdminPassword.
func Seed(ctx context.Context, sqlDB *sql.DB) error {
	if os.Getenv(SeedEnvVar) != "1" {
		return nil
	}

	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	q := New(tx)

	_, err = q.GetUserByEmail(ctx, DevAdminEmail)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("check for seeded admin: %w", err)
	}

	userID, err := idgen.New()
	if err != nil {
		return err
	}
	identityID, err := idgen.New()
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(DevAdminPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash dev password: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if err := q.CreateUser(ctx, CreateUserParams{
		ID:        userID,
		Email:     DevAdminEmail,
		Name:      devAdminName,
		CreatedAt: now,
	}); err != nil {
		return fmt.Errorf("insert dev user: %w", err)
	}
	if err := q.CreateAuthIdentity(ctx, CreateAuthIdentityParams{
		ID:              identityID,
		UserID:          userID,
		Provider:        PasswordProvider,
		ProviderSubject: DevAdminEmail,
		Email:           DevAdminEmail,
		CreatedAt:       now,
	}); err != nil {
		return fmt.Errorf("insert dev identity: %w", err)
	}
	if err := q.CreatePasswordCredential(ctx, CreatePasswordCredentialParams{
		UserID:       userID,
		PasswordHash: string(hash),
		UpdatedAt:    now,
	}); err != nil {
		return fmt.Errorf("insert dev password credential: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}
