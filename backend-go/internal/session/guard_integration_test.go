package session

// Integration test for the resource guards (disk quota, idle reclaim). Needs
// Docker, the lab image and a throwaway Postgres:
//
//	LAB_INTEGRATION=1 LAB_PG_PORT=55450 go test ./internal/session/... -run IntegrationResourceGuards -v
//
// Unlike TestIntegrationLifecycle it never runs the orphan sweeper, so it only
// touches sessions it created itself and is safe next to a running stack.

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
)

func TestIntegrationResourceGuards(t *testing.T) {
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
	hog, quiet, busy := newUser("hog"), newUser("quiet"), newUser("busy")

	start := func(user uint) *models.SessionResponse {
		s, err := m.CreateSession(ctx, user, "nginx-health")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = m.DeleteSession(context.Background(), s.SessionID) })
		return s
	}
	hogSess, quietSess, busySess := start(hog), start(quiet), start(busy)

	row := func(id string) models.Session {
		var s models.Session
		database.DB.First(&s, "id = ?", id)
		return s
	}

	// Disk: the hog writes 64 MiB into its writable layer; limit is 32 MiB.
	if res := docker.ExecCommand(ctx, row(hogSess.SessionID).ContainerID,
		"head -c 67108864 /dev/zero > /root/big.bin && sync", 30*time.Second); res.ExitCode != 0 {
		t.Fatalf("write: %d %s", res.ExitCode, res.Output)
	}
	m.cfg.DiskMaxBytes = 32 << 20
	m.cfg.IdleTimeout = time.Hour // idle reclaim must not interfere with the disk part
	m.SetActivitySource(func(string) (bool, time.Time) { return false, time.Time{} })
	if err := m.EnforceResourceGuards(ctx); err != nil {
		t.Fatal(err)
	}
	h := row(hogSess.SessionID)
	if h.Status != StatusExpired || h.EndReason != ReasonDiskQuota || h.EndedAt == nil {
		t.Fatalf("hog session = status %q reason %q", h.Status, h.EndReason)
	}
	if _, err := docker.InspectContainer(ctx, h.ContainerID); err == nil {
		t.Error("hog container still exists")
	}
	var entry models.ActivityLog
	if err := database.DB.Where("user_id = ? AND action = ?", hog, "session_disk_quota").First(&entry).Error; err != nil || entry.Level != models.LevelWarn {
		t.Errorf("activity log entry = %+v, %v", entry, err)
	} else {
		t.Logf("disk quota enforced: %q", entry.Details)
	}
	for _, id := range []string{quietSess.SessionID, busySess.SessionID} {
		if s := row(id); s.Status != StatusActive {
			t.Fatalf("small session %s was stopped: %q %q", id, s.Status, s.EndReason)
		}
	}

	// Idle: "busy" has a terminal attached, "quiet" never had one.
	m.cfg.IdleTimeout = time.Minute
	m.startedAt = time.Now().Add(-time.Hour)
	database.DB.Model(&models.Session{}).Where("id IN ?", []string{quietSess.SessionID, busySess.SessionID}).
		Update("created_at", time.Now().Add(-10*time.Minute))
	m.SetActivitySource(func(id string) (bool, time.Time) { return id == busySess.SessionID, time.Time{} })
	if err := m.EnforceResourceGuards(ctx); err != nil {
		t.Fatal(err)
	}
	if q := row(quietSess.SessionID); q.Status != StatusExpired || q.EndReason != ReasonIdle {
		t.Fatalf("quiet session = %q %q", q.Status, q.EndReason)
	}
	if b := row(busySess.SessionID); b.Status != StatusActive {
		t.Fatalf("busy session must survive: %q %q", b.Status, b.EndReason)
	}
	t.Log("idle session reclaimed, attached session kept")
	_ = m.Shutdown(5 * time.Second)
}
