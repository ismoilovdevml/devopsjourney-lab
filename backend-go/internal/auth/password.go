package auth

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	bcryptCost = 12

	// MinPasswordLength is the minimum password length in characters.
	MinPasswordLength = 8
	// MaxPasswordBytes is bcrypt's hard input limit; longer inputs would be silently truncated.
	MaxPasswordBytes = 72
)

var (
	ErrPasswordTooShort   = fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	ErrPasswordTooLong    = fmt.Errorf("password must be at most %d bytes", MaxPasswordBytes)
	ErrPasswordWeak       = errors.New("password is too weak")
	ErrPasswordIsUsername = errors.New("password must not equal the username")
	ErrInvalidUsername    = errors.New("username must be 3-32 characters: lowercase letters, digits, '_' or '-'")
)

var usernameRe = regexp.MustCompile(`^[a-z0-9_-]{3,32}$`)

// commonPasswords is a short blocklist of passwords that pass the length rule but are trivially guessable.
var commonPasswords = map[string]struct{}{
	"password": {}, "password1": {}, "password123": {}, "passw0rd": {},
	"12345678": {}, "123456789": {}, "1234567890": {}, "87654321": {},
	"qwerty123": {}, "qwertyuiop": {}, "11111111": {}, "abc12345": {},
	"admin123": {}, "admin1234": {}, "administrator": {}, "iloveyou": {},
	"letmein1": {}, "welcome1": {}, "changeme": {}, "devopslab": {},
}

// HashPassword hashes a password with bcrypt (cost 12). Inputs over 72 bytes are rejected
// instead of being silently truncated by bcrypt.
func HashPassword(password string) (string, error) {
	if len(password) > MaxPasswordBytes {
		return "", ErrPasswordTooLong
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(b), err
}

// CheckPassword reports whether password matches the bcrypt hash.
func CheckPassword(password, hash string) bool {
	if len(password) > MaxPasswordBytes || hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ValidatePasswordStrength enforces the password policy: 8+ characters, at most 72 bytes,
// not a single repeated character, not equal to the username, not a well-known password.
func ValidatePasswordStrength(password, username string) error {
	if utf8.RuneCountInString(password) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	if len(password) > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	first, _ := utf8.DecodeRuneInString(password)
	if strings.Count(password, string(first)) == utf8.RuneCountInString(password) {
		return ErrPasswordWeak
	}
	if username != "" && strings.EqualFold(password, username) {
		return ErrPasswordIsUsername
	}
	if _, bad := commonPasswords[strings.ToLower(password)]; bad {
		return ErrPasswordWeak
	}
	return nil
}

// ValidateUsername enforces ^[a-z0-9_-]{3,32}$.
func ValidateUsername(username string) error {
	if !usernameRe.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}
