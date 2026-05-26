// Package auth owns admin authentication and authorization.
//
// Phase 2 (current):
//   - AdminUser domain model + role/status constants
//   - bcrypt password utilities (HashPassword / ComparePassword)
//   - AdminUserRepository (Create, FindByEmail, ExistsByEmail)
//
// Phase 3 (planned):
//   - HTTP handlers (login, logout, me)
//   - Auth service (login, lockout, token issuance)
//   - DTOs / request validation
//   - JWT / session middleware for protected routes
package auth
