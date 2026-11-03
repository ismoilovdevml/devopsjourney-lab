package websocket

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/devops-lab/backend/internal/docker"
	"github.com/devops-lab/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/moby/moby/client"
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

	// CloseAuthRevoked is the close code sent when the user behind an open
	// terminal is deleted, deactivated, or changed their password (4000-4999
	// are application codes; 4401 mirrors HTTP 401).
	CloseAuthRevoked = 4401
)

// revalidate re-checks the socket owner's account; variable so tests can stub it.
var revalidate = middleware.Revalidate

// authCloseReason maps a revalidation error to a close reason. ok is false
// for nil and for transient errors (DB hiccup), which must not end a terminal.
func authCloseReason(err error) (reason string, ok bool) {
	switch {
	case err == nil:
		return "", false
	case errors.Is(err, middleware.ErrTokenRevoked):
		return "signed out: password changed", true
	case errors.Is(err, middleware.ErrUserDisabled):
		return "account disabled", true
	case errors.Is(err, middleware.ErrUnauthenticated):
		return "account no longer exists", true
	default:
		return "", false
	}
}

// sessionActivity tracks terminal attachment per lab session, for idle
// reclaim (session.Manager.SetActivitySource).
type sessionActivity struct {
	conns int
	last  time.Time
}

// activityRetention bounds how long a detached session's last-activity time is kept.
const activityRetention = 6 * time.Hour

var activity = struct {
	sync.Mutex
	m map[string]*sessionActivity
}{m: make(map[string]*sessionActivity)}

// trackSession records a terminal attaching (delta=+1) or detaching (delta=-1).
func trackSession(sessionID string, delta int, now time.Time) {
	activity.Lock()
	defer activity.Unlock()
	a, ok := activity.m[sessionID]
	if !ok {
		if delta < 0 {
			return
		}
		if len(activity.m) > 1000 {
			for id, e := range activity.m {
				if e.conns == 0 && now.Sub(e.last) > activityRetention {
					delete(activity.m, id)
				}
			}
		}
		a = &sessionActivity{}
		activity.m[sessionID] = a
	}
	a.conns = max(a.conns+delta, 0)
	a.last = now
}

// SessionActivity reports whether a terminal is attached to sessionID and
// when one last attached or detached (zero if never seen by this process).
func SessionActivity(sessionID string) (attached bool, last time.Time) {
	activity.Lock()
	defer activity.Unlock()
	if a, ok := activity.m[sessionID]; ok {
		return a.conns > 0, a.last
	}
	return false, time.Time{}
}

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
	hijack    client.HijackedResponse
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
	created, err := docker.Client.ExecCreate(ctx, sess.ContainerID, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Env:          []string{"TERM=xterm-256color"},
		Cmd:          []string{"bash", "-c", "exec -a " + marker + " bash -l"},
	})
	if err != nil {
		logger.Error().Err(err).Msg("ws: exec create failed")
		failAndClose(conn, "failed to start terminal")
		return
	}
	attached, err := docker.Client.ExecAttach(ctx, created.ID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		logger.Error().Err(err).Msg("ws: exec attach failed")
		failAndClose(conn, "failed to attach terminal")
		return
	}
	hijack := attached.HijackedResponse

	t := &terminal{conn: conn, hijack: hijack, cancel: cancel}
	register(t)
	defer unregister(t)
	trackSession(sess.ID, +1, time.Now())
	defer func() { trackSession(sess.ID, -1, time.Now()) }()
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

	// keepalive, idle timeout, session expiry and account revalidation
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
				// A password change, deactivation or deletion must also end
				// terminals that were opened before it (cached up to 30s).
				if reason, revoked := authCloseReason(revalidate(id)); revoked {
					logger.Info().Str("reason", reason).Msg("ws: closing terminal, credentials no longer valid")
					t.close(CloseAuthRevoked, reason)
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
	if info, err := docker.Client.ExecInspect(ctx, execID, client.ExecInspectOptions{}); err == nil && !info.Running {
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
