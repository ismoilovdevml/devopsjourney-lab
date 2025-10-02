package task

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/devops-lab/backend/internal/models"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

var tasks []models.Task
var tasksDir string

// Init initializes the task loader
func Init(dir string) error {
	tasksDir = dir
	return LoadTasks()
}

// LoadTasks loads all tasks from the tasks directory
func LoadTasks() error {
	if _, err := os.Stat(tasksDir); os.IsNotExist(err) {
		return fmt.Errorf("tasks directory does not exist: %s", tasksDir)
	}

	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return fmt.Errorf("failed to read tasks directory: %w", err)
	}

	tasks = []models.Task{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := entry.Name()
		taskFile := filepath.Join(tasksDir, taskID, "task.yaml")

		if _, err := os.Stat(taskFile); os.IsNotExist(err) {
			log.Warn().Str("task_id", taskID).Msg("Task YAML file not found, skipping")
			continue
		}

		task, err := loadTask(taskFile, taskID)
		if err != nil {
			log.Error().Err(err).Str("task_id", taskID).Msg("Failed to load task")
			continue
		}

		tasks = append(tasks, *task)
	}

	// Sort by order
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Order < tasks[j].Order
	})

	log.Info().Int("count", len(tasks)).Msg("Tasks loaded successfully")
	return nil
}

func loadTask(filename, taskID string) (*models.Task, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var task models.Task
	if err := yaml.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set ID if not specified
	if task.ID == "" {
		task.ID = taskID
	}

	// Load description from README.md if exists
	readmePath := filepath.Join(filepath.Dir(filename), "README.md")
	if readme, err := os.ReadFile(readmePath); err == nil {
		task.Description = string(readme)
	}

	return &task, nil
}

// GetTasks returns all tasks
func GetTasks() []models.Task {
	return tasks
}

// GetTask returns a task by ID
func GetTask(id string) (*models.Task, error) {
	for _, task := range tasks {
		if task.ID == id {
			return &task, nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", id)
}

// ReloadTasks reloads all tasks
func ReloadTasks() error {
	return LoadTasks()
}
