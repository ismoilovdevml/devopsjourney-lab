package websocket

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/devops-lab/backend/internal/middleware"
	"github.com/devops-lab/backend/internal/models"
	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func TestParseControlFrame(t *testing.T) {
	cases := []struct {
		in                  string
		cols, rows          uint
		isControl, validDim bool
	}{
		{`{"type":"resize","cols":120,"rows":40}`, 120, 40, true, true},
		{` {"type":"resize","cols":1,"rows":500} `, 1, 500, true, true},
		{`{"type":"resize","cols":0,"rows":40}`, 0, 0, true, false},
		{`{"type":"resize","cols":501,"rows":40}`, 0, 0, true, false},
		{`{"type":"resize","cols":80,"rows":-1}`, 0, 0, true, false},
		{`{"type":"other","cols":80,"rows":24}`, 0, 0, false, false},
		{`{not json`, 0, 0, false, false},
		{`ls -la`, 0, 0, false, false},
		{"", 0, 0, false, false},
		{`{"type":"resize","cols":80,"rows":24,"pad":"` + strings.Repeat("x", 300) + `"}`, 0, 0, false, false},
	}
	for _, tc := range cases {
		cols, rows, isControl, ok := parseControlFrame([]byte(tc.in))
		if cols != tc.cols || rows != tc.rows || isControl != tc.isControl || ok != tc.validDim {
			t.Errorf("parseControlFrame(%.40q) = (%d,%d,%v,%v), want (%d,%d,%v,%v)",
				tc.in, cols, rows, isControl, ok, tc.cols, tc.rows, tc.isControl, tc.validDim)
		}
	}
}

func TestCheckOrigin(t *testing.T) {
	check := CheckOriginFromConfig([]string{"https://lab.example.com"})
	mk := func(origin, host string) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/ws", nil)
		r.Host = host
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		return r
	}
	cases := []struct {
		origin, host string
		want         bool
	}{
		{"https://lab.example.com", "backend:8080", true},
		{"https://evil.com", "backend:8080", false},
		{"null", "backend:8080", false},
		{"https://lab.example.com.evil.com", "backend:8080", false},
		{"http://backend:8080", "backend:8080", true}, // same host
		{"", "backend:8080", true},                    // non-browser client
	}
	for _, tc := range cases {
		if got := check(mk(tc.origin, tc.host)); got != tc.want {
			t.Errorf("origin %q host %q: got %v want %v", tc.origin, tc.host, got, tc.want)
		}
	}
	// Default upgrader (before InitUpgrader) is same-host only.
	if upgrader.CheckOrigin(mk("https://lab.example.com", "backend:8080")) {
		t.Error("default CheckOrigin must not accept foreign origins")
	}
}

func TestConnLimiter(t *testing.T) {
	l := newConnLimiter(3)
	for i := 0; i < 3; i++ {
		if !l.acquire(1) {
			t.Fatalf("acquire %d failed", i)
		}
	}
	if l.acquire(1) {
		t.Fatal("4th connection must be refused")
	}
	if !l.acquire(2) {
		t.Fatal("other user must not be affected")
	}
	l.release(1)
	if !l.acquire(1) {
		t.Fatal("slot not released")
	}

	// Concurrent safety.
	l2 := newConnLimiter(3)
	var wg sync.WaitGroup
	var mu sync.Mutex
	granted := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if l2.acquire(9) {
				mu.Lock()
				granted++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if granted != 3 {
		t.Fatalf("granted %d, want 3", granted)
	}
}

func stubAuth(t *testing.T, id *middleware.Identity, status int, err error, sess *models.Session, sessErr error) {
	t.Helper()
	origAuth, origFind := authenticate, findSession
	authenticate = func(c *gin.Context) (*middleware.Identity, int, error) { return id, status, err }
	findSession = func(sessionID string, userID uint) (*models.Session, error) {
		if sessErr != nil {
			return nil, sessErr
		}
		if sess.ID != sessionID || sess.UserID != userID {
			return nil, errSessionNotFound
		}
		return sess, nil
	}
	t.Cleanup(func() { authenticate, findSession = origAuth, origFind })
}

func runAuth(path string) (*httptest.ResponseRecorder, error) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, path, nil)
	_, _, err := AuthenticateWebSocket(c)
	return w, err
}

func TestAuthenticateWebSocket(t *testing.T) {
	sess := &models.Session{ID: "abc123def456", UserID: 1, Status: "active", ExpiresAt: time.Now().Add(time.Hour)}
	alice := &middleware.Identity{UserID: 1, Username: "alice", Role: "student"}

	t.Run("unauthenticated", func(t *testing.T) {
		stubAuth(t, nil, http.StatusUnauthorized, middleware.ErrUnauthenticated, sess, nil)
		if w, err := runAuth("/ws?sessionId=abc123def456"); err == nil || w.Code != http.StatusUnauthorized {
			t.Fatalf("got %d err=%v", w.Code, err)
		}
	})
	t.Run("disabled user", func(t *testing.T) {
		stubAuth(t, nil, http.StatusForbidden, middleware.ErrUserDisabled, sess, nil)
		if w, _ := runAuth("/ws?sessionId=abc123def456"); w.Code != http.StatusForbidden {
			t.Fatalf("got %d", w.Code)
		}
	})
	t.Run("owner", func(t *testing.T) {
		stubAuth(t, alice, http.StatusOK, nil, sess, nil)
		if w, err := runAuth("/ws?sessionId=abc123def456"); err != nil {
			t.Fatalf("got %d err=%v", w.Code, err)
		}
	})
	t.Run("not owner gets 404", func(t *testing.T) {
		stubAuth(t, &middleware.Identity{UserID: 2, Username: "mallory"}, http.StatusOK, nil, sess, nil)
		if w, _ := runAuth("/ws?sessionId=abc123def456"); w.Code != http.StatusNotFound {
			t.Fatalf("got %d", w.Code)
		}
	})
	t.Run("missing or malformed session id", func(t *testing.T) {
		stubAuth(t, alice, http.StatusOK, nil, sess, nil)
		for _, p := range []string{"/ws", "/ws?sessionId=", "/ws?sessionId=../../etc", "/ws?sessionId=" + strings.Repeat("a", 65)} {
			if w, _ := runAuth(p); w.Code != http.StatusBadRequest {
				t.Fatalf("%s: got %d", p, w.Code)
			}
		}
	})
	t.Run("token query param is ignored", func(t *testing.T) {
		// authenticate is the real middleware path here: no cookie/header => 401 even with ?token=.
		origAuth := authenticate
		authenticate = middleware.Authenticate
		t.Cleanup(func() { authenticate = origAuth })
		if w, _ := runAuth("/ws?sessionId=abc123def456&token=whatever"); w.Code != http.StatusUnauthorized {
			t.Fatalf("got %d", w.Code)
		}
	})
	t.Run("db error", func(t *testing.T) {
		stubAuth(t, alice, http.StatusOK, nil, sess, errors.New("db down"))
		if w, _ := runAuth("/ws?sessionId=abc123def456"); w.Code != http.StatusServiceUnavailable {
			t.Fatalf("got %d", w.Code)
		}
	})
}

func TestCloseAllEmpty(t *testing.T) {
	if n := CloseAll("test"); n != 0 || ActiveConnections() != 0 {
		t.Fatalf("CloseAll on empty registry = %d", n)
	}
}
