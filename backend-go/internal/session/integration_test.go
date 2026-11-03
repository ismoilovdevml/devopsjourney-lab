package session

// Integration test: needs a Docker daemon and the lab image.
//
//	docker build -t devopslab-lab:latest lab-image
//	LAB_INTEGRATION=1 go test ./internal/session/... -run Integration -v
//
// For every task it provisions a container through the manager's real code
// path (same HostConfig, labels, setup), checks that the task is NOT solved
// initially, applies e2e/fixtures/solutions/<task>.sh, and expects every check
// to pass (twice, to prove the solution is idempotent).

import (
	"context"
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

func integrationManager(t *testing.T) (*Manager, string) {
	t.Helper()
	if os.Getenv("LAB_INTEGRATION") != "1" {
		t.Skip("set LAB_INTEGRATION=1 to run (needs Docker and the devopslab-lab image)")
	}
	if docker.Client == nil {
		if err := docker.InitClient(os.Getenv("DOCKER_SOCKET")); err != nil {
			t.Fatalf("docker: %v", err)
		}
	}
	tasksDir, _ := filepath.Abs(filepath.Join("..", "..", "tasks"))
	if err := task.Init(tasksDir); err != nil {
		t.Fatal(err)
	}
	image := os.Getenv("LAB_IMAGE")
	if image == "" {
		image = DefaultLabImage
	}
	return NewManager(&configs.DockerConfig{
		DefaultImage:     image,
		TasksDir:         tasksDir,
		SessionTTL:       600,
		MemLimit:         512 << 20,
		CPUNano:          500_000_000,
		PidsLimit:        256,
		MaxSessionsTotal: 100,
	}), tasksDir
}

func provisionForTest(t *testing.T, m *Manager, taskObj *models.Task) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	sid, _ := newSessionID()
	start := time.Now()
	id, err := m.provisionContainer(ctx, "itest-"+sid, 0, taskObj)
	if err != nil {
		t.Fatalf("provision: %v", err)
	}
	t.Logf("provisioned %s (create+start+setup) in %s", shortID(id), time.Since(start).Round(time.Millisecond))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := docker.RemoveContainer(ctx, id); err != nil {
			t.Errorf("cleanup: %v", err)
		}
	})
	return id
}

func TestIntegrationTasks(t *testing.T) {
	m, _ := integrationManager(t)
	solutionsDir := filepath.Join("..", "..", "..", "e2e", "fixtures", "solutions")

	for _, tk := range task.GetTasks() {
		tk := tk
		t.Run(tk.ID, func(t *testing.T) {
			ctx := context.Background()
			id := provisionForTest(t, m, &tk)

			info, err := docker.InspectContainer(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			rt, _ := loadTaskRuntime(m.cfg.TasksDir, tk.ID)
			t.Logf("network=%s caps=%s", info.HostConfig.NetworkMode, strings.Join(info.HostConfig.CapAdd, ","))
			if !rt.Network && info.HostConfig.NetworkMode != "none" {
				t.Errorf("NetworkMode = %s, want none", info.HostConfig.NetworkMode)
			}
			if info.HostConfig.Privileged || !strings.Contains(strings.Join(info.HostConfig.SecurityOpt, ","), "no-new-privileges") {
				t.Error("container is not hardened")
			}
			if info.Config.Labels[docker.LabelTask] != tk.ID {
				t.Errorf("labels = %v", info.Config.Labels)
			}

			before := buildCheckResponse(runChecks(ctx, id, tk.Checks))
			t.Logf("before solution: %d/%d", before.Score, before.Total)
			if before.Passed {
				t.Fatal("task is already solved before the student did anything")
			}

			script, err := os.ReadFile(filepath.Join(solutionsDir, tk.ID+".sh"))
			if err != nil {
				t.Fatal(err)
			}
			for run := 1; run <= 2; run++ {
				res := docker.ExecCommand(ctx, id, string(script), 60*time.Second)
				if res.Error != nil || res.ExitCode != 0 {
					t.Fatalf("solution run %d: exit=%d err=%v\n%s", run, res.ExitCode, res.Error, res.Output)
				}
				after := buildCheckResponse(runChecks(ctx, id, tk.Checks))
				for _, r := range after.Results {
					status := "PASS"
					if !r.OK {
						status = "FAIL"
					}
					t.Logf("  run %d [%s] %s", run, status, r.Name)
					if !r.OK {
						t.Logf("        output: %s", r.Output)
					}
				}
				if !after.Passed {
					t.Fatalf("solution run %d: %d/%d checks passed", run, after.Score, after.Total)
				}
				t.Logf("after solution run %d: %d/%d", run, after.Score, after.Total)
			}
		})
	}
}

// TestIntegrationHardening proves the sandbox properties from inside a lab
// container: no network, no mount, no firewall changes without NET_ADMIN,
// bounded output and timeouts.
func TestIntegrationHardening(t *testing.T) {
	m, _ := integrationManager(t)
	tk, err := task.GetTask("nginx-health")
	if err != nil {
		t.Fatal(err)
	}
	id := provisionForTest(t, m, tk)
	ctx := context.Background()

	mustFail := map[string]string{
		"mount a filesystem":         "mount -t tmpfs none /mnt",
		"change iptables":            "iptables -A INPUT -j DROP",
		"create a device node":       "mknod /tmp/sda b 8 0",
		"reach the internet":         "curl -sS -m 3 http://1.1.1.1/",
		"create a network interface": "ip link add dummy0 type dummy",
		"change the hostname":        "hostname evil",
	}
	for name, cmd := range mustFail {
		res := docker.ExecCommand(ctx, id, cmd, 8*time.Second)
		if res.Error == nil && res.ExitCode == 0 {
			t.Errorf("%s: expected failure, got exit 0\n%s", name, res.Output)
		} else {
			t.Logf("blocked as expected: %s (exit %d)", name, res.ExitCode)
		}
	}

	mustPass := map[string]string{
		"hostname is lab":               "test \"$(hostname)\" = lab",
		"only loopback is up, no route": "test \"$(ip -o link show up | awk -F': ' '{print $2}')\" = lo && test -z \"$(ip route show default)\"",
		"no-new-privileges is set":      "grep -q '^NoNewPrivs:[[:space:]]*1' /proc/self/status",
		"/tmp is a tmpfs":               "findmnt -n -o FSTYPE /tmp | grep -qx tmpfs",
		"nofile ulimit is 1024":         "test \"$(ulimit -n)\" = 1024",
		"PID 1 is the init process":     "ps -o comm= -p 1 | grep -Eq 'docker-init|tini'",
		"PIDs limit is 256":             "test \"$(cat /sys/fs/cgroup/pids.max)\" = 256",
		"memory limit is 512MiB":        "test \"$(cat /sys/fs/cgroup/memory.max)\" = 536870912",
	}
	for name, cmd := range mustPass {
		res := docker.ExecCommand(ctx, id, cmd, 5*time.Second)
		if res.Error != nil || res.ExitCode != 0 {
			t.Errorf("%s: exit=%d err=%v %s", name, res.ExitCode, res.Error, res.Output)
		} else {
			t.Logf("ok: %s", name)
		}
	}

	// A hanging check is killed after the timeout and reported as such.
	start := time.Now()
	res := runChecks(ctx, id, []models.Check{{Name: "hangs", Cmd: "sleep 60"}})
	if res[0].OK || !strings.Contains(res[0].Output, "timed out") || time.Since(start) > 20*time.Second {
		t.Errorf("timeout handling: %+v after %s", res[0], time.Since(start))
	} else {
		t.Logf("hanging check stopped after %s: %q", time.Since(start).Round(time.Millisecond), res[0].Output)
	}
	if out := docker.ExecCommand(ctx, id, "pgrep -fx 'sleep 60'", 5*time.Second); out.ExitCode == 0 {
		t.Errorf("timed-out check left processes behind: %s", out.Output)
	}

	// Output is truncated to 2KB.
	res = runChecks(ctx, id, []models.Check{{Name: "noisy", Cmd: "yes | head -c 100000; exit 1"}})
	if len(res[0].Output) > maxCheckOutput {
		t.Errorf("output not truncated: %d bytes", len(res[0].Output))
	} else {
		t.Logf("noisy check output truncated to %d bytes", len(res[0].Output))
	}

	// The container is discoverable by label (used by orphan cleanup and capacity).
	list, err := docker.ListLabContainers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range list {
		if c.ID == id && c.TaskID == "nginx-health" && strings.HasPrefix(c.SessionID, "itest-") {
			found = true
		}
	}
	if !found {
		t.Error("container not found by devopslab.session label")
	}
}
