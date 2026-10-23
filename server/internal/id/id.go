// Package id mints the opaque random identifiers stored in text id columns.
package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const idBytes = 16

// New returns a random 32-character lowercase hex id (16 random bytes).
func New() (string, error) {
	raw := make([]byte, idBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return hex.EncodeToString(raw), nil
}
