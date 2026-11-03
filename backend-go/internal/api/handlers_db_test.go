package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/auth"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/session"
	"github.com/devops-lab/backend/internal/task"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// These tests need Postgres. Configure with DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME
// (defaults match: docker run -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test
// -e POSTGRES_DB=devopslab_test -p 55432:5432 postgres:17-alpine). They are skipped when
// the database is unreachable. The schema of the target database is dropped and recreated.

const testOrigin = "http://localhost:5173"

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// fakeLab is an in-process LabManager that persists sessions like the real manager.
type fakeLab struct {
	mu     sync.Mutex
	result models.CheckResponse
	seq    int
}

func (f *fakeLab) CreateSession(ctx context.Context, userID uint, taskID string) (*models.SessionResponse, error) {
	f.mu.Lock()
	f.seq++
	id := fmt.Sprintf("sess%08d", f.seq)
	f.mu.Unlock()
	now := time.Now().UTC()
	// One active session per user.
	database.DB.Model(&models.Session{}).Where("user_id = ? AND status = ?", userID, models.SessionActive).
		Updates(map[string]any{"status": models.SessionStopped, "ended_at": now})
	s := models.Session{ID: id, UserID: userID, TaskID: taskID, ContainerID: "c-" + id,
		Status: models.SessionActive, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := database.DB.Create(&s).Error; err != nil {
		return nil, err
	}
	r := toSessionResponse(&s, now)
	return &r, nil
}

func (f *fakeLab) GetSession(ctx context.Context, id string) (*models.Session, error) {
	var s models.Session
	err := database.DB.Where("id = ? AND status = ?", id, models.SessionActive).Take(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, session.ErrSessionNotFound // same contract as session.Manager
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (f *fakeLab) DeleteSession(ctx context.Context, id string) error {
	res := database.DB.Model(&models.Session{}).Where("id = ? AND status = ?", id, models.SessionActive).
		Updates(map[string]any{"status": models.SessionStopped, "ended_at": time.Now().UTC()})
	if res.RowsAffected == 0 {
		return session.ErrSessionNotFound
	}
	return res.Error
}

func (f *fakeLab) CheckSolution(ctx context.Context, id string) (*models.CheckResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := f.result
	return &r, nil
}

func (f *fakeLab) setResult(score, total int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.result = models.CheckResponse{Passed: score == total, Score: score, Total: total,
		Results: []models.CheckResult{{Name: "c1", OK: score == total}}}
}

func (f *fakeLab) Recreate(ctx context.Context, id string) (*models.SessionResponse, error) {
	s, err := f.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return f.CreateSession(ctx, s.UserID, s.TaskID)
}

func (f *fakeLab) ActiveSessionForUser(ctx context.Context, userID uint) (*models.Session, error) {
	var s models.Session
	err := database.DB.Where("user_id = ? AND status = ? AND expires_at > ?", userID, models.SessionActive, time.Now().UTC()).
		Order("created_at DESC").Take(&s).Error
	if err != nil {
		return nil, nil //nolint:nilerr // none
	}
	return &s, nil
}

func (f *fakeLab) ResourceStats(ctx context.Context) (*models.ResourceStats, error) {
	return &models.ResourceStats{ContainersRunning: 1, ContainersLimit: 20}, nil
}

type testEnv struct {
	t      *testing.T
	router *gin.Engine
	lab    *fakeLab
	ipSeq  atomic.Int64
}

func writeTestTasks(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	defs := []struct {
		id, cat     string
		order, diff int
	}{
		{"t-one", "web", 1, 1}, {"t-two", "containers", 2, 2}, {"t-three", "web", 3, 3},
	}
	for _, d := range defs {
		dir := filepath.Join(root, d.id)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		y := fmt.Sprintf("id: %s\ntitle: %s\norder: %d\ndifficulty: %d\ncategory: %s\nhints: [\"h1\"]\nsetup: [\"echo SETUP-SECRET\"]\nchecks:\n  - name: c1\n    cmd: echo CMD-SECRET\n",
			d.id, strings.ToUpper(d.id), d.order, d.diff, d.cat)
		if err := os.WriteFile(filepath.Join(dir, "task.yaml"), []byte(y), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# "+d.id+"\n\nReadme body.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func setupDB(t *testing.T) *testEnv {
	t.Helper()
	cfg := configs.DatabaseConfig{
		Host: envOr("DB_HOST", "localhost"), Port: envOr("DB_PORT", "55432"),
		User: envOr("DB_USER", "test"), Password: envOr("DB_PASSWORD", "test"),
		DBName: envOr("DB_NAME", "devopslab_test"), SSLMode: envOr("DB_SSLMODE", "disable"),
	}
	if err := database.Connect(&cfg); err != nil {
		t.Skipf("postgres not reachable: %v", err)
	}
	if err := pingDatabase(context.Background()); err != nil {
		t.Skipf("postgres not reachable: %v", err)
	}
	if err := database.DB.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public").Error; err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Migration must be idempotent.
	if err := database.Migrate(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if err := auth.InitJWT(&configs.JWTConfig{Secret: strings.Repeat("s", 40), Expiration: time.Hour}); err != nil {
		t.Fatal(err)
	}
	if err := task.Init(writeTestTasks(t)); err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.TestMode)
	lab := &fakeLab{}
	Init(lab, Options{Version: "test-version", DefaultImage: "devopslab-lab:test"})
	r := gin.New()
	SetupRoutes(r, []string{testOrigin})
	return &testEnv{t: t, router: r, lab: lab}
}

type resp struct {
	code    int
	body    []byte
	cookies []*http.Cookie
}

func (r resp) json(t *testing.T, v any) {
	t.Helper()
	if err := json.Unmarshal(r.body, v); err != nil {
		t.Fatalf("decode %s: %v", r.body, err)
	}
}

func (e *testEnv) do(method, path string, body any, cookie *http.Cookie) resp {
	e.t.Helper()
	var rd *bytes.Reader
	if s, ok := body.(string); ok {
		rd = bytes.NewReader([]byte(s))
	} else if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", testOrigin)
	// Distinct client IPs so the per-IP auth rate limit does not interfere.
	n := e.ipSeq.Add(1)
	req.RemoteAddr = fmt.Sprintf("10.77.%d.%d:4000", n/250, n%250+1)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	e.router.ServeHTTP(w, req)
	return resp{code: w.Code, body: w.Body.Bytes(), cookies: w.Result().Cookies()}
}

func (e *testEnv) expect(r resp, code int, what string) {
	e.t.Helper()
	if r.code != code {
		e.t.Fatalf("%s: got %d want %d: %s", what, r.code, code, r.body)
	}
}

func authCookie(t *testing.T, r resp) *http.Cookie {
	t.Helper()
	for _, c := range r.cookies {
		if c.Name == auth.CookieName && c.Value != "" {
			if !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
				t.Fatalf("auth cookie must be httpOnly + SameSite=Strict: %+v", c)
			}
			return &http.Cookie{Name: c.Name, Value: c.Value}
		}
	}
	t.Fatalf("no auth cookie in response")
	return nil
}

func (e *testEnv) register(username, password string) (*http.Cookie, models.UserResponse) {
	e.t.Helper()
	r := e.do("POST", "/api/auth/register", map[string]string{
		"username": username, "email": username + "@example.com", "password": password}, nil)
	e.expect(r, http.StatusCreated, "register "+username)
	var body map[string]json.RawMessage
	r.json(e.t, &body)
	if _, leaked := body["token"]; leaked || len(body) != 1 {
		e.t.Fatalf("register body must be exactly {user}: %s", r.body)
	}
	var u models.UserResponse
	if err := json.Unmarshal(body["user"], &u); err != nil {
		e.t.Fatal(err)
	}
	return authCookie(e.t, r), u
}

func keysOf(raw []byte) []string {
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestHandlersWithPostgres(t *testing.T) {
	e := setupDB(t)
	const pw = "s3cure-Passw0rd"

	alice, aliceUser := e.register("alice", pw)
	bob, _ := e.register("bob", pw)
	// carol becomes admin before her first authenticated request (auth middleware caches roles).
	carol, carolUser := e.register("carol", pw)
	if err := database.DB.Model(&models.User{}).Where("id = ?", carolUser.ID).Update("role", "admin").Error; err != nil {
		t.Fatal(err)
	}

	t.Run("auth", func(t *testing.T) {
		e.expect(e.do("POST", "/api/auth/register", map[string]string{"username": "alice", "email": "other@example.com", "password": pw}, nil), 409, "duplicate username")
		e.expect(e.do("POST", "/api/auth/register", map[string]string{"username": "Al", "email": "al@example.com", "password": pw}, nil), 400, "bad username")
		e.expect(e.do("POST", "/api/auth/register", map[string]string{"username": "dave", "email": "dave@example.com", "password": "password"}, nil), 400, "weak password")
		e.expect(e.do("POST", "/api/auth/register", map[string]string{"username": "dave", "email": "nope", "password": pw}, nil), 400, "bad email")

		wrong := e.do("POST", "/api/auth/login", map[string]string{"username": "alice", "password": "wrong-password"}, nil)
		unknown := e.do("POST", "/api/auth/login", map[string]string{"username": "nobody", "password": "wrong-password"}, nil)
		e.expect(wrong, 401, "wrong password")
		e.expect(unknown, 401, "unknown user")
		if !bytes.Equal(wrong.body, unknown.body) {
			t.Fatalf("login failures must be indistinguishable: %s vs %s", wrong.body, unknown.body)
		}
		ok := e.do("POST", "/api/auth/login", map[string]string{"username": "alice", "password": pw}, nil)
		e.expect(ok, 200, "login")
		if strings.Contains(string(ok.body), "token") {
			t.Fatalf("login must not return the token: %s", ok.body)
		}
		authCookie(t, ok)

		e.expect(e.do("GET", "/api/me", nil, nil), 401, "me without cookie")
		me := e.do("GET", "/api/me", nil, alice)
		e.expect(me, 200, "me")
		want := []string{"active", "created_at", "email", "github", "goal", "id", "role", "username"}
		if got := keysOf(me.body); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("User keys: %v", got)
		}
	})

	t.Run("profile and password", func(t *testing.T) {
		e.expect(e.do("PUT", "/api/me", map[string]string{"github": "not valid!"}, alice), 400, "bad github")
		e.expect(e.do("PUT", "/api/me", map[string]string{"goal": strings.Repeat("x", 201)}, alice), 400, "long goal")
		e.expect(e.do("PUT", "/api/me", map[string]string{"email": "bob@example.com"}, alice), 409, "email taken")
		r := e.do("PUT", "/api/me", map[string]string{"github": "alice-gh", "goal": "Become SRE"}, alice)
		e.expect(r, 200, "update me")
		var u models.UserResponse
		r.json(t, &u)
		if u.Github != "alice-gh" || u.Goal != "Become SRE" || u.Email != "alice@example.com" {
			t.Fatalf("profile: %+v", u)
		}

		e.expect(e.do("POST", "/api/me/password", map[string]string{"current_password": "nope-nope", "new_password": "An0ther-pass"}, alice), 401, "wrong current")
		e.expect(e.do("POST", "/api/me/password", map[string]string{"current_password": pw, "new_password": "short"}, alice), 400, "weak new")
		// Cross a second boundary so the old token's iat is strictly before the change.
		time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
		oldAlice := alice
		otherDevice := authCookie(t, e.do("POST", "/api/auth/login", map[string]string{"username": "alice", "password": pw}, nil))
		time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
		cp := e.do("POST", "/api/me/password", map[string]string{"current_password": pw, "new_password": "An0ther-pass"}, alice)
		e.expect(cp, 204, "change password")
		// The client that changed the password gets a fresh cookie and stays signed in ...
		alice = authCookie(t, cp)
		e.expect(e.do("GET", "/api/me", nil, alice), 200, "re-issued cookie works")
		// ... while every token issued before the change is revoked.
		e.expect(e.do("GET", "/api/me", nil, oldAlice), 401, "old cookie revoked")
		e.expect(e.do("GET", "/api/me", nil, otherDevice), 401, "other device revoked")
		var stored models.User
		database.DB.Take(&stored, aliceUser.ID)
		if stored.PasswordChangedAt == nil {
			t.Fatal("password_changed_at not set")
		}
		e.expect(e.do("POST", "/api/auth/login", map[string]string{"username": "alice", "password": "An0ther-pass"}, nil), 200, "login new password")
		e.expect(e.do("POST", "/api/auth/login", map[string]string{"username": "alice", "password": pw}, nil), 401, "old password rejected")
	})

	var sessionID string
	t.Run("tasks and sessions", func(t *testing.T) {
		var anon []models.TaskSummary
		r := e.do("GET", "/api/tasks", nil, nil)
		e.expect(r, 200, "tasks anon")
		r.json(t, &anon)
		if len(anon) != 3 || !anon[0].Unlocked || anon[1].Unlocked || anon[0].Points != 100 || anon[2].Points != 300 {
			t.Fatalf("anon tasks: %+v", anon)
		}
		if strings.Contains(string(r.body), "SECRET") {
			t.Fatal("tasks list leaks cmd/setup")
		}
		var admin []models.TaskSummary
		e.do("GET", "/api/tasks", nil, carol).json(t, &admin)
		for _, s := range admin {
			if !s.Unlocked {
				t.Fatalf("admin sees locked task %s", s.ID)
			}
		}

		d := e.do("GET", "/api/tasks/t-two", nil, alice)
		e.expect(d, 200, "task detail")
		if strings.Contains(string(d.body), "SECRET") || !strings.Contains(string(d.body), `"readme":"# t-two`) {
			t.Fatalf("detail: %s", d.body)
		}
		if !strings.Contains(string(d.body), `"image":"devopslab-lab:test"`) {
			t.Fatalf("detail must fall back to the configured default image: %s", d.body)
		}
		e.expect(e.do("GET", "/api/tasks/NOPE..", nil, nil), 400, "bad task id")
		e.expect(e.do("GET", "/api/tasks/missing", nil, nil), 404, "missing task")

		e.expect(e.do("POST", "/api/sessions", map[string]string{"taskId": "t-two"}, alice), 409, "locked task")
		e.expect(e.do("POST", "/api/sessions", map[string]string{"taskId": "../etc"}, alice), 400, "bad taskId")
		e.expect(e.do("POST", "/api/sessions", map[string]string{"taskId": "missing"}, alice), 404, "unknown task")
		e.expect(e.do("POST", "/api/sessions", map[string]string{"taskId": "t-two"}, carol), 201, "admin may start locked task")

		e.expect(e.do("GET", "/api/sessions/active", nil, alice), 204, "no active session")
		cr := e.do("POST", "/api/sessions", map[string]string{"taskId": "t-one"}, alice)
		e.expect(cr, 201, "create session")
		var sr models.SessionResponse
		cr.json(t, &sr)
		sessionID = sr.SessionID
		if sr.TaskID != "t-one" || sr.TTLSeconds <= 0 {
			t.Fatalf("session response: %+v", sr)
		}
		ar := e.do("GET", "/api/sessions/active", nil, alice)
		e.expect(ar, 200, "active session")
		if want := []string{"createdAt", "expiresAt", "sessionId", "taskId", "ttlSeconds"}; strings.Join(keysOf(ar.body), ",") != strings.Join(want, ",") {
			t.Fatalf("SessionResponse keys: %s", ar.body)
		}
		e.expect(e.do("GET", "/api/sessions/"+sessionID, nil, alice), 200, "own session")

		// Another user must not see or touch it.
		e.expect(e.do("GET", "/api/sessions/"+sessionID, nil, bob), 404, "foreign get")
		e.expect(e.do("POST", "/api/check/"+sessionID, nil, bob), 404, "foreign check")
		e.expect(e.do("POST", "/api/sessions/"+sessionID+"/recreate", nil, bob), 404, "foreign recreate")
		e.expect(e.do("DELETE", "/api/sessions/"+sessionID, nil, bob), 404, "foreign delete")
		e.expect(e.do("GET", "/api/sessions/does-not-exist", nil, alice), 404, "missing session")
	})

	t.Run("checks and progress", func(t *testing.T) {
		e.lab.setResult(0, 1)
		e.expect(e.do("POST", "/api/check/"+sessionID, nil, alice), 200, "failing check")
		var tasks []models.TaskSummary
		e.do("GET", "/api/tasks", nil, alice).json(t, &tasks)
		if !tasks[0].InProgress || tasks[0].Completed || tasks[1].Unlocked {
			t.Fatalf("after failed check: %+v", tasks[0])
		}

		e.lab.setResult(1, 1)
		r := e.do("POST", "/api/check/"+sessionID, nil, alice)
		e.expect(r, 200, "passing check")
		var cr models.CheckResponse
		r.json(t, &cr)
		if !cr.Passed {
			t.Fatalf("check: %s", r.body)
		}
		e.do("GET", "/api/tasks", nil, alice).json(t, &tasks)
		if !tasks[0].Completed || tasks[0].InProgress || !tasks[1].Unlocked || tasks[2].Unlocked {
			t.Fatalf("after pass: %+v", tasks)
		}

		// A later failing check must not undo the pass.
		e.lab.setResult(0, 1)
		e.expect(e.do("POST", "/api/check/"+sessionID, nil, alice), 200, "regression check")

		var rows []models.Progress
		database.DB.Where("user_id = ?", aliceUser.ID).Find(&rows)
		if len(rows) != 1 || rows[0].Attempts != 3 || !rows[0].Passed || rows[0].CompletedAt == nil || rows[0].Score != 1 {
			t.Fatalf("progress rows: %+v", rows)
		}
		var completes int64
		database.DB.Model(&models.ActivityLog{}).Where("user_id = ? AND action = ?", aliceUser.ID, "task_complete").Count(&completes)
		if completes != 1 {
			t.Fatalf("task_complete logged %d times", completes)
		}

		// A closed 10-minute session counts towards time in lab.
		start := time.Now().UTC().Add(-2 * time.Hour)
		end := start.Add(10 * time.Minute)
		database.DB.Create(&models.Session{ID: "old-session", UserID: aliceUser.ID, TaskID: "t-one", ContainerID: "x",
			Status: models.SessionStopped, CreatedAt: start, ExpiresAt: start.Add(time.Hour), EndedAt: &end})

		pr := e.do("GET", "/api/progress", nil, alice)
		e.expect(pr, 200, "progress")
		var p models.ProgressResponse
		pr.json(t, &p)
		if p.TotalScore != 100 || p.TotalPossible != 600 || p.Completed != 1 || p.TotalTasks != 3 || p.Percentage != 17 {
			t.Fatalf("progress totals: %+v", p)
		}
		if p.StreakDays != 1 || p.TimeInLabSeconds < 600 || p.TimeInLabSeconds > 700 {
			t.Fatalf("streak/time: streak=%d time=%d", p.StreakDays, p.TimeInLabSeconds)
		}
		if len(p.Progress) != 1 || p.Progress[0].Attempts != 3 {
			t.Fatalf("progress items: %+v", p.Progress)
		}
		if len(p.Skills) != 2 || p.Skills[0] != (models.SkillProgress{Category: "web", Percent: 25}) {
			t.Fatalf("skills: %+v", p.Skills)
		}
		ach := achievementMap(p.Achievements)
		if !ach["first-blood"].Earned || !ach["speedrun"].Earned || ach["perfectionist"].Earned || ach["all-done"].Earned {
			t.Fatalf("achievements: %+v", p.Achievements)
		}
		want := []string{"achievements", "completed", "percentage", "progress", "skills", "streakDays", "timeInLabSeconds", "totalPossible", "totalScore", "totalTasks"}
		if strings.Join(keysOf(pr.body), ",") != strings.Join(want, ",") {
			t.Fatalf("ProgressResponse keys: %v", keysOf(pr.body))
		}
	})

	t.Run("leaderboard", func(t *testing.T) {
		r := e.do("GET", "/api/leaderboard?period=week&limit=1", nil, bob)
		e.expect(r, 200, "leaderboard")
		var lb models.LeaderboardResponse
		r.json(t, &lb)
		if len(lb.Entries) != 1 || lb.Entries[0].Username != "alice" || lb.Entries[0].Score != 100 || lb.Entries[0].Rank != 1 {
			t.Fatalf("entries: %+v", lb.Entries)
		}
		if lb.Me == nil || lb.Me.Username != "bob" || lb.Me.Rank != 2 || lb.Me.Score != 0 {
			t.Fatalf("me: %+v", lb.Me)
		}
		e.expect(e.do("GET", "/api/leaderboard?period=year", nil, bob), 400, "bad period")
		e.expect(e.do("GET", "/api/leaderboard?limit=0", nil, bob), 400, "limit 0")
		e.expect(e.do("GET", "/api/leaderboard?limit=101", nil, bob), 400, "limit 101")
		e.expect(e.do("GET", "/api/leaderboard", nil, nil), 401, "leaderboard needs auth")
	})

	t.Run("session lifecycle", func(t *testing.T) {
		rr := e.do("POST", "/api/sessions/"+sessionID+"/recreate", nil, alice)
		e.expect(rr, 201, "recreate")
		var sr models.SessionResponse
		rr.json(t, &sr)
		e.expect(e.do("DELETE", "/api/sessions/"+sr.SessionID, nil, alice), 204, "delete session")
		e.expect(e.do("GET", "/api/sessions/active", nil, alice), 204, "no active after delete")
	})

	t.Run("admin", func(t *testing.T) {
		e.expect(e.do("GET", "/api/admin/stats", nil, alice), 403, "student on admin route")

		r := e.do("GET", "/api/admin/stats", nil, carol)
		e.expect(r, 200, "stats")
		var st models.StatsResponse
		r.json(t, &st)
		if st.TotalUsers != 3 || st.TotalAttempts != 3 || len(st.Sessions24h) != 24 || st.Sessions24h[23] < 3 ||
			len(st.TaskCompletion) != 3 || st.TaskCompletion[0].Passed != 1 || st.TaskCompletion[0].Attempts != 3 || st.ActiveUsers < 1 {
			t.Fatalf("stats: %s", r.body)
		}

		r = e.do("GET", "/api/admin/users", nil, carol)
		e.expect(r, 200, "users")
		var users []models.AdminUser
		r.json(t, &users)
		if len(users) != 3 || users[0].Username != "alice" || users[0].Score != 100 || users[0].Completed != 1 || users[0].LastActive == nil {
			t.Fatalf("admin users: %s", r.body)
		}

		self := fmt.Sprint(carolUser.ID)
		e.expect(e.do("PUT", "/api/admin/users/"+self, map[string]any{"role": "student"}, carol), 400, "self demote")
		e.expect(e.do("PUT", "/api/admin/users/"+self, map[string]any{"active": false}, carol), 400, "self deactivate")
		e.expect(e.do("DELETE", "/api/admin/users/"+self, nil, carol), 400, "self delete")
		e.expect(e.do("PUT", "/api/admin/users/"+fmt.Sprint(aliceUser.ID), map[string]any{"role": "root"}, carol), 400, "bad role")
		e.expect(e.do("PUT", "/api/admin/users/999999", map[string]any{"active": false}, carol), 404, "missing user")
		e.expect(e.do("PUT", "/api/admin/users/abc", map[string]any{"active": false}, carol), 400, "bad id")

		var bobUser models.User
		database.DB.Where("username = ?", "bob").Take(&bobUser)
		r = e.do("PUT", "/api/admin/users/"+fmt.Sprint(bobUser.ID), map[string]any{"active": false}, carol)
		e.expect(r, 200, "deactivate bob")
		var u models.UserResponse
		r.json(t, &u)
		if u.Active {
			t.Fatal("bob still active")
		}
		e.expect(e.do("GET", "/api/me", nil, bob), 403, "deactivated user")

		sr := e.do("GET", "/api/admin/sessions", nil, carol)
		e.expect(sr, 200, "admin sessions")
		var sessions []models.AdminSession
		sr.json(t, &sessions)
		if len(sessions) != 1 || sessions[0].Username != "carol" {
			t.Fatalf("admin sessions: %s", sr.body)
		}
		e.expect(e.do("POST", "/api/admin/sessions/nope/terminate", nil, carol), 404, "terminate missing")
		e.expect(e.do("POST", "/api/admin/sessions/"+sessions[0].ID+"/terminate", nil, carol), 204, "terminate")

		e.expect(e.do("GET", "/api/admin/activity?limit=501", nil, carol), 400, "activity limit")
		ar := e.do("GET", "/api/admin/activity?limit=5", nil, carol)
		e.expect(ar, 200, "activity")
		var entries []map[string]any
		ar.json(t, &entries)
		if len(entries) != 5 || entries[0]["level"] == "" {
			t.Fatalf("activity: %s", ar.body)
		}
		var warn int64
		database.DB.Model(&models.ActivityLog{}).Where("action = ? AND level = ?", "login_failed", "WARN").Count(&warn)
		if warn < 2 {
			t.Fatalf("login failures not logged as WARN: %d", warn)
		}

		rs := e.do("GET", "/api/admin/resources", nil, carol)
		e.expect(rs, 200, "resources")
		if !strings.Contains(string(rs.body), `"sessions":[]`) || !strings.Contains(string(rs.body), `"containers_limit":20`) {
			t.Fatalf("resources: %s", rs.body)
		}
		tr := e.do("POST", "/api/admin/tasks/reload", nil, carol)
		e.expect(tr, 200, "reload")
		if string(tr.body) != `{"count":3}` {
			t.Fatalf("reload: %s", tr.body)
		}

		e.expect(e.do("DELETE", "/api/admin/users/"+fmt.Sprint(bobUser.ID), nil, carol), 204, "delete bob")
		e.expect(e.do("DELETE", "/api/admin/users/"+fmt.Sprint(bobUser.ID), nil, carol), 404, "delete bob again")
		e.do("GET", "/api/admin/users", nil, carol).json(t, &users)
		if len(users) != 2 {
			t.Fatalf("users after delete: %d", len(users))
		}
		// The username is free again after a hard delete.
		e.register("bob", pw)
	})

	t.Run("reset progress and logout", func(t *testing.T) {
		e.expect(e.do("DELETE", "/api/progress", nil, alice), 204, "reset")
		var p models.ProgressResponse
		e.do("GET", "/api/progress", nil, alice).json(t, &p)
		if p.Completed != 0 || p.TotalScore != 0 || len(p.Progress) != 0 || p.Achievements[0].Earned {
			t.Fatalf("after reset: %+v", p)
		}
		lo := e.do("POST", "/api/auth/logout", nil, alice)
		e.expect(lo, 204, "logout")
		cleared := false
		for _, c := range lo.cookies {
			if c.Name == auth.CookieName && c.MaxAge < 0 {
				cleared = true
			}
		}
		if !cleared {
			t.Fatal("logout must clear the cookie")
		}
	})

	t.Run("guards and health", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(`{}`))
		req.Header.Set("Origin", "https://evil.example")
		w := httptest.NewRecorder()
		e.router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("foreign origin: %d", w.Code)
		}

		big := `{"github":"` + strings.Repeat("a", MaxBodyBytes+10) + `"}`
		e.expect(e.do("PUT", "/api/me", big, carol), 413, "body too large")

		hz := e.do("GET", "/healthz", nil, nil)
		e.expect(hz, 200, "healthz")
		if !strings.Contains(string(hz.body), `"version":"test-version"`) {
			t.Fatalf("healthz must report the build version: %s", hz.body)
		}
		SetReadyChecks([]ReadyCheck{{Name: "database", Check: pingDatabase}})
		e.expect(e.do("GET", "/readyz", nil, nil), 200, "readyz ok")
		SetReadyChecks([]ReadyCheck{{Name: "docker", Check: func(context.Context) error { return errors.New("down") }}})
		r := e.do("GET", "/readyz", nil, nil)
		e.expect(r, 503, "readyz down")
		if strings.Contains(string(r.body), "down\"}") && strings.Contains(string(r.body), "error") {
			t.Fatalf("readyz must not leak errors: %s", r.body)
		}
	})
}
