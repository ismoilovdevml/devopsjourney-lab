package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// OriginAllowlist is an exact-match set of allowed origins (scheme://host[:port]).
// Wildcards are never honoured because the API uses credentials (cookies).
type OriginAllowlist map[string]struct{}

// NewOriginAllowlist normalizes origins and drops "*" entries.
func NewOriginAllowlist(origins []string) OriginAllowlist {
	set := make(OriginAllowlist, len(origins))
	for _, o := range origins {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o == "" {
			continue
		}
		if strings.Contains(o, "*") {
			log.Warn().Str("origin", o).Msg("ignoring wildcard CORS origin: not allowed with credentials")
			continue
		}
		set[strings.ToLower(o)] = struct{}{}
	}
	return set
}

// Allowed reports whether origin is in the allowlist (case-insensitive exact match).
func (a OriginAllowlist) Allowed(origin string) bool {
	if origin == "" {
		return false
	}
	_, ok := a[strings.ToLower(strings.TrimRight(origin, "/"))]
	return ok
}

// IsSameHost reports whether origin's host:port equals the request Host. A browser on another
// site cannot forge Origin, so a same-host Origin is safe even if not listed in CORS_ORIGINS.
func IsSameHost(origin string, r *http.Request) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || r.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

// OriginPermitted is the shared check for CSRF/WebSocket origin validation.
func (a OriginAllowlist) OriginPermitted(origin string, r *http.Request) bool {
	return a.Allowed(origin) || IsSameHost(origin, r)
}

// CORSMiddleware reflects the Origin only when it is in the exact-match allowlist and
// always sends credentials=true for those; it never emits "*".
func CORSMiddleware(origins []string) gin.HandlerFunc {
	allow := NewOriginAllowlist(origins)
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		h := c.Writer.Header()
		h.Add("Vary", "Origin")

		if allow.Allowed(origin) {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After")
			if c.Request.Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				h.Set("Access-Control-Max-Age", "600")
			}
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// OriginGuard rejects state-changing requests (anything but GET/HEAD/OPTIONS) whose Origin
// (or, if absent, Referer) is present and not allowed. Requests with neither header (non-browser
// API clients) pass; browsers always send Origin on cross-site state-changing requests.
func OriginGuard(origins []string) gin.HandlerFunc {
	allow := NewOriginAllowlist(origins)
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			if ref := c.GetHeader("Referer"); ref != "" {
				origin = originFromURL(ref)
				if origin == "" {
					rejectOrigin(c, ref)
					return
				}
			}
		}
		if origin != "" && !allow.OriginPermitted(origin, c.Request) {
			rejectOrigin(c, origin)
			return
		}
		c.Next()
	}
}

func rejectOrigin(c *gin.Context, origin string) {
	log.Warn().
		Str("origin", truncateForLog(origin, 200)).
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Str("ip", c.ClientIP()).
		Msg("rejected request from disallowed origin")
	abortJSON(c, http.StatusForbidden, "Forbidden", "Origin not allowed")
}

// originFromURL returns scheme://host of an absolute URL, or "" if it is not one.
func originFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func truncateForLog(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
