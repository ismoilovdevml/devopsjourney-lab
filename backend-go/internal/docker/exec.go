package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
	"github.com/rs/zerolog/log"
)

// MaxExecOutput caps how much output ExecCommand keeps in memory. Anything
// beyond it is read and discarded so the exec stream can finish.
const MaxExecOutput = 64 * 1024

// timeoutExitCode is what coreutils `timeout` returns when it killed the command.
const timeoutExitCode = 124

// ExecResult represents the result of a command execution.
type ExecResult struct {
	ExitCode int
	Output   string // combined stdout+stderr, at most MaxExecOutput bytes
	TimedOut bool
	Error    error // transport/API error (not a non-zero exit code)
}

// checkPath is the PATH for checks and setup commands. System directories come
// first so a student cannot shadow e.g. grep with /usr/local/bin/grep; the
// /usr/local directories are still needed for tools such as hadolint and
// docker-compose that the lab image installs there.
const checkPath = "/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/sbin:/usr/local/bin"

// ExecCommand runs command with a clean, non-interactive bash inside a
// container as root and returns its exit code and combined output.
//
// The shell never reads student-controlled startup files: `env -i` drops the
// container environment (including BASH_ENV and ENV) and `--noprofile
// --norc` skips /etc/profile, ~/.bash_profile, ~/.bashrc and friends. A
// student who writes `exit 0` into ~/.bash_profile therefore cannot make every
// check pass. (Root inside the container can still replace system binaries,
// so check results are best-effort anti-cheat; see SECURITY.md.)
//
// The command is wrapped in coreutils `timeout`, so a hanging check is killed
// inside the container (not only abandoned by the API call), and the API call
// itself is bounded by timeout plus a small grace period.
func ExecCommand(ctx context.Context, containerID, command string, timeout time.Duration) *ExecResult {
	secs := int(timeout.Round(time.Second) / time.Second)
	if secs < 1 {
		secs = 1
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout+5*time.Second)
	defer cancel()

	execID, err := Client.ExecCreate(execCtx, containerID, client.ExecCreateOptions{
		AttachStdout: true,
		AttachStderr: true,
		TTY:          false,
		WorkingDir:   "/root",
		Cmd:          wrapWithTimeout(command, secs),
	})
	if err != nil {
		return &ExecResult{ExitCode: -1, Error: fmt.Errorf("exec create: %w", err)}
	}

	resp, err := Client.ExecAttach(execCtx, execID.ID, client.ExecAttachOptions{TTY: false})
	if err != nil {
		return &ExecResult{ExitCode: -1, Error: fmt.Errorf("exec attach: %w", err)}
	}
	defer resp.Close()

	// The hijacked connection ignores ctx; close it when ctx expires so the
	// read below cannot block forever.
	stop := context.AfterFunc(execCtx, resp.Close)
	defer stop()

	out := &limitedBuffer{max: MaxExecOutput}
	if _, err := stdcopy.StdCopy(out, out, resp.Reader); err != nil && !errors.Is(err, io.EOF) {
		if execCtx.Err() != nil {
			return &ExecResult{ExitCode: -1, Output: out.String(), TimedOut: true, Error: fmt.Errorf("exec timed out: %w", execCtx.Err())}
		}
		return &ExecResult{ExitCode: -1, Output: out.String(), Error: fmt.Errorf("exec read: %w", err)}
	}

	inspect, err := Client.ExecInspect(execCtx, execID.ID, client.ExecInspectOptions{})
	if err != nil {
		return &ExecResult{ExitCode: -1, Output: out.String(), Error: fmt.Errorf("exec inspect: %w", err)}
	}

	result := &ExecResult{
		ExitCode: inspect.ExitCode,
		Output:   out.String(),
		TimedOut: inspect.ExitCode == timeoutExitCode,
	}
	log.Debug().
		Str("container", shortID(containerID)).
		Int("exit_code", result.ExitCode).
		Msg("Command executed")
	return result
}

// wrapWithTimeout builds the exec argv: run command in a clean environment
// with a bash that reads no startup files, and kill it after secs seconds
// (SIGTERM, then SIGKILL 2s later). Absolute paths so a PATH lookup cannot be
// redirected.
func wrapWithTimeout(command string, secs int) []string {
	return []string{
		"/usr/bin/timeout", "-k", "2", strconv.Itoa(secs),
		"/usr/bin/env", "-i",
		"PATH=" + checkPath, "HOME=/root", "LANG=C.UTF-8", "TERM=dumb",
		"/bin/bash", "--noprofile", "--norc", "-c", command,
	}
}

// RunSetupCommands runs a task's setup commands in order. Any non-zero exit
// code aborts setup: a half-prepared lab is worse than a clear error.
func RunSetupCommands(ctx context.Context, containerID string, commands []string, perCommand time.Duration) error {
	for i, cmd := range commands {
		result := ExecCommand(ctx, containerID, cmd, perCommand)
		if result.Error != nil {
			return fmt.Errorf("setup command %d/%d failed: %w", i+1, len(commands), result.Error)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("setup command %d/%d (%s) exited with %d: %s",
				i+1, len(commands), Truncate(cmd, 60), result.ExitCode, Truncate(result.Output, 300))
		}
	}
	return nil
}

// Truncate shortens s to at most maxBytes bytes without splitting a UTF-8
// sequence, appending a marker when something was cut.
func Truncate(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	const marker = "\n...[truncated]"
	cut := maxBytes - len(marker)
	if cut < 0 {
		cut = 0
	}
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + marker
}

// limitedBuffer keeps the first max bytes written and silently drops the rest
// (while still reporting success, so the producer keeps draining).
type limitedBuffer struct {
	buf []byte
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if room := b.max - len(b.buf); room > 0 {
		if len(p) > room {
			b.buf = append(b.buf, p[:room]...)
		} else {
			b.buf = append(b.buf, p...)
		}
	}
	return len(p), nil
}

func (b *limitedBuffer) String() string { return string(b.buf) }
