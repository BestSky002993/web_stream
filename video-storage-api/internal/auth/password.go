package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// DefaultBcryptCost is used when callers don't specify a cost.
// 12 is a reasonable production default (slower than the bcrypt default of 10).
const DefaultBcryptCost = 12

// MinPasswordLength is enforced before hashing so we never persist a
// trivially-guessable secret. Tune later if policy changes.
const MinPasswordLength = 8

var (
	ErrPasswordTooShort = errors.New("password is too short")
	ErrPasswordEmpty    = errors.New("password is empty")
	ErrPasswordMismatch = errors.New("password does not match hash")
)

// HashPassword returns a bcrypt hash of the given plain-text password.
// Uses DefaultBcryptCost.
func HashPassword(plain string) (string, error) {
	return HashPasswordWithCost(plain, DefaultBcryptCost)
}

// HashPasswordWithCost returns a bcrypt hash of the given plain-text password
// using the provided cost factor. Useful in tests to keep them fast.
func HashPasswordWithCost(plain string, cost int) (string, error) {
	if plain == "" {
		return "", ErrPasswordEmpty
	}
	if len(plain) < MinPasswordLength {
		return "", ErrPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(hash), nil
}

// ComparePassword compares a plain-text password against a stored bcrypt hash.
// Returns nil when the password matches, ErrPasswordMismatch when it doesn't,
// or a wrapped error for unexpected failures (e.g. malformed hash).
func ComparePassword(hashed, plain string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}
		return fmt.Errorf("bcrypt compare: %w", err)
	}
	return nil
}
