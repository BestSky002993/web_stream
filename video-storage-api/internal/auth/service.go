package auth

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

// Service implements the authentication business logic on top of the
// repository and token manager. It is the only place where lockout
// policy, password verification, and token issuance are coordinated.
type Service struct {
	repo        *AdminUserRepository
	tokens      *TokenManager
	maxAttempts int
	lockFor     time.Duration
	now         func() time.Time
}

func NewService(repo *AdminUserRepository, tokens *TokenManager, maxAttempts int, lockFor time.Duration) *Service {
	return &Service{
		repo:        repo,
		tokens:      tokens,
		maxAttempts: maxAttempts,
		lockFor:     lockFor,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

// LoginResult is everything a handler needs after a successful login.
type LoginResult struct {
	Admin     *AdminUser
	Token     string
	ExpiresIn time.Duration
}

// Login authenticates an admin by email + password.
//
// Outcomes (in priority order):
//   - account is currently within a lockout window → ErrAccountLocked
//   - account has status='disabled'                → ErrAccountDisabled
//   - email unknown OR password wrong              → ErrInvalidCredentials
//     (wrong password also increments the failed-attempt counter and
//     may flip the account into a locked state)
//   - everything OK                                → LoginResult with a fresh JWT
func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrAdminUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("lookup admin: %w", err)
	}

	now := s.now()

	// Within an active lockout window? Reject before touching the password,
	// so brute-forcers can't keep extending the counter.
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		return nil, ErrAccountLocked
	}

	if user.Status == StatusDisabled {
		return nil, ErrAccountDisabled
	}

	// Verify password.
	if cmpErr := ComparePassword(user.PasswordHash, password); cmpErr != nil {
		if errors.Is(cmpErr, ErrPasswordMismatch) {
			lockUntil := now.Add(s.lockFor)
			if _, recErr := s.repo.RecordFailedAttempt(ctx, user.ID, s.maxAttempts, lockUntil); recErr != nil {
				// Don't expose the DB failure to the client — we still
				// want to reply with the generic invalid-credentials error.
				log.Printf("auth: failed to record failed login attempt for %s: %v", user.ID, recErr)
			}
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("compare password: %w", cmpErr)
	}

	// Success path: reset login state.
	updated, err := s.repo.MarkLoginSuccess(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("mark login success: %w", err)
	}

	token, expiresIn, err := s.tokens.Issue(updated)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &LoginResult{
		Admin:     updated,
		Token:     token,
		ExpiresIn: expiresIn,
	}, nil
}

// Authenticate validates a raw bearer token and returns the fresh admin row
// it points at, after enforcing status and lockout rules. Used by middleware.
//
// Any token problem collapses to ErrInvalidToken. Status/lock failures
// surface as ErrAccountDisabled or ErrAccountLocked so handlers can shape
// the response message accordingly.
func (s *Service) Authenticate(ctx context.Context, token string) (*AdminUser, error) {
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.FindByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, ErrAdminUserNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("load admin: %w", err)
	}

	if user.Status == StatusDisabled {
		return nil, ErrAccountDisabled
	}
	if user.Status == StatusLocked {
		return nil, ErrAccountLocked
	}
	if user.LockedUntil != nil && user.LockedUntil.After(s.now()) {
		return nil, ErrAccountLocked
	}
	if user.Status != StatusActive {
		// Future-proof: any new non-active status defaults to deny.
		return nil, ErrAccountDisabled
	}

	return user, nil
}
