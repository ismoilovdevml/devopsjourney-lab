package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// Issuer is the "iss" claim stamped on (and required from) every token.
const Issuer = "devopsjourney-lab"

// MinSecretLength is the minimum HMAC secret length in bytes (256 bits for HS256).
const MinSecretLength = 32

const defaultExpiration = 24 * time.Hour

var (
	jwtSecret     []byte
	jwtExpiration = defaultExpiration
)

// ErrInvalidToken is returned for any token that fails validation. Callers should not
// expose the underlying reason to clients.
var ErrInvalidToken = errors.New("invalid token")

// InitJWT initializes JWT configuration. It rejects secrets shorter than MinSecretLength bytes.
func InitJWT(cfg *configs.JWTConfig) error {
	if cfg == nil {
		return errors.New("jwt config is nil")
	}
	if len(cfg.Secret) < MinSecretLength {
		return fmt.Errorf("JWT secret must be at least %d bytes, got %d", MinSecretLength, len(cfg.Secret))
	}
	jwtSecret = []byte(cfg.Secret)
	jwtExpiration = cfg.Expiration
	if jwtExpiration <= 0 {
		jwtExpiration = defaultExpiration
	}
	return nil
}

// TokenTTL returns the configured token lifetime.
func TokenTTL() time.Duration {
	return jwtExpiration
}

// Claims are the JWT claims issued by this service.
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken issues a signed HS256 token for the user.
func GenerateToken(user *models.User) (string, error) {
	if user == nil {
		return "", errors.New("user is nil")
	}
	if len(jwtSecret) < MinSecretLength {
		return "", errors.New("JWT not initialized")
	}
	now := time.Now()
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			ExpiresAt: jwt.NewNumericDate(now.Add(jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
}

// ValidateToken parses and validates a token: HS256 only, exp and iat required,
// issuer must match, no clock-skew leeway.
func ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" || len(jwtSecret) < MinSecretLength {
		return nil, ErrInvalidToken
	}
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(0),
	)
	claims := &Claims{}
	token, err := parser.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.IssuedAt == nil || claims.UserID == 0 {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
