package database

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/models"
	"github.com/rs/zerolog/log"
)

const generatedAdminPasswordLen = 24

// Unambiguous alphabet (no 0/O, 1/l/I) so the printed password is easy to copy by hand.
const passwordAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

// SeedAdminUser creates the first admin when SEED_ADMIN=true and no admin exists yet.
// ADMIN_PASSWORD is used only if it passes the password policy; otherwise a random password is
// generated and printed once to stdout. It never promotes an existing account and is idempotent.
func SeedAdminUser() error {
	if os.Getenv("SEED_ADMIN") != "true" {
		log.Info().Msg("Admin seeding disabled (SEED_ADMIN != true)")
		return nil
	}

	var admins int64
	if err := DB.Model(&models.User{}).Where("role = ?", "admin").Count(&admins).Error; err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if admins > 0 {
		log.Info().Int64("admins", admins).Msg("Admin user already exists, skipping seed")
		return nil
	}

	username := strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_USERNAME")))
	if username == "" {
		username = "admin"
	}
	if err := auth.ValidateUsername(username); err != nil {
		return fmt.Errorf("ADMIN_USERNAME: %w", err)
	}
	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	if email == "" {
		email = username + "@localhost"
	}

	// Never take over an existing (possibly attacker-registered) account.
	var taken int64
	if err := DB.Unscoped().Model(&models.User{}).
		Where("username = ? OR email = ?", username, email).Count(&taken).Error; err != nil {
		return fmt.Errorf("check admin username: %w", err)
	}
	if taken > 0 {
		return errors.New("ADMIN_USERNAME or ADMIN_EMAIL already belongs to a non-admin account; refusing to promote it — choose another username/email")
	}

	password, generated, err := resolveAdminPassword(os.Getenv("ADMIN_PASSWORD"), username)
	if err != nil {
		return err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash admin password: %w", err)
	}

	admin := models.User{
		Username: username,
		Email:    email,
		Password: hash,
		Role:     "admin",
		Active:   true,
	}
	if err := DB.Create(&admin).Error; err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	if generated {
		printGeneratedPassword(username, password)
	}
	log.Info().Str("username", username).Bool("generated_password", generated).Msg("Admin user created")
	return nil
}

// resolveAdminPassword returns the configured password if it satisfies the policy,
// otherwise a freshly generated one (generated=true).
func resolveAdminPassword(configured, username string) (password string, generated bool, err error) {
	if configured != "" {
		if strings.EqualFold(configured, "admin123") {
			log.Warn().Msg("ADMIN_PASSWORD is the insecure default 'admin123'; generating a random password instead")
		} else if verr := auth.ValidatePasswordStrength(configured, username); verr != nil {
			log.Warn().Err(verr).Msg("ADMIN_PASSWORD does not meet the password policy; generating a random password instead")
		} else {
			return configured, false, nil
		}
	}
	p, err := generatePassword(generatedAdminPasswordLen)
	if err != nil {
		return "", false, err
	}
	return p, true, nil
}

func generatePassword(n int) (string, error) {
	max := big.NewInt(int64(len(passwordAlphabet)))
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("generate password: %w", err)
		}
		b[i] = passwordAlphabet[idx.Int64()]
	}
	return string(b), nil
}

// printGeneratedPassword writes the one-time credentials to stdout (not the structured log)
// so they are visible once in the container logs and never stored anywhere else.
func printGeneratedPassword(username, password string) {
	line := strings.Repeat("=", 64)
	fmt.Fprintf(os.Stdout, "\n%s\n  INITIAL ADMIN ACCOUNT CREATED (shown only once)\n\n  username: %s\n  password: %s\n\n  Log in and change this password now (POST /api/me/password).\n%s\n\n",
		line, username, password, line)
}
