package sites

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey_Shape(t *testing.T) {
	plain, hash, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if !strings.HasPrefix(plain, APIKeyPrefix) {
		t.Errorf("expected %q prefix, got %q", APIKeyPrefix, plain)
	}
	body := strings.TrimPrefix(plain, APIKeyPrefix)
	if len(body) != apiKeyRandomBytes*2 {
		t.Errorf("expected %d-char body, got %d", apiKeyRandomBytes*2, len(body))
	}
	if len(hash) != 64 { // sha256 hex
		t.Errorf("expected 64-char hex hash, got %d", len(hash))
	}
	if plain == hash {
		t.Error("plain and hash must differ")
	}
}

func TestGenerateAPIKey_Unique(t *testing.T) {
	seenKey := make(map[string]struct{}, 100)
	seenHash := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		k, h, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey: %v", err)
		}
		if _, dup := seenKey[k]; dup {
			t.Fatalf("duplicate plain key after %d iterations", i)
		}
		if _, dup := seenHash[h]; dup {
			t.Fatalf("duplicate hash after %d iterations", i)
		}
		seenKey[k] = struct{}{}
		seenHash[h] = struct{}{}
	}
}

func TestHashAPIKey_Deterministic(t *testing.T) {
	plain := "vsk_deadbeef"
	a := HashAPIKey(plain)
	b := HashAPIKey(plain)
	if a != b {
		t.Errorf("expected deterministic hash, got %q vs %q", a, b)
	}
}

func TestHashAPIKey_DiffersForDifferentInput(t *testing.T) {
	if HashAPIKey("a") == HashAPIKey("b") {
		t.Error("expected different inputs to hash differently")
	}
}

func TestHashAPIKey_RoundTrip(t *testing.T) {
	plain, hash, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey: %v", err)
	}
	if HashAPIKey(plain) != hash {
		t.Error("HashAPIKey of returned plain must equal returned hash")
	}
}

func TestLooksLikeAPIKey(t *testing.T) {
	ok, _, _ := GenerateAPIKey()
	if !LooksLikeAPIKey(ok) {
		t.Errorf("expected %q to look like an api key", ok)
	}
	bad := []string{
		"",
		"hello",
		"vsk_",
		"vsk_short",
		"VSK_" + strings.Repeat("a", 64), // wrong prefix case
		"vsk_" + strings.Repeat("z", 64), // non-hex
		"vsk_" + strings.Repeat("a", 63), // too short
		"vsk_" + strings.Repeat("a", 65), // too long
	}
	for _, b := range bad {
		if LooksLikeAPIKey(b) {
			t.Errorf("expected %q to be rejected", b)
		}
	}
}
