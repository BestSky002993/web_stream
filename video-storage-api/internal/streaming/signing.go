package streaming

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

// Signature scopes namespace the signed message so a signature minted
// for one endpoint can't be replayed against another.
const (
	SignatureScope    = "stream" // MP4 progressive streaming (Phase 6)
	SignatureScopeHLS = "hls"    // HLS manifest + segments (Phase 13)
)

// Signer issues and verifies HMAC-SHA256 stream signatures.
//
// Canonical signed message: "stream:<videoID>:<expires>".
// `expires` is a unix timestamp (seconds since epoch).
type Signer struct {
	secret []byte
	expIn  time.Duration
	now    func() time.Time
}

type SignatureClaims struct {
	Scope   string
	VideoID string
	SiteID  string
	Domain  string
	Nonce   string
	Expires int64
}

// NewSigner constructs a Signer with the given secret + default expiration.
func NewSigner(secret string, expiresIn time.Duration) *Signer {
	return &Signer{
		secret: []byte(secret),
		expIn:  expiresIn,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// IssueNow returns a signature valid until now+expiresIn, plus the
// expiration timestamp itself for inclusion in the URL. Uses the
// default "stream" scope.
func (s *Signer) IssueNow(videoID string) (signature string, expires int64) {
	return s.IssueNowScoped(SignatureScope, videoID)
}

// IssueNowScoped is the scope-aware variant. Use SignatureScopeHLS for
// HLS endpoints; signatures minted under one scope never validate
// under another.
func (s *Signer) IssueNowScoped(scope, videoID string) (signature string, expires int64) {
	expires = s.now().Add(s.expIn).Unix()
	signature = s.computeHexScoped(scope, videoID, expires)
	return signature, expires
}

func (s *Signer) IssueClaimsNow(scope, videoID, siteID, domain string) (signature string, claims SignatureClaims) {
	claims = SignatureClaims{
		Scope:   scope,
		VideoID: videoID,
		SiteID:  strings.TrimSpace(siteID),
		Domain:  strings.ToLower(strings.TrimSpace(domain)),
		Nonce:   randomNonce(),
		Expires: s.now().Add(s.expIn).Unix(),
	}
	signature = s.computeHexClaims(claims)
	return signature, claims
}

// Expiration returns the configured default expiration.
func (s *Signer) Expiration() time.Duration {
	return s.expIn
}

// Verify checks the signature under the default "stream" scope.
func (s *Signer) Verify(videoID, signature string, expires int64) error {
	return s.VerifyScoped(SignatureScope, videoID, signature, expires)
}

// VerifyScoped checks the signature against the given scope. Returns
// ErrSignatureExpired for stale tokens and ErrSignatureInvalid for
// anything else.
func (s *Signer) VerifyScoped(scope, videoID, signature string, expires int64) error {
	if videoID == "" || signature == "" || expires <= 0 {
		return ErrSignatureInvalid
	}
	if s.now().Unix() > expires {
		return ErrSignatureExpired
	}
	want := s.computeHexScoped(scope, videoID, expires)
	// hex digests have a fixed length; hmac.Equal is constant-time and
	// safe to call on differing lengths (returns false).
	if !hmac.Equal([]byte(want), []byte(signature)) {
		return ErrSignatureInvalid
	}
	return nil
}

// VerifyParams parses raw URL params and delegates to Verify (default scope).
func (s *Signer) VerifyParams(videoID, signature, expiresRaw string) error {
	return s.VerifyParamsScoped(SignatureScope, videoID, signature, expiresRaw)
}

// VerifyParamsScoped is the scope-aware variant.
func (s *Signer) VerifyParamsScoped(scope, videoID, signature, expiresRaw string) error {
	if expiresRaw == "" || signature == "" {
		return ErrSignatureRequired
	}
	expires, err := strconv.ParseInt(expiresRaw, 10, 64)
	if err != nil {
		return ErrSignatureInvalid
	}
	return s.VerifyScoped(scope, videoID, signature, expires)
}

func (s *Signer) VerifyClaims(c SignatureClaims, signature string) error {
	if c.Scope == "" || c.VideoID == "" || c.SiteID == "" || c.Domain == "" || c.Nonce == "" || signature == "" || c.Expires <= 0 {
		return ErrSignatureInvalid
	}
	if s.now().Unix() > c.Expires {
		return ErrSignatureExpired
	}
	want := s.computeHexClaims(c)
	if !hmac.Equal([]byte(want), []byte(signature)) {
		return ErrSignatureInvalid
	}
	return nil
}

func (s *Signer) computeHex(videoID string, expires int64) string {
	return s.computeHexScoped(SignatureScope, videoID, expires)
}

func (s *Signer) computeHexScoped(scope, videoID string, expires int64) string {
	msg := scope + ":" + videoID + ":" + strconv.FormatInt(expires, 10)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s *Signer) computeHexClaims(c SignatureClaims) string {
	msg := strings.Join([]string{
		c.Scope,
		c.VideoID,
		c.SiteID,
		strings.ToLower(c.Domain),
		c.Nonce,
		strconv.FormatInt(c.Expires, 10),
	}, ":")
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(msg))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomNonce() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b[:])
}
