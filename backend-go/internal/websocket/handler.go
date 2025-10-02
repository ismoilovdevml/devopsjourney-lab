package websocket

import (
	"context"
	"io"

	"github.com/devops-lab/backend/internal/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// CheckOrigin is set dynamically in InitUpgrader
}

// InitUpgrader initializes the WebSocket upgrader with allowed origins
func InitUpgrader(allowedOrigins []string) {
	upgrader.CheckOrigin = CheckOriginFromConfig(allowedOrigins)
	log.Info().Strs("allowed_origins", allowedOrigins).Msg("WebSocket CORS configured")
}

// Handler handles WebSocket terminal connections with authentication
func Handler(c *gin.Context) {
	// Authenticate WebSocket connection
	session, claims, err := AuthenticateWebSocket(c)
	if err != nil {
		// Error response already sent by AuthenticateWebSocket
		return
	}

	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to upgrade to WebSocket")
		return
	}
	defer conn.Close()

	// Set read limit to prevent DOS attacks
	conn.SetReadLimit(1024 * 1024) // 1MB max message size

	log.Info().
		Str("session_id", session.ID).
		Uint("user_id", claims.UserID).
		Str("username", claims.Username).
		Msg("WebSocket connected")

	ctx := context.Background()

	// Create exec instance
	execConfig := container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
		Tty:          true,
		WorkingDir:   "/root",
		Cmd:          []string{"bash", "-l"},
	}

	execID, err := docker.Client.ContainerExecCreate(ctx, session.ContainerID, execConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create exec")
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Error: Failed to create terminal\r\n"))
		return
	}

	// Attach to exec
	resp, err := docker.Client.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{
		Tty: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to attach exec")
		_ = conn.WriteMessage(websocket.TextMessage, []byte("Error: Failed to attach to terminal\r\n"))
		return
	}
	defer resp.Close()

	// Context for graceful cleanup
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	// Channel for errors
	errChan := make(chan error, 2)

	// Docker -> WebSocket
	go func() {
		defer cancel() // Signal other goroutine to stop
		buf := make([]byte, 8192)
		for {
			select {
			case <-ctx2.Done():
				return
			default:
				n, err := resp.Reader.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Error().Err(err).Msg("Error reading from Docker")
						errChan <- err
					}
					return
				}

				if err := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
					log.Error().Err(err).Msg("Error writing to WebSocket")
					errChan <- err
					return
				}
			}
		}
	}()

	// WebSocket -> Docker
	go func() {
		defer cancel() // Signal other goroutine to stop
		for {
			select {
			case <-ctx2.Done():
				return
			default:
				messageType, data, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						log.Error().Err(err).Msg("WebSocket read error")
					}
					errChan <- err
					return
				}

				if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
					if _, err := resp.Conn.Write(data); err != nil {
						log.Error().Err(err).Msg("Error writing to Docker")
						errChan <- err
						return
					}
				}
			}
		}
	}()

	// Wait for error, disconnect, or context cancellation
	select {
	case <-errChan:
	case <-ctx2.Done():
	}

	log.Info().
		Str("session_id", session.ID).
		Uint("user_id", claims.UserID).
		Msg("WebSocket disconnected")
}
