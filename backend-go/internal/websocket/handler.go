package websocket

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/devops-lab/backend/internal/docker"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = 25 * time.Second
	idleTimeout    = 30 * time.Minute
	maxFrameBytes  = 64 << 10
	maxTermSize    = 500
	killExecWait   = 5 * time.Second
	outputBufBytes = 32 << 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:   4096,
	WriteBufferSize:  outputBufBytes,
	HandshakeTimeout: 10 * time.Second,
	// Same-host only until InitUpgrader installs the configured allowlist.
	CheckOrigin: CheckOriginFromConfig(nil),
}

// InitUpgrader installs the Origin allowlist (CORS_ORIGINS) used to validate upgrades.
func InitUpgrader(allowedOrigins []string) {
	upgrader.CheckOrigin = CheckOriginFromConfig(allowedOrigins)
	log.Info().Strs("allowed_origins", allowedOrigins).Msg("WebSocket origin check configured")
}

// terminal is one live socket <-> docker exec bridge.
type terminal struct {
	conn      *websocket.Conn
	hijack    types.HijackedResponse
	cancel    context.CancelFunc
	closeOnce sync.Once
}

// close tears down both sides exactly once; safe to call from any goroutine.
func (t *terminal) close(code int, reason string) {
	t.closeOnce.Do(func() {
		_ = t.conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(code, reason), time.Now().Add(writeWait))
		t.cancel()
		t.hijack.Close()
		_ = t.conn.Close()
	})
}

var registry = struct {
	sync.Mutex
	m map[*terminal]struct{}
}{m: make(map[*terminal]struct{})}

func register(t *terminal) {
	registry.Lock()
	registry.m[t] = struct{}{}
	registry.Unlock()
}

func unregister(t *terminal) {
	registry.Lock()
	delete(registry.m, t)
	registry.Unlock()
}

// CloseAll closes every open terminal socket (CloseGoingAway). Used on graceful shutdown,
// since http.Server.Shutdown does not track hijacked connections.
func CloseAll(reason string) int {
	registry.Lock()
	ts := make([]*terminal, 0, len(registry.m))
	for t := range registry.m {
		ts = append(ts, t)
	}
	registry.Unlock()
	for _, t := range ts {
		t.close(websocket.CloseGoingAway, reason)
	}
	return len(ts)
}

// ActiveConnections returns the number of open terminal sockets.
func ActiveConnections() int {
	registry.Lock()
	defer registry.Unlock()
	return len(registry.m)
}

// resizeFrame is the only structured client->server message.
type resizeFrame struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// parseControlFrame inspects a text frame. isControl reports a {"type":"resize"} frame
// (which must never be forwarded to stdin); ok reports whether its dimensions are valid.
func parseControlFrame(data []byte) (cols, rows uint, isControl, ok bool) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' || len(trimmed) > 256 {
		return 0, 0, false, false
	}
	var f resizeFrame
	if err := json.Unmarshal(trimmed, &f); err != nil || f.Type != "resize" {
		return 0, 0, false, false
	}
	if f.Cols < 1 || f.Cols > maxTermSize || f.Rows < 1 || f.Rows > maxTermSize {
		return 0, 0, true, false
	}
	return uint(f.Cols), uint(f.Rows), true, true
}

// Handler bridges an authenticated WebSocket to an interactive shell in the session container.
func Handler(c *gin.Context) {
	sess, id, err := AuthenticateWebSocket(c)
	if err != nil {
		return // response already written
	}
	if docker.Client == nil {
		writeErr(c, http.StatusServiceUnavailable, "Service unavailable", "Container runtime unavailable")
		return
	}
	if !userConns.acquire(id.UserID) {
		writeErr(c, http.StatusTooManyRequests, "Too many terminals",
			fmt.Sprintf("At most %d terminal connections per user", MaxConnsPerUser))
		return
	}
	defer userConns.release(id.UserID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Warn().Err(err).Str("session_id", sess.ID).Msg("ws: upgrade failed")
		return // upgrader already replied
	}
	conn.SetReadLimit(maxFrameBytes)

	logger := log.With().Str("session_id", sess.ID).Uint("user_id", id.UserID).Logger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// argv[0] marker lets us find and SIGHUP this shell if the socket goes away first
	// (Docker has no API to kill an exec).
	marker := "dj-term-" + randomHex(8)
	created, err := docker.Client.ContainerExecCreate(ctx, sess.ContainerID, container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Env:          []string{"TERM=xterm-256color"},
		Cmd:          []string{"bash", "-c", "exec -a " + marker + " bash -l"},
	})
	if err != nil {
		logger.Error().Err(err).Msg("ws: exec create failed")
		failAndClose(conn, "failed to start terminal")
		return
	}
	hijack, err := docker.Client.ContainerExecAttach(ctx, created.ID, container.ExecStartOptions{Tty: true})
	if err != nil {
		logger.Error().Err(err).Msg("ws: exec attach failed")
		failAndClose(conn, "failed to attach terminal")
		return
	}

	t := &terminal{conn: conn, hijack: hijack, cancel: cancel}
	register(t)
	defer unregister(t)
	logger.Info().Msg("ws: terminal connected")

	var lastActivity atomic.Int64
	lastActivity.Store(time.Now().UnixNano())

	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	var wg sync.WaitGroup
	wg.Add(2)

	// docker -> websocket (the only data writer on conn)
	go func() {
		defer wg.Done()
		buf := make([]byte, outputBufBytes)
		for {
			n, rerr := hijack.Reader.Read(buf)
			if n > 0 {
				_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					t.close(websocket.CloseGoingAway, "")
					return
				}
			}
			if rerr != nil {
				t.close(websocket.CloseNormalClosure, "terminal exited")
				return
			}
		}
	}()

	// keepalive, idle timeout and session expiry
	go func() {
		defer wg.Done()
		ping := time.NewTicker(pingPeriod)
		defer ping.Stop()
		expiry := time.NewTimer(time.Until(sess.ExpiresAt))
		defer expiry.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-expiry.C:
				t.close(websocket.CloseNormalClosure, "session expired")
				return
			case <-ping.C:
				if time.Since(time.Unix(0, lastActivity.Load())) > idleTimeout {
					t.close(websocket.CloseNormalClosure, "idle timeout")
					return
				}
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
					t.close(websocket.CloseGoingAway, "")
					return
				}
			}
		}
	}()

	// websocket -> docker
	var lastCols, lastRows uint
	for {
		mt, data, rerr := conn.ReadMessage()
		if rerr != nil {
			if websocket.IsUnexpectedCloseError(rerr, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logger.Debug().Err(rerr).Msg("ws: read ended")
			}
			break
		}
		lastActivity.Store(time.Now().UnixNano())

		if mt == websocket.TextMessage {
			if cols, rows, isControl, ok := parseControlFrame(data); isControl {
				if ok && (cols != lastCols || rows != lastRows) {
					lastCols, lastRows = cols, rows
					rctx, rcancel := context.WithTimeout(ctx, 2*time.Second)
					if err := docker.ResizeExec(rctx, created.ID, cols, rows); err != nil {
						logger.Debug().Err(err).Msg("ws: resize failed")
					}
					rcancel()
				}
				continue
			}
		}
		if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
			continue
		}
		_ = hijack.Conn.SetWriteDeadline(time.Now().Add(writeWait))
		if _, werr := hijack.Conn.Write(data); werr != nil {
			break
		}
	}

	t.close(websocket.CloseNormalClosure, "")
	wg.Wait()
	killExec(sess.ContainerID, created.ID, marker)
	logger.Info().Msg("ws: terminal disconnected")
}

// killExec HUPs the shell if it is still running after the socket closed.
func killExec(containerID, execID, marker string) {
	ctx, cancel := context.WithTimeout(context.Background(), killExecWait)
	defer cancel()
	if info, err := docker.Client.ContainerExecInspect(ctx, execID); err == nil && !info.Running {
		return
	}
	script := `for p in /proc/[0-9]*; do case "$(tr '\0' ' ' < "$p/cmdline" 2>/dev/null)" in "` +
		marker + ` "*) kill -HUP "${p#/proc/}" 2>/dev/null;; esac; done; true`
	if res := docker.ExecCommand(ctx, containerID, script, killExecWait); res != nil && res.Error != nil {
		log.Debug().Err(res.Error).Str("exec_id", execID).Msg("ws: failed to stop orphaned shell")
	}
}

func failAndClose(conn *websocket.Conn, msg string) {
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	_ = conn.WriteMessage(websocket.TextMessage, []byte("\r\nError: "+msg+"\r\n"))
	_ = conn.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseInternalServerErr, msg), time.Now().Add(writeWait))
	_ = conn.Close()
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
