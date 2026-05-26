package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims is the payload signed into an access token.
type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

// TokenManager signs and verifies admin JWTs using HS256.
type TokenManager struct {
	secret    []byte
	expiresIn time.Duration
	now       func() time.Time
}

func NewTokenManager(secret string, expiresIn time.Duration) *TokenManager {
	return &TokenManager{
		secret:    []byte(secret),
		expiresIn: expiresIn,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

// ExpiresIn returns the configured token TTL.
func (m *TokenManager) ExpiresIn() time.Duration {
	return m.expiresIn
}

// Issue returns a signed JWT for the given admin and the token's TTL.
func (m *TokenManager) Issue(user *AdminUser) (string, time.Duration, error) {
	if user == nil {
		return "", 0, errors.New("nil admin user")
	}

	now := m.now()
	claims := Claims{
		Email: user.Email,
		Role:  user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiresIn)),
			Issuer:    "video-storage-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", 0, fmt.Errorf("sign token: %w", err)
	}
	return signed, m.expiresIn, nil
}

// Parse validates a JWT string and returns its claims. Any failure
// (malformed, expired, bad signature, wrong algorithm) is collapsed
// into ErrInvalidToken so handlers can return a uniform 401.
func (m *TokenManager) Parse(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrInvalidToken
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuedAt(),
	)

	claims := &Claims{}
	_, err := parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Subject == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
