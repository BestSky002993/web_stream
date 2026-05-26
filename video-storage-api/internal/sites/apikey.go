package sites

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// APIKeyPrefix prefixes every issued key so leaked keys are
// identifiable in logs and screenshots.
const APIKeyPrefix = "vsk_"

// apiKeyRandomBytes is the entropy in the key body. 32 bytes = 256
// bits — well beyond brute-force range.
const apiKeyRandomBytes = 32

// GenerateAPIKey returns a freshly minted plain key (to hand to the
// admin) and its SHA-256 hex hash (to store in the DB).
//
// The plain key looks like `vsk_<64 hex chars>`. Plain text is never
// persisted.
func GenerateAPIKey() (plain, hash string, err error) {
	buf := make([]byte, apiKeyRandomBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("sites: read random: %w", err)
	}
	plain = APIKeyPrefix + hex.EncodeToString(buf)
	hash = HashAPIKey(plain)
	return plain, hash, nil
}

// HashAPIKey returns the lowercase hex SHA-256 digest of `plain`.
// The same input always produces the same hash, so verification works
// by hashing the incoming key and looking up the row by hash.
//
// Pre-image attacks on a 256-bit random key are not a practical concern,
// so SHA-256 (no pepper) is sufficient and keeps the hot path fast.
func HashAPIKey(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// LooksLikeAPIKey is a cheap shape check before hitting the DB.
// Rejects empty input and anything missing the prefix.
func LooksLikeAPIKey(s string) bool {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, APIKeyPrefix) {
		return false
	}
	body := s[len(APIKeyPrefix):]
	if len(body) != apiKeyRandomBytes*2 {
		return false
	}
	for _, r := range body {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}
