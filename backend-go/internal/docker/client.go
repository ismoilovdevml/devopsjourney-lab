package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

// Client is the shared Docker API client. It is exported because the
// websocket package attaches terminals with it directly.
var Client *client.Client

// InitClient initializes the Docker client.
//
// host may be a full daemon URL (unix://, tcp://, ...) or a bare unix socket
// path. Precedence: an explicit URL argument, then the DOCKER_HOST environment
// variable (e.g. a docker-socket-proxy such as tcp://docker-proxy:2375), then
// the socket path. DOCKER_API_VERSION / DOCKER_CERT_PATH / DOCKER_TLS_VERIFY
// are honoured through client.FromEnv.
func InitClient(host string) error {
	opts := []client.Opt{client.FromEnv, client.WithAPIVersionNegotiation()}
	if h := resolveHost(host, os.Getenv("DOCKER_HOST")); h != "" {
		opts = append(opts, client.WithHost(h))
	}

	c, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.Ping(ctx); err != nil {
		_ = c.Close()
		return fmt.Errorf("failed to ping Docker daemon at %s: %w", c.DaemonHost(), err)
	}

	Client = c
	log.Info().Str("host", c.DaemonHost()).Msg("Docker client initialized successfully")
	return nil
}

// resolveHost returns the host to pass to client.WithHost, or "" to keep the
// value client.FromEnv already applied (DOCKER_HOST).
func resolveHost(arg, envHost string) string {
	switch {
	case strings.Contains(arg, "://"):
		return arg
	case envHost != "":
		return ""
	case arg != "":
		return "unix://" + arg
	default:
		return ""
	}
}

// Ping checks that the Docker daemon is reachable.
func Ping(ctx context.Context) error {
	if Client == nil {
		return fmt.Errorf("docker client not initialized")
	}
	_, err := Client.Ping(ctx)
	return err
}

// EnsureImage makes sure an image is present locally, pulling it if needed.
// Locally built images (e.g. devopslab-lab:latest) are never pulled when they
// already exist.
func EnsureImage(ctx context.Context, imageName string) error {
	if _, _, err := Client.ImageInspectWithRaw(ctx, imageName); err == nil {
		return nil
	}

	log.Info().Str("image", imageName).Msg("Pulling image...")
	reader, err := Client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("image %s not found locally and pull failed: %w", imageName, err)
	}
	defer reader.Close()

	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("failed to read image pull output: %w", err)
	}

	log.Info().Str("image", imageName).Msg("Image pulled successfully")
	return nil
}

// BuildCapAdd returns the capability list for a container: BaseCapabilities
// plus the requested extras. Unknown or non-allowlisted extras are an error.
func BuildCapAdd(extra []string) ([]string, error) {
	caps := append([]string(nil), BaseCapabilities...)
	seen := make(map[string]bool, len(caps))
	for _, c := range caps {
		seen[c] = true
	}
	for _, raw := range extra {
		c := strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(raw)), "CAP_")
		if c == "" || seen[c] {
			continue
		}
		if !ExtraCapabilityAllowlist[c] {
			return nil, fmt.Errorf("capability %q is not allowed for lab containers", raw)
		}
		seen[c] = true
		caps = append(caps, c)
	}
	return caps, nil
}

// BuildContainerConfig returns the container.Config for a lab container.
func BuildContainerConfig(cfg *ContainerConfig) *container.Config {
	labels := make(map[string]string, len(cfg.Labels))
	for k, v := range cfg.Labels {
		labels[k] = v
	}
	return &container.Config{
		Image:      cfg.Image,
		Hostname:   LabHostname,
		WorkingDir: "/root",
		// PID 1 is docker-init (HostConfig.Init) which reaps the daemons
		// students start; sleep just keeps the container alive.
		Cmd:    []string{"sleep", "infinity"},
		Env:    []string{"TERM=xterm-256color", "LANG=C.UTF-8"},
		Labels: labels,
	}
}

// BuildHostConfig returns the hardened HostConfig for a lab container.
//
// The backend effectively has root on the host through the Docker API, so a
// lab container gets: all capabilities dropped except a small allowlist,
// no-new-privileges, memory/CPU/PIDs limits, no host mounts, no network by
// default, and an init process to reap zombies.
func BuildHostConfig(cfg *ContainerConfig) (*container.HostConfig, error) {
	caps, err := BuildCapAdd(cfg.ExtraCapabilities)
	if err != nil {
		return nil, err
	}

	networkMode := container.NetworkMode("none")
	if cfg.Network {
		networkMode = container.NetworkMode("bridge")
	}

	pids := cfg.PidsLimit
	var pidsPtr *int64
	if pids > 0 {
		pidsPtr = &pids
	}

	initProc := true
	hc := &container.HostConfig{
		NetworkMode: networkMode,
		Privileged:  false,
		CapDrop:     []string{"ALL"},
		CapAdd:      caps,
		SecurityOpt: []string{"no-new-privileges:true"},
		Init:        &initProc,
		IpcMode:     container.IpcMode("private"),
		Resources: container.Resources{
			Memory:     cfg.MemoryLimit,
			MemorySwap: cfg.MemoryLimit, // equal to Memory => no swap
			NanoCPUs:   cfg.CPULimit,
			PidsLimit:  pidsPtr,
			Ulimits: []*container.Ulimit{
				{Name: "nofile", Soft: 1024, Hard: 4096},
				{Name: "core", Soft: 0, Hard: 0},
			},
		},
		Tmpfs: map[string]string{
			"/tmp": "rw,nosuid,nodev,size=64m,mode=1777",
		},
		// Nothing useful is written to the container's stdout; don't let a
		// student fill the host disk through the log driver.
		LogConfig:      container.LogConfig{Type: "none"},
		ReadonlyRootfs: false, // students edit files all over the filesystem
		AutoRemove:     false,
		RestartPolicy:  container.RestartPolicy{Name: container.RestartPolicyDisabled},
	}
	if cfg.MemoryLimit <= 0 {
		hc.Resources.MemorySwap = 0
	}
	return hc, nil
}

// CreateContainer creates a new hardened lab container and returns its ID.
func CreateContainer(ctx context.Context, cfg *ContainerConfig) (string, error) {
	hostConfig, err := BuildHostConfig(cfg)
	if err != nil {
		return "", err
	}

	resp, err := Client.ContainerCreate(ctx, BuildContainerConfig(cfg), hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	log.Debug().Str("container_id", shortID(resp.ID)).Msg("Container created")
	return resp.ID, nil
}

// StartContainer starts a container.
func StartContainer(ctx context.Context, containerID string) error {
	if err := Client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}
	log.Debug().Str("container_id", shortID(containerID)).Msg("Container started")
	return nil
}

// StopContainer stops a container.
func StopContainer(ctx context.Context, containerID string, timeout int) error {
	stopTimeout := timeout
	if err := Client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &stopTimeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}
	log.Debug().Str("container_id", shortID(containerID)).Msg("Container stopped")
	return nil
}

// RemoveContainer force-removes a container. A container that no longer exists
// is not an error.
func RemoveContainer(ctx context.Context, containerID string) error {
	err := Client.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	if err != nil && !client.IsErrNotFound(err) {
		return fmt.Errorf("failed to remove container: %w", err)
	}
	log.Debug().Str("container_id", shortID(containerID)).Msg("Container removed")
	return nil
}

// InspectContainer gets container info.
func InspectContainer(ctx context.Context, containerID string) (*types.ContainerJSON, error) {
	info, err := Client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	return &info, nil
}

// ListLabContainers returns every container (running or not) that carries the
// devopslab.session label, oldest first.
func ListLabContainers(ctx context.Context) ([]LabContainer, error) {
	list, err := Client.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filters.NewArgs(filters.Arg("label", LabelSession)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list lab containers: %w", err)
	}

	out := make([]LabContainer, 0, len(list))
	for _, c := range list {
		out = append(out, LabContainer{
			ID:        c.ID,
			SessionID: c.Labels[LabelSession],
			UserID:    c.Labels[LabelUser],
			TaskID:    c.Labels[LabelTask],
			State:     c.State,
			Created:   time.Unix(c.Created, 0),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.Before(out[j].Created) })
	return out, nil
}

// ContainerUsage is a point-in-time resource usage sample of a container.
type ContainerUsage struct {
	CPUPercent    float64
	MemBytes      uint64
	MemLimitBytes uint64
}

// ContainerStats samples CPU and memory usage of a container. It uses a
// non-streaming stats call, which makes the daemon collect two samples about a
// second apart so the CPU percentage is meaningful.
func ContainerStats(ctx context.Context, containerID string) (*ContainerUsage, error) {
	resp, err := Client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer resp.Body.Close()

	var s container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, fmt.Errorf("failed to decode container stats: %w", err)
	}
	return usageFromStats(&s), nil
}

func usageFromStats(s *container.StatsResponse) *ContainerUsage {
	u := &ContainerUsage{MemLimitBytes: s.MemoryStats.Limit}

	// Same formula as `docker stats`: exclude the page cache.
	mem := s.MemoryStats.Usage
	if v, ok := s.MemoryStats.Stats["inactive_file"]; ok && v < mem {
		mem -= v
	} else if v, ok := s.MemoryStats.Stats["total_inactive_file"]; ok && v < mem {
		mem -= v
	}
	u.MemBytes = mem

	cpuDelta := float64(s.CPUStats.CPUUsage.TotalUsage) - float64(s.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(s.CPUStats.SystemUsage) - float64(s.PreCPUStats.SystemUsage)
	online := float64(s.CPUStats.OnlineCPUs)
	if online == 0 {
		online = float64(len(s.CPUStats.CPUUsage.PercpuUsage))
	}
	if cpuDelta > 0 && sysDelta > 0 && online > 0 {
		u.CPUPercent = cpuDelta / sysDelta * online * 100
	}
	return u
}

// ResizeExec resizes the TTY of an exec instance (used by the terminal
// websocket on {"type":"resize"} frames).
func ResizeExec(ctx context.Context, execID string, cols, rows uint) error {
	if cols == 0 || rows == 0 {
		return fmt.Errorf("invalid terminal size %dx%d", cols, rows)
	}
	return Client.ContainerExecResize(ctx, execID, container.ResizeOptions{Width: cols, Height: rows})
}

// Close closes the Docker client.
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
