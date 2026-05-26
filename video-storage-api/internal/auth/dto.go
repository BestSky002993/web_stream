package auth

// LoginRequest is the JSON body accepted by POST /api/v1/admin/auth/login.
//
// Validation rules (enforced by Gin's binding tags via go-playground/validator):
//   - email must be present and look like an email
//   - password must be present and non-empty
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=1"`
}

// LoginResponse is what we return on a successful login.
type LoginResponse struct {
	Token     string             `json:"token"`
	ExpiresIn int64              `json:"expires_in"`
	Admin     AdminBriefResponse `json:"admin"`
}

// AdminBriefResponse is the subset of admin fields embedded in the login
// response. Does not include status/timestamps — see AdminResponse for
// the fuller projection used by /me.
type AdminBriefResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// AdminResponse is the projection returned by GET /api/v1/admin/me.
type AdminResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

// ErrorResponse is the canonical error envelope for the auth endpoints.
type ErrorResponse struct {
	Error string `json:"error"`
}

func newAdminBrief(u *AdminUser) AdminBriefResponse {
	return AdminBriefResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
	}
}

func newAdminResponse(u *AdminUser) AdminResponse {
	return AdminResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Role:        u.Role,
		Status:      u.Status,
	}
}
