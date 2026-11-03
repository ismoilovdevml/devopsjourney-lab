package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "0123456789abcdef0123456789abcdef-test"

func initTestJWT(t *testing.T) {
	t.Helper()
	if err := InitJWT(&configs.JWTConfig{Secret: testSecret, Expiration: time.Hour}); err != nil {
		t.Fatalf("InitJWT: %v", err)
	}
}

func signWith(t *testing.T, method jwt.SigningMethod, key interface{}, claims jwt.Claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, claims).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func validClaims() Claims {
	now := time.Now()
	return Claims{
		UserID: 7, Username: "alice", Role: "student",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
		},
	}
}

func TestInitJWTRejectsShortSecret(t *testing.T) {
	if err := InitJWT(&configs.JWTConfig{Secret: "short", Expiration: time.Hour}); err == nil {
		t.Fatal("expected error for short secret")
	}
	if err := InitJWT(&configs.JWTConfig{Secret: strings.Repeat("x", 31)}); err == nil {
		t.Fatal("expected error for 31-byte secret")
	}
	if err := InitJWT(nil); err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestGenerateAndValidate(t *testing.T) {
	initTestJWT(t)
	tok, err := GenerateToken(&models.User{ID: 42, Username: "bob", Role: "admin"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	c, err := ValidateToken(tok)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if c.UserID != 42 || c.Username != "bob" || c.Role != "admin" || c.Issuer != Issuer {
		t.Fatalf("unexpected claims: %+v", c)
	}
	if c.IssuedAt == nil || c.ExpiresAt == nil {
		t.Fatal("iat/exp must be set")
	}
}

func TestValidateRejects(t *testing.T) {
	initTestJWT(t)
	key := []byte(testSecret)

	expired := validClaims()
	expired.IssuedAt = jwt.NewNumericDate(time.Now().Add(-2 * time.Hour))
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Second))

	noExp := validClaims()
	noExp.ExpiresAt = nil

	noIat := validClaims()
	noIat.IssuedAt = nil

	futureIat := validClaims()
	futureIat.IssuedAt = jwt.NewNumericDate(time.Now().Add(10 * time.Minute))

	wrongIss := validClaims()
	wrongIss.Issuer = "someone-else"

	noUser := validClaims()
	noUser.UserID = 0

	cases := map[string]string{
		"expired":      signWith(t, jwt.SigningMethodHS256, key, expired),
		"missing exp":  signWith(t, jwt.SigningMethodHS256, key, noExp),
		"missing iat":  signWith(t, jwt.SigningMethodHS256, key, noIat),
		"future iat":   signWith(t, jwt.SigningMethodHS256, key, futureIat),
		"wrong issuer": signWith(t, jwt.SigningMethodHS256, key, wrongIss),
		"no user id":   signWith(t, jwt.SigningMethodHS256, key, noUser),
		"wrong secret": signWith(t, jwt.SigningMethodHS256, []byte(strings.Repeat("z", 40)), validClaims()),
		"HS512":        signWith(t, jwt.SigningMethodHS512, key, validClaims()),
		"alg none":     signWith(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"garbage":      "not.a.jwt",
		"empty":        "",
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateToken(tok); err == nil {
				t.Fatalf("%s: expected rejection", name)
			}
		})
	}

	// Sanity: the same helper produces an accepted token when claims are valid.
	if _, err := ValidateToken(signWith(t, jwt.SigningMethodHS256, key, validClaims())); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
}

func TestTokenFromRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mk := func(setup func(r *http.Request)) string {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		setup(r)
		c.Request = r
		return TokenFromRequest(c)
	}
	if got := mk(func(r *http.Request) { r.AddCookie(&http.Cookie{Name: CookieName, Value: "cookie-tok"}) }); got != "cookie-tok" {
		t.Fatalf("cookie: got %q", got)
	}
	if got := mk(func(r *http.Request) { r.Header.Set("Authorization", "Bearer hdr-tok") }); got != "hdr-tok" {
		t.Fatalf("bearer: got %q", got)
	}
	if got := mk(func(r *http.Request) {}); got != "" {
		t.Fatalf("none: got %q", got)
	}
}
