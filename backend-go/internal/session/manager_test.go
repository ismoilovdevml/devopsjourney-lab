package session

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/task"
)

func TestHasCapacity(t *testing.T) {
	cases := []struct {
		live, max int
		want      bool
	}{
		{0, 20, true},
		{19, 20, true},
		{20, 20, false},
		{25, 20, false},
		{0, 0, false},
	}
	for _, c := range cases {
		if got := hasCapacity(c.live, c.max); got != c.want {
			t.Errorf("hasCapacity(%d, %d) = %v, want %v", c.live, c.max, got, c.want)
		}
	}
}

func TestCountLive(t *testing.T) {
	cs := []docker.LabContainer{
		{State: "running"}, {State: "created"}, {State: "restarting"},
		{State: "exited"}, {State: "dead"}, {State: "removing"},
	}
	if got := countLive(cs); got != 3 {
		t.Errorf("countLive = %d, want 3", got)
	}
}

func TestNewManagerMaxSessions(t *testing.T) {
	if got := NewManager(&configs.DockerConfig{}).MaxSessions(); got != DefaultMaxSessions {
		t.Errorf("default max = %d, want %d", got, DefaultMaxSessions)
	}
	if got := NewManager(&configs.DockerConfig{MaxSessionsTotal: 3}).MaxSessions(); got != 3 {
		t.Errorf("max = %d, want 3", got)
	}
}

func TestErrorsWrap(t *testing.T) {
	if !errors.Is(ErrSessionExpired, ErrSessionNotFound) {
		t.Error("ErrSessionExpired must wrap ErrSessionNotFound")
	}
}

func TestFindOrphans(t *testing.T) {
	now := time.Now()
	old := now.Add(-time.Hour)
	young := now.Add(-10 * time.Second)
	cs := []docker.LabContainer{
		{ID: "live", SessionID: "s1", State: "running", Created: old},
		{ID: "orphan-old", SessionID: "gone", State: "running", Created: old},
		{ID: "provisioning", SessionID: "new", State: "running", Created: young},
		{ID: "exited-young", SessionID: "new2", State: "exited", Created: young},
	}
	got := findOrphans(cs, map[string]bool{"s1": true}, now, 3*time.Minute)
	var ids []string
	for _, c := range got {
		ids = append(ids, c.ID)
	}
	if strings.Join(ids, ",") != "orphan-old,exited-young" {
		t.Errorf("orphans = %v", ids)
	}
}

func TestBuildCheckResponse(t *testing.T) {
	r := buildCheckResponse([]models.CheckResult{{OK: true}, {OK: false}, {OK: true}})
	if r.Passed || r.Score != 2 || r.Total != 3 {
		t.Errorf("got %+v", r)
	}
	if !buildCheckResponse([]models.CheckResult{{OK: true}}).Passed {
		t.Error("all ok should pass")
	}
	if buildCheckResponse(nil).Passed {
		t.Error("no checks must not count as passed")
	}
}

func TestResolveImage(t *testing.T) {
	if resolveImage("", "") != DefaultLabImage {
		t.Error("fallback should be the lab image")
	}
	if resolveImage("", "registry/lab:1") != "registry/lab:1" {
		t.Error("DEFAULT_IMAGE should be used when the task has none")
	}
	if resolveImage("custom:1", "registry/lab:1") != "custom:1" {
		t.Error("task image should win")
	}
}

func TestParseTaskRuntime(t *testing.T) {
	rt, err := parseTaskRuntime([]byte("id: x\ncapabilities: [NET_ADMIN]\nnetwork: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !rt.Network || len(rt.Capabilities) != 1 {
		t.Errorf("got %+v", rt)
	}
	if _, err := parseTaskRuntime([]byte("capabilities: [SYS_ADMIN]\n")); err == nil {
		t.Error("SYS_ADMIN must be rejected")
	}
	rt, err = parseTaskRuntime([]byte("id: x\n"))
	if err != nil || rt.Network || len(rt.Capabilities) != 0 {
		t.Errorf("defaults: %+v, %v", rt, err)
	}
	if _, err := loadTaskRuntime(t.TempDir(), "../etc"); err == nil {
		t.Error("path traversal in task id must be rejected")
	}
}

func TestSessionIDs(t *testing.T) {
	a, err := newSessionID()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := newSessionID()
	if len(a) != 16 || a == b {
		t.Errorf("ids %q %q", a, b)
	}
}

// TestShippedTasks guards the task catalogue: every task must be loadable,
// carry the metadata the UI needs, request only allowlisted capabilities, run
// offline, and ship a reference solution.
func TestShippedTasks(t *testing.T) {
	tasksDir := filepath.Join("..", "..", "tasks")
	tasks, err := task.LoadDir(tasksDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 6 {
		t.Fatalf("expected 6 tasks, got %d", len(tasks))
	}
	categories := map[string]bool{"web": true, "containers": true, "services": true, "users": true, "network": true, "automation": true}
	for _, tk := range tasks {
		t.Run(tk.ID, func(t *testing.T) {
			if strings.HasPrefix(tk.Title, "Test ") {
				t.Errorf("title still has a 'Test N:' prefix: %q", tk.Title)
			}
			if !categories[tk.Category] {
				t.Errorf("category %q", tk.Category)
			}
			if tk.Points != tk.Difficulty*100 {
				t.Errorf("points %d != difficulty*100", tk.Points)
			}
			if len(tk.Hints) != 3 {
				t.Errorf("want 3 hints, got %d", len(tk.Hints))
			}
			if tk.Description == "" || len(tk.Checks) < 3 {
				t.Error("missing description or checks")
			}
			for _, s := range tk.Setup {
				if strings.Contains(s, "apt-get") || strings.Contains(s, "curl http") {
					t.Errorf("setup must run offline: %q", s)
				}
			}
			if _, err := loadTaskRuntime(tasksDir, tk.ID); err != nil {
				t.Error(err)
			}
			sol := filepath.Join("..", "..", "..", "e2e", "fixtures", "solutions", tk.ID+".sh")
			if _, err := os.Stat(sol); err != nil {
				t.Errorf("missing reference solution: %v", err)
			}
		})
	}
	for _, junk := range []string{"*/task.yaml.bak", "*/task.yml"} {
		if m, _ := filepath.Glob(filepath.Join(tasksDir, junk)); len(m) > 0 {
			t.Errorf("junk files: %v", m)
		}
	}
}
