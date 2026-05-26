package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Use the minimum bcrypt cost in tests so they stay fast.
const testCost = bcrypt.MinCost

func TestHashPassword_ProducesVerifiableHash(t *testing.T) {
	hash, err := HashPasswordWithCost("correct horse battery staple", testCost)
	if err != nil {
		t.Fatalf("HashPasswordWithCost: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("expected bcrypt hash prefix, got %q", hash)
	}

	if err := ComparePassword(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("ComparePassword should succeed: %v", err)
	}
}

func TestHashPassword_DifferentHashesEachCall(t *testing.T) {
	a, err := HashPasswordWithCost("hunter2hunter2", testCost)
	if err != nil {
		t.Fatalf("first hash: %v", err)
	}
	b, err := HashPasswordWithCost("hunter2hunter2", testCost)
	if err != nil {
		t.Fatalf("second hash: %v", err)
	}
	if a == b {
		t.Fatal("expected distinct hashes due to salt")
	}
}

func TestHashPassword_RejectsEmpty(t *testing.T) {
	_, err := HashPasswordWithCost("", testCost)
	if !errors.Is(err, ErrPasswordEmpty) {
		t.Fatalf("expected ErrPasswordEmpty, got %v", err)
	}
}

func TestHashPassword_RejectsTooShort(t *testing.T) {
	_, err := HashPasswordWithCost("abc", testCost)
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestComparePassword_Mismatch(t *testing.T) {
	hash, err := HashPasswordWithCost("right-password", testCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	err = ComparePassword(hash, "wrong-password")
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch, got %v", err)
	}
}

func TestComparePassword_MalformedHash(t *testing.T) {
	err := ComparePassword("not-a-bcrypt-hash", "whatever")
	if err == nil {
		t.Fatal("expected error for malformed hash")
	}
	if errors.Is(err, ErrPasswordMismatch) {
		t.Fatal("malformed hash should not be classified as mismatch")
	}
}
