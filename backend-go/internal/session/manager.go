package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/devops-lab/backend/configs"
	"github.com/devops-lab/backend/internal/database"
	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/models"
	"github.com/devops-lab/backend/internal/task"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Typed errors callers can match with errors.Is.
var (
	// ErrCapacity means the global MAX_SESSIONS_TOTAL cap is reached.
	ErrCapacity = errors.New("all lab slots are busy, try again in a few minutes")
	// ErrSessionNotFound means there is no active session with that ID.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired wraps ErrSessionNotFound, so errors.Is(err, ErrSessionNotFound) also holds.
	ErrSessionExpired = fmt.Errorf("session expired: %w", ErrSessionNotFound)
	// ErrTaskNotFound means the task ID is unknown.
	ErrTaskNotFound = errors.New("task not found")
	// ErrCheckInProgress means a check is already running for this session.
	ErrCheckInProgress = errors.New("a check is already running for this session")
	// ErrContainerNotRunning means the lab container is gone or stopped.
	ErrContainerNotRunning = errors.New("lab container is not running")
)

// Session statuses stored in the database.
const (
	StatusActive  = "active"
	StatusStopped = "stopped"
	StatusExpired = "expired"
)

const (
	// DefaultLabImage is the image built from backend-go/lab-image.
	DefaultLabImage = "devopslab-lab:latest"
	// DefaultMaxSessions is used when the config does not set a cap.
	DefaultMaxSessions = 20

	checkTimeout    = 10 * time.Second
	setupTimeout    = 30 * time.Second
	maxCheckOutput  = 2048
	stopGracePeriod = 2 // seconds
	// defaultOrphanGrace protects containers that are still being provisioned
	// (no DB row yet) from the orphan sweeper.
	defaultOrphanGrace = 3 * time.Minute
)

// Manager owns the lifecycle of lab sessions: one Docker container per
// session, persisted in Postgres and cached in Redis.
type Manager struct {
	cfg            *configs.DockerConfig
	maxSessions    int
	orphanGrace    time.Duration
	cleanupManager *CleanupManager

	// createMu serializes "count running containers + create container" so
	// concurrent requests cannot overshoot the global cap.
	createMu sync.Mutex

	locksMu    sync.Mutex
	userLocks  map[uint]*sync.Mutex
	checksBusy map[string]bool
}

// NewManager creates a session manager.
func NewManager(cfg *configs.DockerConfig) *Manager {
	maxSessions := cfg.MaxSessionsTotal
	if maxSessions <= 0 {
		maxSessions = DefaultMaxSessions
	}
	return &Manager{
		cfg:            cfg,
		maxSessions:    maxSessions,
		orphanGrace:    defaultOrphanGrace,
		cleanupManager: NewCleanupManager(),
		userLocks:      make(map[uint]*sync.Mutex),
		checksBusy:     make(map[string]bool),
	}
}

// MaxSessions returns the global cap on concurrently running lab containers.
func (m *Manager) MaxSessions() int { return m.maxSessions }

// CreateSession starts a lab container for userID and taskID. Any previous
// active session of the same user is stopped first (one session per user).
func (m *Manager) CreateSession(ctx context.Context, userID uint, taskID string) (*models.SessionResponse, error) {
	taskObj, err := task.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	unlock := m.lockUser(userID)
	defer unlock()

	if err := m.stopUserSessions(ctx, userID); err != nil {
		return nil, err
	}

	sessionID, err := newSessionID()
	if err != nil {
		return nil, err
	}

	log.Info().Uint("user_id", userID).Str("task_id", taskID).Str("session_id", sessionID).Msg("Creating session")

	containerID, err := m.provisionContainer(ctx, sessionID, userID, taskObj)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	ttl := time.Duration(m.cfg.SessionTTL) * time.Second
	sess := &models.Session{
		ID:          sessionID,
		UserID:      userID,
		TaskID:      taskID,
		ContainerID: containerID,
		Status:      StatusActive,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
	}
	if err := database.DB.WithContext(ctx).Create(sess).Error; err != nil {
		m.destroyContainer(containerID)
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	if err := m.cacheSession(ctx, sess); err != nil {
		log.Warn().Err(err).Msg("Failed to cache session in Redis")
	}

	m.cleanupManager.Schedule(sessionID, ttl, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := m.endSession(ctx, sessionID, StatusExpired); err != nil && !errors.Is(err, ErrSessionNotFound) {
			log.Error().Err(err).Str("session_id", sessionID).Msg("Failed to clean up expired session")
		}
	})

	log.Info().Str("session_id", sessionID).Str("container_id", shortID(containerID)).Msg("Session created")
	return toResponse(sess), nil
}

// provisionContainer enforces the global cap, creates and starts a hardened
// container for the task and runs the task's setup commands. It does not touch
// the database, so the integration test drives exactly this code path.
func (m *Manager) provisionContainer(ctx context.Context, sessionID string, userID uint, taskObj *models.Task) (string, error) {
	rt, err := loadTaskRuntime(m.cfg.TasksDir, taskObj.ID)
	if err != nil {
		return "", err
	}

	cc := &docker.ContainerConfig{
		Image:       resolveImage(taskObj.Image, m.cfg.DefaultImage),
		MemoryLimit: m.cfg.MemLimit,
		CPULimit:    m.cfg.CPUNano,
		PidsLimit:   m.cfg.PidsLimit,
		Labels: map[string]string{
			docker.LabelSession: sessionID,
			docker.LabelUser:    strconv.FormatUint(uint64(userID), 10),
			docker.LabelTask:    taskObj.ID,
		},
		ExtraCapabilities: rt.Capabilities,
		Network:           rt.Network,
	}

	if err := docker.EnsureImage(ctx, cc.Image); err != nil {
		return "", fmt.Errorf("lab image unavailable: %w", err)
	}

	containerID, err := m.createWithinCapacity(ctx, cc)
	if err != nil {
		return "", err
	}

	if err := docker.StartContainer(ctx, containerID); err != nil {
		m.destroyContainer(containerID)
		return "", err
	}

	if err := docker.RunSetupCommands(ctx, containerID, taskObj.Setup, setupTimeout); err != nil {
		m.destroyContainer(containerID)
		return "", fmt.Errorf("task setup failed: %w", err)
	}
	return containerID, nil
}

// createWithinCapacity counts live lab containers and creates a new one only if
// the global cap allows it.
func (m *Manager) createWithinCapacity(ctx context.Context, cc *docker.ContainerConfig) (string, error) {
	m.createMu.Lock()
	defer m.createMu.Unlock()

	existing, err := docker.ListLabContainers(ctx)
	if err != nil {
		return "", err
	}
	if !hasCapacity(countLive(existing), m.maxSessions) {
		log.Warn().Int("max", m.maxSessions).Msg("Session capacity reached")
		return "", ErrCapacity
	}
	return docker.CreateContainer(ctx, cc)
}

// GetSession returns an active, non-expired session.
func (m *Manager) GetSession(ctx context.Context, sessionID string) (*models.Session, error) {
	if sess, err := m.getCachedSession(ctx, sessionID); err == nil && sess != nil && sess.Status == StatusActive {
		if time.Now().After(sess.ExpiresAt) {
			return nil, ErrSessionExpired
		}
		return sess, nil
	}

	var sess models.Session
	err := database.DB.WithContext(ctx).Where("id = ? AND status = ?", sessionID, StatusActive).First(&sess).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load session: %w", err)
	}
	if time.Now().After(sess.ExpiresAt) {
		// Expire it now instead of waiting for the periodic sweep.
		if err := m.endSession(ctx, sessionID, StatusExpired); err != nil && !errors.Is(err, ErrSessionNotFound) {
			log.Warn().Err(err).Str("session_id", sessionID).Msg("Failed to expire session")
		}
		return nil, ErrSessionExpired
	}
	return &sess, nil
}

// ActiveSessionForUser returns the user's active session, or (nil, nil).
func (m *Manager) ActiveSessionForUser(ctx context.Context, userID uint) (*models.Session, error) {
	var sess models.Session
	err := database.DB.WithContext(ctx).
		Where("user_id = ? AND status = ? AND expires_at > ?", userID, StatusActive, time.Now()).
		Order("created_at DESC").
		First(&sess).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load active session: %w", err)
	}
	return &sess, nil
}

// DeleteSession stops a session: removes its container and marks it stopped.
func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	return m.endSession(ctx, sessionID, StatusStopped)
}

// Recreate replaces a session's container with a fresh one for the same user
// and task (the old container is removed).
func (m *Manager) Recreate(ctx context.Context, sessionID string) (*models.SessionResponse, error) {
	sess, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	// CreateSession stops every active session of the user, including this one.
	return m.CreateSession(ctx, sess.UserID, sess.TaskID)
}

// CheckSolution runs the task's checks inside the session's container. Checks
// run sequentially, each with a 10s timeout; at most one CheckSolution runs per
// session at a time (ErrCheckInProgress otherwise).
func (m *Manager) CheckSolution(ctx context.Context, sessionID string) (*models.CheckResponse, error) {
	sess, err := m.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	taskObj, err := task.GetTask(sess.TaskID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, sess.TaskID)
	}

	if !m.tryLockCheck(sessionID) {
		return nil, ErrCheckInProgress
	}
	defer m.unlockCheck(sessionID)

	info, err := docker.InspectContainer(ctx, sess.ContainerID)
	if err != nil || info.State == nil || !info.State.Running {
		return nil, ErrContainerNotRunning
	}

	results := runChecks(ctx, sess.ContainerID, taskObj.Checks)
	resp := buildCheckResponse(results)

	if err := database.DB.WithContext(ctx).Model(&models.Session{}).
		Where("id = ?", sessionID).
		Update("last_check_at", time.Now()).Error; err != nil {
		log.Warn().Err(err).Str("session_id", sessionID).Msg("Failed to update last_check_at")
	}

	log.Info().Str("session_id", sessionID).Int("score", resp.Score).Int("total", resp.Total).
		Bool("passed", resp.Passed).Msg("Checks completed")
	return resp, nil
}

// runChecks executes checks one after another and never returns an error:
// a check that cannot run is reported as failed with an explanation.
func runChecks(ctx context.Context, containerID string, checks []models.Check) []models.CheckResult {
	results := make([]models.CheckResult, 0, len(checks))
	for _, check := range checks {
		res := docker.ExecCommand(ctx, containerID, check.Cmd, checkTimeout)
		cr := models.CheckResult{Name: check.Name, OK: res.Error == nil && res.ExitCode == 0}
		switch {
		case res.TimedOut:
			cr.OK = false
			cr.Output = fmt.Sprintf("check timed out after %s", checkTimeout)
		case res.Error != nil:
			cr.Output = "check could not run: " + res.Error.Error()
		default:
			cr.Output = res.Output
		}
		cr.Output = docker.Truncate(cr.Output, maxCheckOutput)
		results = append(results, cr)
	}
	return results
}

func buildCheckResponse(results []models.CheckResult) *models.CheckResponse {
	passed := 0
	for _, r := range results {
		if r.OK {
			passed++
		}
	}
	return &models.CheckResponse{
		Passed:  len(results) > 0 && passed == len(results),
		Score:   passed,
		Total:   len(results),
		Results: results,
	}
}

// ResourceStats reports running lab containers and their CPU/memory usage.
func (m *Manager) ResourceStats(ctx context.Context) (*models.ResourceStats, error) {
	containers, err := docker.ListLabContainers(ctx)
	if err != nil {
		return nil, err
	}

	running := make([]docker.LabContainer, 0, len(containers))
	for _, c := range containers {
		if c.State == "running" {
			running = append(running, c)
		}
	}

	usernames := m.usernames(ctx, running)

	out := &models.ResourceStats{
		ContainersRunning: len(running),
		ContainersLimit:   m.maxSessions,
		Sessions:          make([]models.ResourceSessionStats, len(running)),
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, c := range running {
		out.Sessions[i] = models.ResourceSessionStats{
			SessionID: c.SessionID,
			Username:  usernames[c.UserID],
			TaskID:    c.TaskID,
		}
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			u, err := docker.ContainerStats(sctx, id)
			if err != nil {
				log.Debug().Err(err).Str("container_id", shortID(id)).Msg("Failed to read container stats")
				return
			}
			out.Sessions[i].CPUPercent = u.CPUPercent
			out.Sessions[i].MemBytes = u.MemBytes
			out.Sessions[i].MemLimitBytes = u.MemLimitBytes
		}(i, c.ID)
	}
	wg.Wait()
	return out, nil
}

func (m *Manager) usernames(ctx context.Context, containers []docker.LabContainer) map[string]string {
	names := map[string]string{}
	if database.DB == nil || len(containers) == 0 {
		return names
	}
	ids := make([]uint64, 0, len(containers))
	for _, c := range containers {
		if id, err := strconv.ParseUint(c.UserID, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	var users []models.User
	if err := database.DB.WithContext(ctx).Select("id", "username").Where("id IN ?", ids).Find(&users).Error; err != nil {
		log.Warn().Err(err).Msg("Failed to load usernames for resource stats")
		return names
	}
	for _, u := range users {
		names[strconv.FormatUint(uint64(u.ID), 10)] = u.Username
	}
	return names
}

// Shutdown gracefully shuts down the manager (containers are left running; the
// next start reconciles them).
func (m *Manager) Shutdown(timeout time.Duration) error {
	log.Info().Msg("Shutting down session manager")
	return m.cleanupManager.Shutdown(timeout)
}

// GetActiveTimers returns the number of active cleanup timers.
func (m *Manager) GetActiveTimers() int {
	return m.cleanupManager.GetActiveTimers()
}

// endSession removes the session's container and marks it with status.
func (m *Manager) endSession(ctx context.Context, sessionID, status string) error {
	var sess models.Session
	err := database.DB.WithContext(ctx).Where("id = ? AND status = ?", sessionID, StatusActive).First(&sess).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		m.dropCache(ctx, sessionID)
		return ErrSessionNotFound
	}
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}
	m.finishSession(ctx, &sess, status)
	return nil
}

// finishSession tears down a loaded session; errors are logged, not returned,
// so the DB row is always closed even if Docker misbehaves.
func (m *Manager) finishSession(ctx context.Context, sess *models.Session, status string) {
	log.Info().Str("session_id", sess.ID).Str("status", status).Msg("Ending session")
	m.cleanupManager.Remove(sess.ID)

	if err := docker.RemoveContainer(ctx, sess.ContainerID); err != nil {
		log.Error().Err(err).Str("session_id", sess.ID).Msg("Failed to remove container")
	}

	now := time.Now()
	if err := database.DB.WithContext(ctx).Model(&models.Session{}).
		Where("id = ?", sess.ID).
		Updates(map[string]interface{}{"status": status, "ended_at": now}).Error; err != nil {
		log.Error().Err(err).Str("session_id", sess.ID).Msg("Failed to update session status")
	}
	m.dropCache(ctx, sess.ID)
}

// stopUserSessions stops every active session of a user.
func (m *Manager) stopUserSessions(ctx context.Context, userID uint) error {
	var sessions []models.Session
	if err := database.DB.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, StatusActive).
		Find(&sessions).Error; err != nil {
		return fmt.Errorf("failed to load previous sessions: %w", err)
	}
	for i := range sessions {
		m.finishSession(ctx, &sessions[i], StatusStopped)
	}
	return nil
}

func (m *Manager) destroyContainer(containerID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := docker.RemoveContainer(ctx, containerID); err != nil {
		log.Error().Err(err).Str("container_id", shortID(containerID)).Msg("Failed to remove container")
	}
}

func (m *Manager) lockUser(userID uint) func() {
	m.locksMu.Lock()
	l, ok := m.userLocks[userID]
	if !ok {
		l = &sync.Mutex{}
		m.userLocks[userID] = l
	}
	m.locksMu.Unlock()
	l.Lock()
	return l.Unlock
}

func (m *Manager) tryLockCheck(sessionID string) bool {
	m.locksMu.Lock()
	defer m.locksMu.Unlock()
	if m.checksBusy[sessionID] {
		return false
	}
	m.checksBusy[sessionID] = true
	return true
}

func (m *Manager) unlockCheck(sessionID string) {
	m.locksMu.Lock()
	delete(m.checksBusy, sessionID)
	m.locksMu.Unlock()
}

func (m *Manager) cacheSession(ctx context.Context, sess *models.Session) error {
	if database.RedisClient == nil {
		return nil
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return database.RedisClient.Set(ctx, "session:"+sess.ID, data, time.Until(sess.ExpiresAt)).Err()
}

func (m *Manager) getCachedSession(ctx context.Context, sessionID string) (*models.Session, error) {
	if database.RedisClient == nil {
		return nil, errors.New("redis not configured")
	}
	data, err := database.RedisClient.Get(ctx, "session:"+sessionID).Bytes()
	if err != nil {
		return nil, err
	}
	var sess models.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (m *Manager) dropCache(ctx context.Context, sessionID string) {
	if database.RedisClient != nil {
		_ = database.RedisClient.Del(ctx, "session:"+sessionID).Err()
	}
}

func toResponse(s *models.Session) *models.SessionResponse {
	ttl := int(time.Until(s.ExpiresAt).Seconds())
	if ttl < 0 {
		ttl = 0
	}
	return &models.SessionResponse{
		SessionID:  s.ID,
		TaskID:     s.TaskID,
		TTLSeconds: ttl,
		ExpiresAt:  s.ExpiresAt,
		CreatedAt:  s.CreatedAt,
	}
}

// ToResponse converts a stored session into the API response shape.
func ToResponse(s *models.Session) *models.SessionResponse { return toResponse(s) }

// countLive counts lab containers that occupy a slot (anything not already
// exited/dead; those are orphans the sweeper removes).
func countLive(containers []docker.LabContainer) int {
	n := 0
	for _, c := range containers {
		switch c.State {
		case "exited", "dead", "removing":
		default:
			n++
		}
	}
	return n
}

// hasCapacity reports whether one more container fits under the cap.
func hasCapacity(live, max int) bool {
	return max > 0 && live < max
}

func resolveImage(taskImage, defaultImage string) string {
	switch {
	case taskImage != "":
		return taskImage
	case defaultImage != "":
		return defaultImage
	default:
		return DefaultLabImage
	}
}

func newSessionID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
