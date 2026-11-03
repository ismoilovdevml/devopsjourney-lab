package api

import (
	"context"
	"net/http"
	"time"

	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/task"
	"github.com/gin-gonic/gin"
)

// taskState is what a user has done on each task.
type taskState struct {
	completed  map[string]bool
	inProgress map[string]bool
}

func emptyTaskState() taskState {
	return taskState{completed: map[string]bool{}, inProgress: map[string]bool{}}
}

// loadTaskState fetches the user's per-task state with two indexed queries.
func loadTaskState(ctx context.Context, userID uint) (taskState, error) {
	st := emptyTaskState()
	if userID == 0 {
		return st, nil
	}

	var rows []struct {
		TaskID   string
		Passed   bool
		Attempts int
	}
	if err := database.DB.WithContext(ctx).Model(&models.Progress{}).
		Select("task_id", "passed", "attempts").Where("user_id = ?", userID).Scan(&rows).Error; err != nil {
		return st, err
	}
	for _, r := range rows {
		if r.Passed {
			st.completed[r.TaskID] = true
		} else if r.Attempts > 0 {
			st.inProgress[r.TaskID] = true
		}
	}

	var active []string
	if err := database.DB.WithContext(ctx).Model(&models.Session{}).
		Where("user_id = ? AND status = ? AND expires_at > ?", userID, models.SessionActive, time.Now().UTC()).
		Distinct().Pluck("task_id", &active).Error; err != nil {
		return st, err
	}
	for _, id := range active {
		if !st.completed[id] {
			st.inProgress[id] = true
		}
	}
	return st, nil
}

// availability maps task ID -> reason it cannot run on this server. Tasks not
// in the map are available.
type availability map[string]string

func (a availability) available(id string) bool {
	_, blocked := a[id]
	return !blocked
}

// loadAvailability asks the lab manager which tasks this server can run.
func loadAvailability(tasks []models.Task) availability {
	out := availability{}
	if sessionManager == nil {
		return out
	}
	for _, t := range tasks {
		if ok, reason := sessionManager.TaskAvailability(t.ID); !ok {
			out[t.ID] = reason
		}
	}
	return out
}

// unlockedSet applies the unlock rule. Unavailable tasks are never unlocked and are skipped
// in the chain: an available task is unlocked if it is the first available task or the
// previous available task is completed. Admins see every available task unlocked.
// tasks must be sorted by order.
func unlockedSet(tasks []models.Task, completed map[string]bool, admin bool, avail availability) map[string]bool {
	out := make(map[string]bool, len(tasks))
	prev := ""
	for _, t := range tasks {
		if !avail.available(t.ID) {
			out[t.ID] = false
			continue
		}
		out[t.ID] = admin || prev == "" || completed[prev]
		prev = t.ID
	}
	return out
}

// buildSummaries converts tasks into client-safe summaries for a user.
func buildSummaries(tasks []models.Task, st taskState, admin bool, avail availability) []models.TaskSummary {
	unlocked := unlockedSet(tasks, st.completed, admin, avail)
	out := make([]models.TaskSummary, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, summarize(t, unlocked[t.ID], st, avail))
	}
	return out
}

func summarize(t models.Task, unlocked bool, st taskState, avail availability) models.TaskSummary {
	return models.TaskSummary{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		Category:    t.Category,
		Difficulty:  t.Difficulty,
		Points:      t.Points,
		Order:       t.Order,
		ChecksCount: len(t.Checks),
		Unlocked:    unlocked,
		Completed:   st.completed[t.ID],
		InProgress:  st.inProgress[t.ID] && !st.completed[t.ID],

		Available:         avail.available(t.ID),
		UnavailableReason: avail[t.ID],
	}
}

// buildDetail converts a task into a TaskDetail (never includes checks[].cmd or setup).
func buildDetail(t models.Task, unlocked bool, st taskState, avail availability) models.TaskDetail {
	checks := make([]models.CheckName, 0, len(t.Checks))
	for _, ch := range t.Checks {
		checks = append(checks, models.CheckName{Name: ch.Name})
	}
	hints := t.Hints
	if hints == nil {
		hints = []string{}
	}
	image := t.Image
	if image == "" {
		image = defaultImage
	}
	return models.TaskDetail{
		TaskSummary: summarize(t, unlocked, st, avail),
		Image:       image,
		Readme:      t.Readme,
		Hints:       hints,
		Checks:      checks,
	}
}

// GetTasks handles GET /api/tasks (optional auth).
func GetTasks(c *gin.Context) {
	tasks := task.GetTasks()
	userID, _, _, _ := currentUser(c)
	st, err := loadTaskState(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err, "failed to load task state")
		return
	}
	c.JSON(http.StatusOK, buildSummaries(tasks, st, isAdmin(c), loadAvailability(tasks)))
}

// GetTask handles GET /api/tasks/:id (optional auth).
func GetTask(c *gin.Context) {
	id := c.Param("id")
	if !taskIDPattern.MatchString(id) {
		respondError(c, http.StatusBadRequest, "Invalid task id", "")
		return
	}
	tasks := task.GetTasks()
	idx := indexOfTask(tasks, id)
	if idx < 0 {
		respondError(c, http.StatusNotFound, "Task not found", "")
		return
	}
	userID, _, _, _ := currentUser(c)
	st, err := loadTaskState(c.Request.Context(), userID)
	if err != nil {
		internalError(c, err, "failed to load task state")
		return
	}
	avail := loadAvailability(tasks)
	unlocked := unlockedSet(tasks, st.completed, isAdmin(c), avail)
	c.JSON(http.StatusOK, buildDetail(tasks[idx], unlocked[id], st, avail))
}

func indexOfTask(tasks []models.Task, id string) int {
	for i := range tasks {
		if tasks[i].ID == id {
			return i
		}
	}
	return -1
}
