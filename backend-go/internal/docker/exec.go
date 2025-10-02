package docker

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/rs/zerolog/log"
)

// ExecResult represents the result of a command execution
type ExecResult struct {
	ExitCode int
	Output   string
	Error    error
}

// ExecCommand executes a command in a container
func ExecCommand(ctx context.Context, containerID, command string, timeout time.Duration) *ExecResult {
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create exec instance
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		WorkingDir:   "/root",
		Cmd:          []string{"bash", "-lc", command},
	}

	execID, err := Client.ContainerExecCreate(execCtx, containerID, execConfig)
	if err != nil {
		log.Error().Err(err).Str("container", containerID[:12]).Msg("Failed to create exec")
		return &ExecResult{ExitCode: 1, Error: err}
	}

	// Start exec
	resp, err := Client.ContainerExecAttach(execCtx, execID.ID, container.ExecStartOptions{
		Tty: false,
	})
	if err != nil {
		log.Error().Err(err).Str("exec_id", execID.ID[:12]).Msg("Failed to attach exec")
		return &ExecResult{ExitCode: 1, Error: err}
	}
	defer resp.Close()

	// Read output
	var buf bytes.Buffer
	_, err = buf.ReadFrom(resp.Reader)
	if err != nil {
		log.Error().Err(err).Msg("Failed to read exec output")
		return &ExecResult{ExitCode: 1, Error: err}
	}

	// Get exit code
	inspect, err := Client.ContainerExecInspect(execCtx, execID.ID)
	if err != nil {
		log.Error().Err(err).Msg("Failed to inspect exec")
		return &ExecResult{ExitCode: 1, Output: buf.String(), Error: err}
	}

	result := &ExecResult{
		ExitCode: inspect.ExitCode,
		Output:   buf.String(),
	}

	log.Debug().
		Str("container", containerID[:12]).
		Int("exit_code", result.ExitCode).
		Str("command", command).
		Msg("Command executed")

	return result
}

// RunSetupCommands runs setup commands for a task
func RunSetupCommands(ctx context.Context, containerID string, commands []string) error {
	for i, cmd := range commands {
		log.Info().
			Int("step", i+1).
			Int("total", len(commands)).
			Str("command", truncate(cmd, 60)).
			Msg("Running setup command")

		result := ExecCommand(ctx, containerID, cmd, 60*time.Second)
		if result.Error != nil {
			return fmt.Errorf("setup command %d failed: %w", i+1, result.Error)
		}

		if result.ExitCode != 0 {
			log.Warn().
				Int("step", i+1).
				Int("exit_code", result.ExitCode).
				Str("output", truncate(result.Output, 200)).
				Msg("Setup command returned non-zero exit code")
		}
	}

	log.Info().Msg("All setup commands completed")
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
