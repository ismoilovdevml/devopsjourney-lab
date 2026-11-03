package session

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/devops-lab/backend/internal/docker"
	"gopkg.in/yaml.v3"
)

// TaskRuntime holds the container-runtime fields of a task.yaml. They are read
// here (not by the task loader) because they only matter to the session
// manager and are never sent to clients:
//
//	capabilities: [NET_ADMIN]   # extra Linux capabilities, see docker.ExtraCapabilityAllowlist
//	network: false              # true = attach to the bridge network (default: no network)
type TaskRuntime struct {
	Capabilities []string `yaml:"capabilities"`
	Network      bool     `yaml:"network"`
}

var taskIDPattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

// loadTaskRuntime reads tasksDir/<taskID>/task.yaml and validates the
// requested capabilities against the allowlist.
func loadTaskRuntime(tasksDir, taskID string) (*TaskRuntime, error) {
	if !taskIDPattern.MatchString(taskID) {
		return nil, fmt.Errorf("invalid task id %q", taskID)
	}
	data, err := os.ReadFile(filepath.Join(tasksDir, taskID, "task.yaml"))
	if err != nil {
		return nil, fmt.Errorf("failed to read task runtime config: %w", err)
	}
	return parseTaskRuntime(data)
}

func parseTaskRuntime(data []byte) (*TaskRuntime, error) {
	var rt TaskRuntime
	if err := yaml.Unmarshal(data, &rt); err != nil {
		return nil, fmt.Errorf("failed to parse task runtime config: %w", err)
	}
	if _, err := docker.BuildCapAdd(rt.Capabilities); err != nil {
		return nil, err
	}
	return &rt, nil
}
