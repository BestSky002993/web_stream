package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// adminContextKey is the gin.Context key under which the authenticated
// admin is stored by RequireAdmin. Unexported so callers must use
// AdminFromContext.
const adminContextKey = "auth.admin"

// RequireAdmin returns a Gin middleware that:
//
//   - extracts the bearer token from the Authorization header
//   - validates the signature + expiration via TokenManager
//   - re-loads the admin from the DB (so deleted/disabled/locked accounts
//     can't keep using a still-valid token)
//   - injects the live AdminUser into the request context
//
// All failure paths abort with HTTP 401 and a JSON body.
func RequireAdmin(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := extractBearer(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "missing or malformed authorization header"})
			return
		}

		admin, err := svc.Authenticate(c.Request.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, ErrAccountLocked):
				c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "account is temporarily locked"})
			case errors.Is(err, ErrAccountDisabled):
				c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "account is disabled"})
			case errors.Is(err, ErrInvalidToken):
				c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			default:
				c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
			}
			return
		}

		c.Set(adminContextKey, admin)
		c.Next()
	}
}

// AdminFromContext returns the admin stashed by RequireAdmin.
func AdminFromContext(c *gin.Context) (*AdminUser, bool) {
	v, ok := c.Get(adminContextKey)
	if !ok {
		return nil, false
	}
	u, ok := v.(*AdminUser)
	return u, ok
}

// extractBearer pulls the token out of an `Authorization: Bearer <token>`
// header. Whitespace and case in the scheme are tolerated; everything else
// is rejected.
func extractBearer(c *gin.Context) (string, bool) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", false
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}
