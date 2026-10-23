package db_test

import (
	"testing"

	"github.com/alecbakholdin/mybadcoach/server/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func TestSeedDoesNothingWithoutDevFlag(t *testing.T) {
	t.Setenv(db.SeedEnvVar, "")
	sqlDB := openTestDB(t)

	if err := db.Seed(t.Context(), sqlDB); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	var users int
	if err := sqlDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&users); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 0 {
		t.Fatalf("users = %d, want 0 without %s=1", users, db.SeedEnvVar)
	}
}

func TestSeedCreatesAdminIdempotently(t *testing.T) {
	t.Setenv(db.SeedEnvVar, "1")
	sqlDB := openTestDB(t)
	ctx := t.Context()

	for run := 1; run <= 2; run++ {
		if err := db.Seed(ctx, sqlDB); err != nil {
			t.Fatalf("Seed run %d: %v", run, err)
		}
	}

	var (
		userID string
		email  string
		name   string
	)
	err := sqlDB.QueryRow(
		"SELECT id, email, name FROM users WHERE email = ?",
		db.DevAdminEmail,
	).Scan(&userID, &email, &name)
	if err != nil {
		t.Fatalf("read seeded user: %v", err)
	}
	if name == "" {
		t.Fatal("seeded user name must not be empty")
	}

	var users int
	err = sqlDB.QueryRow("SELECT COUNT(*) FROM users").Scan(&users)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 1 {
		t.Fatalf("users = %d, want 1 after two seed runs", users)
	}

	var identities int
	err = sqlDB.QueryRow(
		`SELECT COUNT(*) FROM auth_identities
		 WHERE user_id = ? AND provider = ? AND provider_subject = ? AND email = ?`,
		userID,
		db.PasswordProvider,
		db.DevAdminEmail,
		db.DevAdminEmail,
	).Scan(&identities)
	if err != nil {
		t.Fatalf("count auth_identities: %v", err)
	}
	if identities != 1 {
		t.Fatalf("password identities = %d, want 1", identities)
	}

	var hash string
	err = sqlDB.QueryRow(
		"SELECT password_hash FROM password_credentials WHERE user_id = ?",
		userID,
	).Scan(&hash)
	if err != nil {
		t.Fatalf("read password credential: %v", err)
	}
	if compareErr := bcrypt.CompareHashAndPassword([]byte(hash), []byte(db.DevAdminPassword)); compareErr != nil {
		t.Fatalf("stored hash does not verify against %q: %v", db.DevAdminPassword, compareErr)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("read bcrypt cost: %v", err)
	}
	if cost != 12 {
		t.Fatalf("bcrypt cost = %d, want 12", cost)
	}
}
