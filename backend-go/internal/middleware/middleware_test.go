package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

const testSecret = "middleware-test-secret-0123456789abcdef"

// stubUsers installs a fake user lookup and returns a counter of DB hits.
func stubUsers(t *testing.T, users map[uint]*UserState, dbErr error) *int32 {
	t.Helper()
	if err := auth.InitJWT(&configs.JWTConfig{Secret: testSecret, Expiration: time.Hour}); err != nil {
		t.Fatal(err)
	}
	var hits int32
	orig := lookupUser
	lookupUser = func(id uint) (*UserState, error) {
		atomic.AddInt32(&hits, 1)
		if dbErr != nil {
			return nil, dbErr
		}
		if u, ok := users[id]; ok {
			cp := *u
			return &cp, nil
		}
		return nil, errUserNotFound
	}
	resetUserCache()
	t.Cleanup(func() { lookupUser = orig; resetUserCache() })
	return &hits
}

func tokenFor(t *testing.T, id uint, role string) string {
	t.Helper()
	tok, err := auth.GenerateToken(&models.User{ID: id, Username: "u", Role: role})
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func authRouter(mw ...gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	handlers := append(mw, func(c *gin.Context) {
		uid, _ := c.Get("user_id")
		role, _ := c.Get("role")
		name, _ := c.Get("username")
		id, _ := uid.(uint)
		c.JSON(http.StatusOK, gin.H{"user_id": id, "role": role, "username": name})
	})
	r.GET("/p", handlers...)
	return r
}

func do(r http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuthMiddleware(t *testing.T) {
	hits := stubUsers(t, map[uint]*UserState{
		1: {ID: 1, Username: "alice", Role: "student", Active: true},
		2: {ID: 2, Username: "bob", Role: "student", Active: false},
		3: {ID: 3, Username: "carol", Role: "student", Active: true}, // token claims admin
	}, nil)
	r := authRouter(AuthMiddleware())

	t.Run("cookie", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tokenFor(t, 1, "student")})
		w := do(r, req)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"username":"alice"`) || !strings.Contains(w.Body.String(), `"user_id":1`) {
			t.Fatalf("got %d %s", w.Code, w.Body)
		}
	})
	t.Run("bearer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, 1, "student"))
		if w := do(r, req); w.Code != http.StatusOK {
			t.Fatalf("got %d %s", w.Code, w.Body)
		}
	})
	t.Run("missing", func(t *testing.T) {
		w := do(r, httptest.NewRequest(http.MethodGet, "/p", nil))
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), `"code":401`) {
			t.Fatalf("got %d %s", w.Code, w.Body)
		}
	})
	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("Authorization", "Bearer garbage")
		if w := do(r, req); w.Code != http.StatusUnauthorized {
			t.Fatalf("got %d", w.Code)
		}
	})
	t.Run("inactive user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, 2, "student"))
		if w := do(r, req); w.Code != http.StatusForbidden {
			t.Fatalf("got %d", w.Code)
		}
	})
	t.Run("deleted user", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, 99, "student"))
		if w := do(r, req); w.Code != http.StatusUnauthorized {
			t.Fatalf("got %d", w.Code)
		}
	})
	t.Run("role comes from DB not token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, 3, "admin"))
		w := do(r, req)
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"role":"student"`) {
			t.Fatalf("got %d %s", w.Code, w.Body)
		}
	})
	t.Run("cache avoids repeated lookups", func(t *testing.T) {
		resetUserCache()
		atomic.StoreInt32(hits, 0)
		tok := tokenFor(t, 1, "student")
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/p", nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			do(r, req)
		}
		if n := atomic.LoadInt32(hits); n != 1 {
			t.Fatalf("lookups = %d, want 1", n)
		}
		InvalidateUserCache(1)
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		do(r, req)
		if n := atomic.LoadInt32(hits); n != 2 {
			t.Fatalf("lookups after invalidate = %d, want 2", n)
		}
	})
}

func TestTokenRevoked(t *testing.T) {
	changed := time.Date(2026, 1, 2, 3, 4, 5, 600_000_000, time.UTC)
	cases := []struct {
		name string
		iat  time.Time
		pca  *time.Time
		want bool
	}{
		{"never changed", changed.Add(-time.Hour), nil, false},
		{"issued earlier", changed.Add(-2 * time.Second), &changed, true},
		{"issued one second before", changed.Truncate(time.Second).Add(-time.Second), &changed, true},
		// iat has second precision: a token minted in the same second (the re-issued cookie) is valid.
		{"same second", changed.Truncate(time.Second), &changed, false},
		{"issued later", changed.Add(time.Minute), &changed, false},
	}
	for _, tc := range cases {
		if got := tokenRevoked(tc.iat, tc.pca); got != tc.want {
			t.Errorf("%s: tokenRevoked = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestAuthMiddlewareRejectsTokensIssuedBeforePasswordChange(t *testing.T) {
	future := time.Now().Add(5 * time.Second)
	past := time.Now().Add(-5 * time.Second)
	stubUsers(t, map[uint]*UserState{
		1: {ID: 1, Username: "alice", Role: "student", Active: true, PasswordChangedAt: &future},
		2: {ID: 2, Username: "bob", Role: "student", Active: true, PasswordChangedAt: &past},
	}, nil)
	r := authRouter(AuthMiddleware())

	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tokenFor(t, 1, "student")})
	if w := do(r, req); w.Code != http.StatusUnauthorized {
		t.Fatalf("token issued before password change: got %d %s", w.Code, w.Body)
	}
	req = httptest.NewRequest(http.MethodGet, "/p", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tokenFor(t, 2, "student")})
	if w := do(r, req); w.Code != http.StatusOK {
		t.Fatalf("token issued after password change: got %d %s", w.Code, w.Body)
	}

	// The cached state carries PasswordChangedAt, so a cache hit must still reject.
	req = httptest.NewRequest(http.MethodGet, "/p", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tokenFor(t, 1, "student")})
	if _, status, err := Authenticate(ginContext(req)); !errors.Is(err, ErrTokenRevoked) || status != http.StatusUnauthorized {
		t.Fatalf("Authenticate (cached) = %d, %v; want 401, ErrTokenRevoked", status, err)
	}
}

func ginContext(req *http.Request) *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	return c
}

func TestAuthMiddlewareDBErrorFailsClosed(t *testing.T) {
	stubUsers(t, nil, errors.New("db down"))
	r := authRouter(AuthMiddleware())
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer "+tokenFor(t, 1, "student"))
	if w := do(r, req); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d", w.Code)
	}
}

func TestOptionalAuthMiddleware(t *testing.T) {
	stubUsers(t, map[uint]*UserState{1: {ID: 1, Username: "alice", Role: "student", Active: true}}, nil)
	r := authRouter(OptionalAuthMiddleware())

	w := do(r, httptest.NewRequest(http.MethodGet, "/p", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"user_id":0`) {
		t.Fatalf("anonymous: %d %s", w.Code, w.Body)
	}
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	if w := do(r, req); w.Code != http.StatusOK {
		t.Fatalf("bad token must not abort: %d", w.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/p", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tokenFor(t, 1, "student")})
	if w := do(r, req); !strings.Contains(w.Body.String(), `"user_id":1`) {
		t.Fatalf("valid cookie: %s", w.Body)
	}
}

func TestRoleMiddleware(t *testing.T) {
	stubUsers(t, map[uint]*UserState{
		1: {ID: 1, Username: "alice", Role: "student", Active: true},
		2: {ID: 2, Username: "root", Role: "admin", Active: true},
	}, nil)
	r := authRouter(AuthMiddleware(), RoleMiddleware("admin"))
	for id, want := range map[uint]int{1: http.StatusForbidden, 2: http.StatusOK} {
		req := httptest.NewRequest(http.MethodGet, "/p", nil)
		req.Header.Set("Authorization", "Bearer "+tokenFor(t, id, "admin"))
		if w := do(r, req); w.Code != want {
			t.Errorf("user %d: got %d want %d", id, w.Code, want)
		}
	}
	// Without AuthMiddleware there is no role at all.
	if w := do(authRouter(RoleMiddleware("admin")), httptest.NewRequest(http.MethodGet, "/p", nil)); w.Code != http.StatusForbidden {
		t.Fatalf("no role: %d", w.Code)
	}
}

func guardRouter() *gin.Engine {
	r := gin.New()
	r.Use(OriginGuard([]string{"https://lab.example.com", "http://localhost:3000/"}))
	ok := func(c *gin.Context) { c.Status(http.StatusNoContent) }
	r.POST("/x", ok)
	r.GET("/x", ok)
	r.DELETE("/x", ok)
	return r
}

func TestOriginGuard(t *testing.T) {
	r := guardRouter()
	cases := []struct {
		name, method, origin, referer, host string
		want                                int
	}{
		{"allowed origin", http.MethodPost, "https://lab.example.com", "", "api.internal", 204},
		{"allowed origin trailing slash in config", http.MethodPost, "http://localhost:3000", "", "api.internal", 204},
		{"evil origin", http.MethodPost, "https://evil.com", "", "api.internal", 403},
		{"evil origin delete", http.MethodDelete, "https://evil.com", "", "api.internal", 403},
		{"null origin", http.MethodPost, "null", "", "api.internal", 403},
		{"lookalike origin", http.MethodPost, "https://lab.example.com.evil.com", "", "api.internal", 403},
		{"scheme mismatch", http.MethodPost, "http://lab.example.com", "", "api.internal", 403},
		{"same host", http.MethodPost, "https://api.internal", "", "api.internal", 204},
		{"no origin no referer (api client)", http.MethodPost, "", "", "api.internal", 204},
		{"allowed referer", http.MethodPost, "", "https://lab.example.com/tasks/1", "api.internal", 204},
		{"evil referer", http.MethodPost, "", "https://evil.com/page", "api.internal", 403},
		{"relative referer", http.MethodPost, "", "/relative", "api.internal", 403},
		{"GET from evil origin passes", http.MethodGet, "https://evil.com", "", "api.internal", 204},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/x", nil)
			req.Host = tc.host
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			if tc.referer != "" {
				req.Header.Set("Referer", tc.referer)
			}
			if w := do(r, req); w.Code != tc.want {
				t.Fatalf("got %d want %d (%s)", w.Code, tc.want, w.Body)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(CORSMiddleware([]string{"https://lab.example.com", "*"}))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://lab.example.com")
	w := do(r, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://lab.example.com" {
		t.Fatalf("ACAO = %q", got)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credentials header missing")
	}
	if !strings.Contains(strings.Join(w.Header().Values("Vary"), ","), "Origin") {
		t.Fatal("Vary: Origin missing")
	}

	// "*" in config must not allow arbitrary origins.
	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://evil.com")
	w = do(r, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("evil origin got ACAO %q", got)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("credentials must not be granted to disallowed origin")
	}

	// Preflight.
	req = httptest.NewRequest(http.MethodOptions, "/x", nil)
	req.Header.Set("Origin", "https://lab.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	w = do(r, req)
	if w.Code != http.StatusNoContent || !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("preflight: %d %v", w.Code, w.Header())
	}
}

func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/api/me", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := do(r, httptest.NewRequest(http.MethodGet, "/api/me", nil))
	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Cache-Control":          "no-store",
	}
	for k, v := range want {
		if got := w.Header().Get(k); got != v {
			t.Errorf("%s = %q want %q", k, got, v)
		}
	}
	if w.Header().Get("Permissions-Policy") == "" {
		t.Error("Permissions-Policy missing")
	}
	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must not be sent over plain HTTP")
	}

	w = do(r, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Header().Get("Cache-Control") != "" {
		t.Error("Cache-Control should only be forced on /api")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	if w := do(r, req); !strings.HasPrefix(w.Header().Get("Strict-Transport-Security"), "max-age=") {
		t.Error("HSTS missing behind TLS proxy")
	}
}

func TestBodySizeLimit(t *testing.T) {
	r := gin.New()
	r.Use(BodySizeLimit(16))
	r.POST("/x", func(c *gin.Context) {
		b, err := io.ReadAll(c.Request.Body)
		if err != nil {
			if IsBodyTooLarge(err) {
				c.Status(http.StatusRequestEntityTooLarge)
				return
			}
			c.Status(http.StatusBadRequest)
			return
		}
		c.String(http.StatusOK, string(b))
	})

	if w := do(r, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("small"))); w.Code != 200 || w.Body.String() != "small" {
		t.Fatalf("small: %d %q", w.Code, w.Body)
	}
	if w := do(r, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("a", 17)))); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("declared oversize: %d", w.Code)
	}
	// Unknown length (chunked): enforced while reading.
	req := httptest.NewRequest(http.MethodPost, "/x", io.MultiReader(bytes.NewReader(bytes.Repeat([]byte("a"), 64))))
	req.ContentLength = -1
	if w := do(r, req); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("chunked oversize: %d", w.Code)
	}
}

func TestRequestID(t *testing.T) {
	r := gin.New()
	r.Use(RequestID(), LoggerMiddleware())
	r.GET("/x", func(c *gin.Context) {
		id, _ := c.Get("request_id")
		c.String(http.StatusOK, id.(string))
	})

	w := do(r, httptest.NewRequest(http.MethodGet, "/x", nil))
	gen := w.Header().Get(RequestIDHeader)
	if len(gen) != 32 || w.Body.String() != gen {
		t.Fatalf("generated id %q body %q", gen, w.Body)
	}

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(RequestIDHeader, "proxy-id-12345")
	if got := do(r, req).Header().Get(RequestIDHeader); got != "proxy-id-12345" {
		t.Fatalf("incoming id not kept: %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(RequestIDHeader, "bad id\nwith newline")
	if got := do(r, req).Header().Get(RequestIDHeader); got == "bad id\nwith newline" || len(got) != 32 {
		t.Fatalf("malformed id must be replaced: %q", got)
	}
}

func TestAuthRateLimiterInMemoryFallback(t *testing.T) {
	// database.RedisClient is nil in unit tests -> in-memory limiter.
	fallbackLimiter.reset()
	t.Cleanup(fallbackLimiter.reset)

	r := gin.New()
	rl := AuthRateLimiter()
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.POST("/api/auth/login", rl, ok)
	r.POST("/api/auth/register", AuthRateLimiter(), ok)

	send := func(path, ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.RemoteAddr = ip + ":1234"
		return do(r, req)
	}
	for i := 0; i < 6; i++ {
		if w := send("/api/auth/login", "203.0.113.5"); w.Code != 200 {
			t.Fatalf("login %d: %d", i, w.Code)
		}
	}
	for i := 0; i < 4; i++ {
		if w := send("/api/auth/register", "203.0.113.5"); w.Code != 200 {
			t.Fatalf("register %d: %d", i, w.Code)
		}
	}
	// 11th request in the window (login+register share the bucket).
	w := send("/api/auth/login", "203.0.113.5")
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Fatalf("11th: %d retry-after=%q", w.Code, w.Header().Get("Retry-After"))
	}
	// Other IPs are unaffected.
	if w := send("/api/auth/login", "203.0.113.6"); w.Code != 200 {
		t.Fatalf("other ip: %d", w.Code)
	}
}

func TestClientIPNotSpoofableWithoutTrustedProxy(t *testing.T) {
	fallbackLimiter.reset()
	t.Cleanup(fallbackLimiter.reset)
	r := gin.New()
	if err := r.SetTrustedProxies([]string{"10.0.0.0/8"}); err != nil {
		t.Fatal(err)
	}
	r.POST("/login", AuthRateLimiter(), func(c *gin.Context) { c.Status(200) })
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "198.51.100.9:5555" // untrusted peer
		req.Header.Set("X-Forwarded-For", "192.0.2."+string(rune('1'+i%9)))
		w := do(r, req)
		if i == 10 && w.Code != http.StatusTooManyRequests {
			t.Fatalf("rotating X-Forwarded-For bypassed the limiter: %d", w.Code)
		}
	}
}

func TestAuthRateLimiterConfigurable(t *testing.T) {
	fallbackLimiter.reset()
	t.Cleanup(func() { fallbackLimiter.reset(); SetAuthRateLimit(DefaultAuthRateLimit) })

	SetAuthRateLimit(3)
	r := gin.New()
	r.POST("/login", AuthRateLimiter(), func(c *gin.Context) { c.Status(http.StatusOK) })
	send := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "198.51.100.9:1234"
		return do(r, req)
	}
	for i := 0; i < 3; i++ {
		if w := send(); w.Code != http.StatusOK || w.Header().Get("X-RateLimit-Limit") != "3" {
			t.Fatalf("request %d: %d limit=%q", i, w.Code, w.Header().Get("X-RateLimit-Limit"))
		}
	}
	if w := send(); w.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request: %d, want 429", w.Code)
	}

	SetAuthRateLimit(0) // clamped to 1
	if got := authRateLimit.Load(); got != 1 {
		t.Fatalf("clamp low: %d", got)
	}
	SetAuthRateLimit(1 << 30) // clamped to 10000
	if got := authRateLimit.Load(); got != 10000 {
		t.Fatalf("clamp high: %d", got)
	}
}

func TestSessionRateLimiterPerUser(t *testing.T) {
	fallbackLimiter.reset()
	t.Cleanup(func() { fallbackLimiter.reset(); SetSessionRateLimit(DefaultSessionRateLimit) })

	SetSessionRateLimit(DefaultSessionRateLimit)
	r := gin.New()
	asUser := func(c *gin.Context) {
		if v := c.GetHeader("X-Test-User"); v != "" {
			id, _ := strconv.Atoi(v)
			c.Set("user_id", uint(id))
		}
		c.Next()
	}
	rl := SessionRateLimiter()
	ok := func(c *gin.Context) { c.Status(http.StatusCreated) }
	r.POST("/api/sessions", asUser, rl, ok)
	r.POST("/api/sessions/:id/recreate", asUser, rl, ok)

	send := func(path, user string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.RemoteAddr = "198.51.100.20:1234" // same IP for everyone: the key is the user
		if user != "" {
			req.Header.Set("X-Test-User", user)
		}
		return do(r, req)
	}
	// 6 per minute, shared between create and recreate.
	for i := 0; i < 4; i++ {
		if w := send("/api/sessions", "7"); w.Code != http.StatusCreated {
			t.Fatalf("create %d: %d", i, w.Code)
		}
	}
	for i := 0; i < 2; i++ {
		if w := send("/api/sessions/abc/recreate", "7"); w.Code != http.StatusCreated {
			t.Fatalf("recreate %d: %d", i, w.Code)
		}
	}
	w := send("/api/sessions", "7")
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Fatalf("7th: %d retry-after=%q", w.Code, w.Header().Get("Retry-After"))
	}
	if !strings.Contains(w.Body.String(), "Too many lab starts") {
		t.Fatalf("body = %s", w.Body.String())
	}
	// Another user behind the same IP is unaffected.
	if w := send("/api/sessions", "8"); w.Code != http.StatusCreated {
		t.Fatalf("other user: %d", w.Code)
	}
	// Unauthenticated requests are not counted (AuthMiddleware handles them).
	for i := 0; i < 10; i++ {
		if w := send("/api/sessions", ""); w.Code != http.StatusCreated {
			t.Fatalf("anonymous %d: %d", i, w.Code)
		}
	}

	SetSessionRateLimit(0)
	if got := sessionRateLimit.Load(); got != 1 {
		t.Fatalf("clamp low: %d", got)
	}
}

func TestRevalidate(t *testing.T) {
	changed := time.Now().Add(-time.Minute)
	stubUsers(t, map[uint]*UserState{
		1: {ID: 1, Username: "ok", Role: "user", Active: true},
		2: {ID: 2, Username: "off", Role: "user", Active: false},
		3: {ID: 3, Username: "pw", Role: "user", Active: true, PasswordChangedAt: &changed},
	}, nil)

	if err := Revalidate(&Identity{UserID: 1, IssuedAt: time.Now().Add(-time.Hour)}); err != nil {
		t.Fatalf("active user: %v", err)
	}
	if err := Revalidate(&Identity{UserID: 2, IssuedAt: time.Now()}); !errors.Is(err, ErrUserDisabled) {
		t.Fatalf("disabled user: %v", err)
	}
	if err := Revalidate(&Identity{UserID: 3, IssuedAt: changed.Add(-time.Hour)}); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("token older than password change: %v", err)
	}
	if err := Revalidate(&Identity{UserID: 3, IssuedAt: changed.Add(time.Second)}); err != nil {
		t.Fatalf("token newer than password change: %v", err)
	}
	if err := Revalidate(&Identity{UserID: 99, IssuedAt: time.Now()}); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("deleted user: %v", err)
	}
}

func TestRevalidateTransientError(t *testing.T) {
	stubUsers(t, nil, errors.New("db down"))
	err := Revalidate(&Identity{UserID: 1, IssuedAt: time.Now()})
	if err == nil || errors.Is(err, ErrUnauthenticated) || errors.Is(err, ErrUserDisabled) || errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("transient error must be distinguishable, got %v", err)
	}
}

func TestAuthenticateSetsIssuedAt(t *testing.T) {
	stubUsers(t, map[uint]*UserState{5: {ID: 5, Username: "u", Role: "user", Active: true}}, nil)
	tok := tokenFor(t, 5, "user")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tok)
	id, _, err := Authenticate(c)
	if err != nil {
		t.Fatal(err)
	}
	if id.IssuedAt.IsZero() || time.Since(id.IssuedAt) > time.Minute {
		t.Fatalf("IssuedAt = %v", id.IssuedAt)
	}
}
