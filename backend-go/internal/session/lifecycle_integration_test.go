package session

// DB-backed lifecycle test. Needs Docker, the lab image and a throwaway
// Postgres (Redis is optional and not used here):
//
//	docker run -d --name itest-pg -e POSTGRES_PASSWORD=itest -e POSTGRES_USER=itest \
//	  -e POSTGRES_DB=itest -p 127.0.0.1:55432:5432 postgres:17-alpine
//	LAB_INTEGRATION=1 LAB_PG_PORT=55432 go test ./internal/session/... -run IntegrationLifecycle -v

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
)

func TestIntegrationLifecycle(t *testing.T) {
	m, _ := integrationManager(t)
	port := os.Getenv("LAB_PG_PORT")
	if port == "" {
		t.Skip("set LAB_PG_PORT to a throwaway Postgres (user/password/db = itest)")
	}
	if err := database.Connect(&configs.DatabaseConfig{
		Host: "127.0.0.1", Port: port, User: "itest", Password: "itest", DBName: "itest", SSLMode: "disable",
	}); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	suffix := time.Now().UnixNano()
	newUser := func(name string) uint {
		u := models.User{Username: fmt.Sprintf("%s%d", name, suffix), Email: fmt.Sprintf("%s%d@example.test", name, suffix), Password: "x"}
		if err := database.DB.Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		return u.ID
	}
	alice, bob := newUser("alice"), newUser("bob")

	// Create, then replace: one active session per user.
	first, err := m.CreateSession(ctx, alice, "nginx-health")
	if err != nil {
		t.Fatal(err)
	}
	if first.TaskID != "nginx-health" || first.CreatedAt.IsZero() || first.TTLSeconds <= 0 {
		t.Errorf("response = %+v", first)
	}
	second, err := m.CreateSession(ctx, alice, "user-management")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetSession(ctx, first.SessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("previous session should be gone, got %v", err)
	}
	var old models.Session
	database.DB.First(&old, "id = ?", first.SessionID)
	if old.Status != StatusStopped || old.EndedAt == nil {
		t.Errorf("previous session row = status %q ended_at %v", old.Status, old.EndedAt)
	}
	if _, err := docker.InspectContainer(ctx, old.ContainerID); err == nil {
		t.Error("previous container still exists")
	}
	active, err := m.ActiveSessionForUser(ctx, alice)
	if err != nil || active == nil || active.ID != second.SessionID {
		t.Fatalf("ActiveSessionForUser = %+v, %v", active, err)
	}
	if none, err := m.ActiveSessionForUser(ctx, bob); none != nil || err != nil {
		t.Errorf("bob should have no session: %+v %v", none, err)
	}
	t.Logf("create + replace ok (alice: %s -> %s)", first.SessionID, second.SessionID)

	// Check: concurrent checks on one session are refused.
	m.tryLockCheck(second.SessionID)
	if _, err := m.CheckSolution(ctx, second.SessionID); !errors.Is(err, ErrCheckInProgress) {
		t.Errorf("concurrent check: got %v", err)
	}
	m.unlockCheck(second.SessionID)
	res, err := m.CheckSolution(ctx, second.SessionID)
	if err != nil || res.Passed || res.Total != 6 {
		t.Fatalf("check = %+v, %v", res, err)
	}
	t.Logf("check before solving: %d/%d", res.Score, res.Total)

	// Recreate gives a fresh container for the same task.
	re, err := m.Recreate(ctx, second.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if re.SessionID == second.SessionID || re.TaskID != "user-management" {
		t.Errorf("recreate = %+v", re)
	}
	t.Logf("recreate ok: %s -> %s", second.SessionID, re.SessionID)

	// Resource stats see the running container.
	stats, err := m.ResourceStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range stats.Sessions {
		if s.SessionID == re.SessionID {
			found = s.Username == fmt.Sprintf("alice%d", suffix) && s.MemBytes > 0 && s.MemLimitBytes == 512<<20
			t.Logf("resource stats: %+v (running %d / limit %d)", s, stats.ContainersRunning, stats.ContainersLimit)
		}
	}
	if !found {
		t.Errorf("session missing or incomplete in resource stats: %+v", stats)
	}

	// Capacity: with the cap at the current number of live containers, a new
	// user is refused with ErrCapacity.
	list, _ := docker.ListLabContainers(ctx)
	m.maxSessions = countLive(list)
	if _, err := m.CreateSession(ctx, bob, "nginx-health"); !errors.Is(err, ErrCapacity) {
		t.Errorf("expected ErrCapacity, got %v", err)
	} else {
		t.Logf("capacity enforced at %d live containers", m.maxSessions)
	}
	m.maxSessions = 100

	// Delete.
	if err := m.DeleteSession(ctx, re.SessionID); err != nil {
		t.Fatal(err)
	}
	if err := m.DeleteSession(ctx, re.SessionID); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("second delete: %v", err)
	}

	// Reconciliation: an expired session and an orphaned container (no DB row,
	// as after a crash) are both cleaned up.
	exp, err := m.CreateSession(ctx, bob, "nginx-health")
	if err != nil {
		t.Fatal(err)
	}
	database.DB.Model(&models.Session{}).Where("id = ?", exp.SessionID).Update("expires_at", time.Now().Add(-time.Minute))
	var expRow models.Session
	database.DB.First(&expRow, "id = ?", exp.SessionID)

	orphanID, err := docker.CreateContainer(ctx, &docker.ContainerConfig{
		Image:  DefaultLabImage,
		Labels: map[string]string{docker.LabelSession: "crashed-" + fmt.Sprint(suffix), docker.LabelUser: "0", docker.LabelTask: "nginx-health"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := docker.StartContainer(ctx, orphanID); err != nil {
		t.Fatal(err)
	}
	// A young running container is protected while it may still be provisioning.
	if err := m.CleanupExpiredSessions(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := docker.InspectContainer(ctx, orphanID); err != nil {
		t.Error("young container must survive the sweep (provisioning grace)")
	}
	m.orphanGrace = 0 // pretend the grace period has passed
	if err := m.CleanupExpiredSessions(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := docker.InspectContainer(ctx, orphanID); err == nil {
		t.Error("orphaned container was not removed")
		_ = docker.RemoveContainer(ctx, orphanID)
	}
	database.DB.First(&expRow, "id = ?", exp.SessionID)
	if expRow.Status != StatusExpired || expRow.EndedAt == nil {
		t.Errorf("expired session row = %q %v", expRow.Status, expRow.EndedAt)
	}
	if _, err := docker.InspectContainer(ctx, expRow.ContainerID); err == nil {
		t.Error("expired session container still exists")
	}
	t.Log("reconciliation removed the expired session and the orphaned container")

	_ = m.Shutdown(5 * time.Second)
}
