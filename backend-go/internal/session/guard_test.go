package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
)

func TestDiskViolations(t *testing.T) {
	const limit = 2 << 30
	active := []models.Session{
		{ID: "s-small", ContainerID: "c1"},
		{ID: "s-big", ContainerID: "c2"},
		{ID: "s-exact", ContainerID: "c3"},
		{ID: "s-nocontainer", ContainerID: "c4"},
		{ID: "s-mislabel", ContainerID: "c5"},
	}
	sizes := []docker.LabContainer{
		{ID: "c1", SessionID: "s-small", SizeRw: 10 << 20},
		{ID: "c2", SessionID: "s-big", SizeRw: limit + 1},
		{ID: "c3", SessionID: "s-exact", SizeRw: limit},
		{ID: "c5", SessionID: "someone-else", SizeRw: limit * 4},
		{ID: "c9", SessionID: "orphan", SizeRw: limit * 4},
	}
	got := diskViolations(active, sizes, limit)
	if len(got) != 1 || got[0].session.ID != "s-big" || got[0].sizeRw != limit+1 {
		t.Fatalf("violations = %+v, want only s-big", got)
	}
	if v := diskViolations(active, sizes, 0); v != nil {
		t.Fatalf("limit 0 disables the check, got %+v", v)
	}
}

func TestIdleSessions(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	start := now.Add(-2 * time.Hour) // backend has been up for 2h
	timeout := 15 * time.Minute

	active := []models.Session{
		{ID: "attached", CreatedAt: now.Add(-time.Hour)},
		{ID: "never-attached-old", CreatedAt: now.Add(-20 * time.Minute)},
		{ID: "never-attached-new", CreatedAt: now.Add(-5 * time.Minute)},
		{ID: "left-recently", CreatedAt: now.Add(-time.Hour)},
		{ID: "left-long-ago", CreatedAt: now.Add(-time.Hour)},
	}
	activity := func(id string) (bool, time.Time) {
		switch id {
		case "attached":
			return true, now.Add(-time.Hour) // attached wins even if quiet
		case "left-recently":
			return false, now.Add(-10 * time.Minute)
		case "left-long-ago":
			return false, now.Add(-16 * time.Minute)
		}
		return false, time.Time{}
	}
	got := map[string]bool{}
	for _, s := range idleSessions(active, activity, now, start, timeout) {
		got[s.ID] = true
	}
	want := map[string]bool{"never-attached-old": true, "left-long-ago": true}
	if len(got) != len(want) {
		t.Fatalf("idle = %v, want %v", got, want)
	}
	for id := range want {
		if !got[id] {
			t.Fatalf("idle = %v, want %v", got, want)
		}
	}

	// Right after a restart nothing is known about terminals: the process
	// start time protects every session for one full timeout.
	fresh := now.Add(-time.Minute)
	if s := idleSessions(active, activity, now, fresh, timeout); len(s) != 0 {
		t.Fatalf("sessions reclaimed right after restart: %d", len(s))
	}
}

func TestTaskAvailable(t *testing.T) {
	plain := &TaskRuntime{}
	netAdmin := &TaskRuntime{Capabilities: []string{"cap_net_admin"}}

	if ok, _ := taskAvailable(plain, false); !ok {
		t.Error("task without extra caps must always be available")
	}
	if ok, _ := taskAvailable(netAdmin, true); !ok {
		t.Error("NET_ADMIN task must be available when allowed")
	}
	ok, reason := taskAvailable(netAdmin, false)
	if ok || reason == "" {
		t.Errorf("NET_ADMIN task must be unavailable with a reason, got ok=%v reason=%q", ok, reason)
	}
}

func TestManagerTaskAvailabilityFromTaskFiles(t *testing.T) {
	dir := t.TempDir()
	write := func(id, body string) {
		if err := os.MkdirAll(filepath.Join(dir, id), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, id, "task.yaml"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("fw", "id: fw\ncapabilities: [NET_ADMIN]\n")
	write("web", "id: web\n")

	m := NewManager(&configs.DockerConfig{TasksDir: dir, AllowNetAdmin: false})
	if ok, _ := m.TaskAvailability("fw"); ok {
		t.Error("fw must be unavailable without NET_ADMIN")
	}
	if ok, _ := m.TaskAvailability("web"); !ok {
		t.Error("web must be available")
	}
	if ok, _ := m.TaskAvailability("missing"); ok {
		t.Error("unreadable task config must be reported unavailable")
	}
	m = NewManager(&configs.DockerConfig{TasksDir: dir, AllowNetAdmin: true})
	if ok, _ := m.TaskAvailability("fw"); !ok {
		t.Error("fw must be available with NET_ADMIN allowed")
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{512: "512 B", 2048: "2.0 KiB", 2 << 30: "2.0 GiB", 3 << 20: "3.0 MiB"}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", in, got, want)
		}
	}
}
