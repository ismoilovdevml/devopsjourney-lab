package task

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/devops-lab/backend/internal/models"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// DefaultCategory is used when task.yaml has no category.
const DefaultCategory = "other"

// IDPattern is the allowed shape of a task id (also used to validate path params).
var IDPattern = regexp.MustCompile(`^[a-z0-9-]{1,64}$`)

var (
	mu       sync.RWMutex
	tasks    []models.Task
	tasksDir string
)

// Init initializes the task loader.
func Init(dir string) error {
	mu.Lock()
	tasksDir = dir
	mu.Unlock()
	return LoadTasks()
}

// LoadTasks loads all tasks from the tasks directory. On failure the previously
// loaded tasks are kept.
func LoadTasks() error {
	mu.RLock()
	dir := tasksDir
	mu.RUnlock()

	loaded, err := LoadDir(dir)
	if err != nil {
		return err
	}

	mu.Lock()
	tasks = loaded
	mu.Unlock()

	log.Info().Int("count", len(loaded)).Msg("Tasks loaded successfully")
	return nil
}

// LoadDir parses every <dir>/<id>/task.yaml and returns the tasks sorted by order.
func LoadDir(dir string) ([]models.Task, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("tasks directory not accessible: %s: %w", dir, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks directory: %w", err)
	}

	loaded := []models.Task{}
	seen := map[string]bool{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskFile := filepath.Join(dir, entry.Name(), "task.yaml")
		if _, err := os.Stat(taskFile); err != nil {
			log.Warn().Str("task_dir", entry.Name()).Msg("task.yaml not found, skipping")
			continue
		}

		t, err := loadTask(taskFile, entry.Name())
		if err != nil {
			log.Error().Err(err).Str("task_dir", entry.Name()).Msg("Failed to load task")
			continue
		}
		if seen[t.ID] {
			log.Error().Str("task_id", t.ID).Msg("Duplicate task id, skipping")
			continue
		}
		seen[t.ID] = true
		loaded = append(loaded, *t)
	}

	sort.SliceStable(loaded, func(i, j int) bool {
		if loaded[i].Order != loaded[j].Order {
			return loaded[i].Order < loaded[j].Order
		}
		return loaded[i].ID < loaded[j].ID
	})
	return loaded, nil
}

func loadTask(filename, dirName string) (*models.Task, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var t models.Task
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if t.ID == "" {
		t.ID = dirName
	}
	if !IDPattern.MatchString(t.ID) {
		return nil, fmt.Errorf("invalid task id %q", t.ID)
	}

	if readme, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "README.md")); err == nil {
		t.Readme = string(readme)
	}

	normalize(&t)
	return &t, nil
}

// normalize fills defaults: difficulty in 1..3, points = difficulty*100,
// category, description from the README, non-nil hints.
func normalize(t *models.Task) {
	if t.Difficulty < 1 {
		t.Difficulty = 1
	} else if t.Difficulty > 3 {
		t.Difficulty = 3
	}
	if t.Points <= 0 {
		t.Points = t.Difficulty * 100
	}
	t.Category = strings.ToLower(strings.TrimSpace(t.Category))
	if t.Category == "" {
		t.Category = DefaultCategory
	}
	if strings.TrimSpace(t.Description) == "" {
		t.Description = summarize(t.Readme, 200)
	}
	if t.Hints == nil {
		t.Hints = []string{}
	}
}

// summarize returns the first plain paragraph line of a markdown document, truncated to max runes.
func summarize(markdown string, max int) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "```") ||
			strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "|") {
			continue
		}
		r := []rune(line)
		if len(r) > max {
			return string(r[:max-1]) + "…"
		}
		return line
	}
	return ""
}

// GetTasks returns a copy of all tasks sorted by order.
func GetTasks() []models.Task {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]models.Task, len(tasks))
	copy(out, tasks)
	return out
}

// GetTask returns a task by ID.
func GetTask(id string) (*models.Task, error) {
	mu.RLock()
	defer mu.RUnlock()
	for i := range tasks {
		if tasks[i].ID == id {
			t := tasks[i]
			return &t, nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", id)
}

// SetTasks replaces the loaded tasks (used by tests).
func SetTasks(ts []models.Task) {
	mu.Lock()
	defer mu.Unlock()
	tasks = append([]models.Task(nil), ts...)
}

// ReloadTasks reloads all tasks.
func ReloadTasks() error {
	return LoadTasks()
}
