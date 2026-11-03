package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CookieName is the httpOnly cookie that carries the JWT for browser clients.
const CookieName = "dj_token"

var cookieSecure = true

// SetCookieSecure controls the Secure flag on the auth cookie (disable only for plain-HTTP local dev).
func SetCookieSecure(secure bool) {
	cookieSecure = secure
}

// SetAuthCookie stores the JWT in an httpOnly, SameSite=Strict cookie.
func SetAuthCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(jwtExpiration.Seconds()),
		HttpOnly: true,
		Secure:   cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearAuthCookie expires the auth cookie.
func ClearAuthCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
}

// TokenFromRequest returns the JWT from the auth cookie, falling back to a Bearer header for API clients.
func TokenFromRequest(c *gin.Context) string {
	if cookie, err := c.Cookie(CookieName); err == nil && cookie != "" {
		return cookie
	}
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
