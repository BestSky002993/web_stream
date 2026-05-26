package auth

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler exposes the auth HTTP endpoints. It is intentionally thin —
// all business logic lives in Service.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Login handles POST /api/v1/admin/auth/login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:     result.Token,
		ExpiresIn: int64(result.ExpiresIn.Seconds()),
		Admin:     newAdminBrief(result.Admin),
	})
}

// Me handles GET /api/v1/admin/me. The admin is injected into the request
// context by the RequireAdmin middleware.
func (h *Handler) Me(c *gin.Context) {
	admin, ok := AdminFromContext(c)
	if !ok {
		// Should never happen behind RequireAdmin, but be defensive.
		log.Printf("auth: /me invoked without admin in context")
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}
	c.JSON(http.StatusOK, newAdminResponse(admin))
}

// writeAuthError maps service-level errors to HTTP responses. Unknown errors
// are logged server-side and returned as a generic 500.
func writeAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid credentials"})
	case errors.Is(err, ErrAccountLocked):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "account is temporarily locked"})
	case errors.Is(err, ErrAccountDisabled):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "account is disabled"})
	case errors.Is(err, ErrInvalidToken):
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
	default:
		log.Printf("auth: internal error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
	}
}
