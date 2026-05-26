package streaming

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

func testSigner(t *testing.T) *Signer {
	t.Helper()
	s := NewSigner("test-secret", time.Hour)
	return s
}

func TestSigner_RoundTrip(t *testing.T) {
	s := testSigner(t)
	sig, exp := s.IssueNow("video-1")
	if sig == "" || exp == 0 {
		t.Fatalf("issued empty signature/exp: %q %d", sig, exp)
	}
	if err := s.Verify("video-1", sig, exp); err != nil {
		t.Fatalf("expected verify to pass, got %v", err)
	}
}

func TestSigner_RejectsWrongVideoID(t *testing.T) {
	s := testSigner(t)
	sig, exp := s.IssueNow("video-1")
	if err := s.Verify("video-2", sig, exp); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid, got %v", err)
	}
}

func TestSigner_RejectsTamperedSignature(t *testing.T) {
	s := testSigner(t)
	sig, exp := s.IssueNow("video-1")

	// flip the first hex digit
	mutated := []byte(sig)
	if mutated[0] == '0' {
		mutated[0] = '1'
	} else {
		mutated[0] = '0'
	}
	if err := s.Verify("video-1", string(mutated), exp); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid, got %v", err)
	}
}

func TestSigner_RejectsExpired(t *testing.T) {
	s := NewSigner("test-secret", time.Hour)
	pastExpires := time.Now().Add(-time.Minute).Unix()
	sig := s.computeHex("video-1", pastExpires)
	if err := s.Verify("video-1", sig, pastExpires); !errors.Is(err, ErrSignatureExpired) {
		t.Fatalf("expected ErrSignatureExpired, got %v", err)
	}
}

func TestSigner_DifferentSecretFails(t *testing.T) {
	a := NewSigner("secret-a", time.Hour)
	b := NewSigner("secret-b", time.Hour)
	sig, exp := a.IssueNow("video-1")
	if err := b.Verify("video-1", sig, exp); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid across secrets, got %v", err)
	}
}

func TestSigner_VerifyParams(t *testing.T) {
	s := testSigner(t)
	sig, exp := s.IssueNow("video-x")

	if err := s.VerifyParams("video-x", sig, strconv.FormatInt(exp, 10)); err != nil {
		t.Fatalf("VerifyParams: %v", err)
	}

	if err := s.VerifyParams("video-x", "", ""); !errors.Is(err, ErrSignatureRequired) {
		t.Fatalf("expected ErrSignatureRequired, got %v", err)
	}

	if err := s.VerifyParams("video-x", sig, "not-a-number"); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid for bad exp, got %v", err)
	}
}

func TestSigner_ClaimsRoundTripAndNonce(t *testing.T) {
	s := testSigner(t)
	sig, claims := s.IssueClaimsNow(SignatureScope, "video-1", "site-1", "Example.COM")
	if claims.Nonce == "" || claims.SiteID != "site-1" || claims.Domain != "example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if err := s.VerifyClaims(claims, sig); err != nil {
		t.Fatalf("VerifyClaims: %v", err)
	}
	claims.SiteID = "site-2"
	if err := s.VerifyClaims(claims, sig); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected ErrSignatureInvalid after site tamper, got %v", err)
	}
}

func TestSignedRequestDomain(t *testing.T) {
	if err := checkSignedRequestDomain("example.com", "https://example.com", ""); err != nil {
		t.Fatalf("expected exact signed domain pass: %v", err)
	}
	if err := checkSignedRequestDomain("example.com", "https://cdn.example.com", ""); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected subdomain forbidden, got %v", err)
	}
}
