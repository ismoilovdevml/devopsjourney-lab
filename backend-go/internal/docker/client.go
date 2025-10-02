package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

var Client *client.Client

// InitClient initializes Docker client
func InitClient(socketPath string) error {
	var err error
	Client, err = client.NewClientWithOpts(
		client.WithHost("unix://"+socketPath),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test connection
	ctx := context.Background()
	_, err = Client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping Docker daemon: %w", err)
	}

	log.Info().Msg("Docker client initialized successfully")
	return nil
}

// EnsureImage pulls image if not exists
func EnsureImage(ctx context.Context, imageName string) error {
	// Check if image exists
	_, _, err := Client.ImageInspectWithRaw(ctx, imageName)
	if err == nil {
		log.Debug().Str("image", imageName).Msg("Image already exists")
		return nil
	}

	log.Info().Str("image", imageName).Msg("Pulling image...")

	reader, err := Client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer reader.Close()

	// Read output to ensure pull completes
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read image pull output: %w", err)
	}

	log.Info().Str("image", imageName).Msg("Image pulled successfully")
	return nil
}

// CreateContainer creates a new container
func CreateContainer(ctx context.Context, cfg *ContainerConfig) (string, error) {
	config := &container.Config{
		Image:      cfg.Image,
		Tty:        true,
		OpenStdin:  true,
		WorkingDir: "/root",
		Cmd:        []string{"/bin/bash"},
	}

	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:   cfg.MemoryLimit,
			NanoCPUs: cfg.CPULimit,
			PidsLimit: &cfg.PidsLimit,
		},
		SecurityOpt: []string{"no-new-privileges:true"},
		CapDrop:     []string{"ALL"},
		CapAdd:      []string{"NET_BIND_SERVICE"},
		ReadonlyRootfs: false,
		Tmpfs: map[string]string{
			"/run": "rw,noexec,nosuid,size=64m",
		},
	}

	resp, err := Client.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	log.Debug().Str("container_id", resp.ID[:12]).Msg("Container created")
	return resp.ID, nil
}

// StartContainer starts a container
func StartContainer(ctx context.Context, containerID string) error {
	err := Client.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	log.Debug().Str("container_id", containerID[:12]).Msg("Container started")
	return nil
}

// StopContainer stops a container
func StopContainer(ctx context.Context, containerID string, timeout int) error {
	stopTimeout := timeout
	err := Client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &stopTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	log.Debug().Str("container_id", containerID[:12]).Msg("Container stopped")
	return nil
}

// RemoveContainer removes a container
func RemoveContainer(ctx context.Context, containerID string) error {
	err := Client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true,
	})
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	log.Debug().Str("container_id", containerID[:12]).Msg("Container removed")
	return nil
}

// InspectContainer gets container info
func InspectContainer(ctx context.Context, containerID string) (*types.ContainerJSON, error) {
	info, err := Client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}
	return &info, nil
}

// Close closes the Docker client
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}
