// Command seed-admin creates the first admin user, idempotently.
//
// Reads ADMIN_EMAIL, ADMIN_PASSWORD and ADMIN_DISPLAY_NAME from the environment
// (plus the usual DB_* variables already used by the API). If a non-deleted
// admin already exists with that email, it exits cleanly without creating a
// duplicate unless --reset-password or ADMIN_RESET_PASSWORD=true is set. The
// password is hashed with bcrypt before insertion/update — plain passwords are
// never persisted.
//
// Typical production usage from the repo root:
//
//	docker compose -f docker-compose.prod.yml run --rm seed-admin
//	docker compose -f docker-compose.prod.yml run --rm seed-admin --reset-password
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/video-storage-api/internal/auth"
	"github.com/video-storage-api/internal/config"
	"github.com/video-storage-api/internal/database"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("seed-admin: %v", err)
	}
}

func run() error {
	resetPassword := flag.Bool("reset-password", false, "reset the password for the existing ADMIN_EMAIL user")
	flag.Parse()

	dbConfig, err := config.LoadDatabase()
	if err != nil {
		return fmt.Errorf("load database config: %w", err)
	}

	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	password := os.Getenv("ADMIN_PASSWORD")
	displayName := strings.TrimSpace(os.Getenv("ADMIN_DISPLAY_NAME"))

	if email == "" || password == "" || displayName == "" {
		return errors.New("ADMIN_EMAIL, ADMIN_PASSWORD and ADMIN_DISPLAY_NAME must all be set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := database.New(ctx, dbConfig)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer db.Close()

	repo := auth.NewAdminUserRepository(db.Pool)

	exists, err := repo.ExistsByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("check existing admin: %w", err)
	}
	if exists {
		if *resetPassword || shouldResetPassword() {
			hash, err := auth.HashPassword(password)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}

			const query = `
                UPDATE admin_users
                SET password_hash = $2,
                    display_name = $3,
                    status = 'active',
                    failed_login_attempts = 0,
                    locked_until = NULL,
                    updated_at = NOW()
                WHERE LOWER(email) = LOWER($1)
                  AND deleted_at IS NULL
            `
			tag, err := db.Pool.Exec(ctx, query, email, hash, displayName)
			if err != nil {
				return fmt.Errorf("reset admin password: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("reset admin password: no rows affected for %q", email)
			}

			log.Printf("reset password for admin user email=%s", email)
			return nil
		}

		log.Printf("admin user %q already exists — nothing to do", email)
		return nil
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	created, err := repo.Create(ctx, auth.NewAdminUserInput{
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Role:         auth.RoleSuperAdmin,
		Status:       auth.StatusActive,
	})
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	log.Printf("created admin user id=%s email=%s role=%s", created.ID, created.Email, created.Role)
	return nil
}

func shouldResetPassword() bool {
	value := strings.TrimSpace(os.Getenv("ADMIN_RESET_PASSWORD"))
	return value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes")
}
