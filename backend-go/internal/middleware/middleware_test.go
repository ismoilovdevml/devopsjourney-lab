package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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
