package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	seedIDBytes  = 16
)

// Seed creates the development admin account when SeedEnvVar is "1" and no
// account with DevAdminEmail exists yet. It is idempotent: calling it on every
// startup is safe.
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

	var existingID string
	err = tx.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", DevAdminEmail).Scan(&existingID)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("check for seeded admin: %w", err)
	}

	userID, err := newID()
	if err != nil {
		return err
	}
	identityID, err := newID()
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(DevAdminPassword), bcryptCost)
	if err != nil {
		return fmt.Errorf("hash dev password: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO users (id, email, name, created_at) VALUES (?, ?, ?, ?)",
		userID,
		DevAdminEmail,
		devAdminName,
		now,
	); err != nil {
		return fmt.Errorf("insert dev user: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO auth_identities
		 (id, user_id, provider, provider_subject, email, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		identityID,
		userID,
		PasswordProvider,
		DevAdminEmail,
		DevAdminEmail,
		now,
	); err != nil {
		return fmt.Errorf("insert dev identity: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		"INSERT INTO password_credentials (user_id, password_hash, updated_at) VALUES (?, ?, ?)",
		userID,
		string(hash),
		now,
	); err != nil {
		return fmt.Errorf("insert dev password credential: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}

// newID returns a random 32-character hex id (16 random bytes).
func newID() (string, error) {
	var raw [seedIDBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(raw[:]), nil
}
