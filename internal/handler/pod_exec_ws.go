package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/response"
	logx "yunshu/internal/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	wsExecMaxDuration    = 2 * time.Hour
	wsExecReauthInterval = time.Minute
	wsExecReauthTimeout  = 10 * time.Second
	wsExecWriteTimeout   = 10 * time.Second
)

type wsExecMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

type wsTerminalSizeQueue struct {
	ch <-chan remotecommand.TerminalSize
}

func (q *wsTerminalSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  32 * 1024,
	WriteBufferSize: 32 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		// 浏览器导航通常带 Sec-Fetch-Site；无 Origin 的非浏览器客户端仍放行。
		if origin == "" {
			return strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")) == ""
		}
		oh, err := url.Parse(origin)
		if err != nil || oh.Host == "" {
			return false
		}
		reqHost := strings.TrimSpace(r.Host)
		if reqHost == "" {
			return false
		}
		if strings.EqualFold(oh.Host, reqHost) {
			return true
		}
		ohName := strings.ToLower(strings.Split(oh.Host, ":")[0])
		reqName := strings.ToLower(strings.Split(reqHost, ":")[0])
		return ohName != "" && ohName == reqName
	},
}

// ExecWS godoc
// @Summary Interactive pod exec (WebSocket)
// @Description WebSocket shell into a pod. Requires cluster_id, namespace, name query params. Obtain ticket via POST /api/v1/auth/ws-ticket first.
// @Tags Pods
// @Produce json
// @Security BearerAuth
// @Param cluster_id query int true "Cluster ID"
// @Param namespace query string true "Pod namespace"
// @Param name query string true "Pod name"
// @Param container query string false "Container name"
// @Param ticket query string true "One-time WebSocket ticket"
// @Router /api/v1/pods/exec/ws [get]
func (h *PodHandler) ExecWS(c *gin.Context) {
	clusterID64, err := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	if err != nil || clusterID64 == 0 {
		response.Error(c, constants.ErrBadRequestWithMsg(constants.ErrMsgba2a155d1253))
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	name := strings.TrimSpace(c.Query("name"))
	container := strings.TrimSpace(c.Query("container"))
	if namespace == "" || name == "" {
		response.Error(c, constants.ErrBadRequestWithMsg(constants.ErrMsge278df185255))
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	audit := logx.With(c.Request.Context(), "component", "http.ws.pod_exec")
	sess := newWSSession(c.Request.Context(), audit)
	defer sess.Cancel()
	defer sess.Wait()
	startedAt := time.Now()

	userID, username := uint(0), ""
	if u, ok := auth.CurrentUserFromContext(c); ok && u != nil {
		userID, username = u.ID, u.Username
	}
	auditFields := []any{
		"user_id", userID,
		"username", username,
		"cluster_id", clusterID64,
		"namespace", namespace,
		"pod", name,
		"container", container,
	}

	stdinR, stdinW := io.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	sizeCh := make(chan remotecommand.TerminalSize, 10)
	defer close(sizeCh)

	var writeMu sync.Mutex
	writeJSON := func(msg wsExecMessage) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(wsExecWriteTimeout))
		_ = conn.WriteJSON(msg)
	}
	writeText := func(s string) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(wsExecWriteTimeout))
		_ = conn.WriteMessage(websocket.TextMessage, []byte(s))
	}

	wsWriter := &wsTextWriter{write: func(p []byte) (int, error) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(wsExecWriteTimeout))
		err := conn.WriteJSON(wsExecMessage{Type: "stdout", Data: string(p)})
		if err != nil {
			return 0, err
		}
		return len(p), nil
	}}

	var (
		stopOnce sync.Once
		stopped  atomic.Bool
		stopNote atomic.Value
	)
	// stop 保证会话只终止一次；reason 非空时先告知前端再断开。
	stop := func(reason string) {
		stopOnce.Do(func() {
			stopped.Store(true)
			stopNote.Store(reason)
			if reason != "" {
				writeJSON(wsExecMessage{Type: "error", Data: reason})
			}
			sess.Cancel()
			_ = stdinW.Close()
		})
	}

	conn.SetReadLimit(32 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	sess.Go("deadline", func() {
		timer := time.NewTimer(wsExecMaxDuration)
		defer timer.Stop()
		select {
		case <-sess.Context().Done():
		case <-timer.C:
			stop("会话已达到最长时长限制，请重新连接")
		}
	})

	if revalidate, ok := auth.WSRevalidatorFromContext(sess.Context()); ok {
		sess.Go("reauth", func() {
			ticker := time.NewTicker(wsExecReauthInterval)
			defer ticker.Stop()
			for {
				select {
				case <-sess.Context().Done():
					return
				case <-ticker.C:
					checkCtx, cancel := context.WithTimeout(sess.Context(), wsExecReauthTimeout)
					err := revalidate(checkCtx)
					cancel()
					if err != nil {
						audit.Warn("pod exec ws revalidation failed", append(auditFields, "error", err)...)
						stop(wsReauthReason(err))
						return
					}
				}
			}
		})
	}

	sess.Go("ping", func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-sess.Context().Done():
				return
			case <-ticker.C:
				writeMu.Lock()
				err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
				writeMu.Unlock()
				if err != nil {
					stop("")
					return
				}
			}
		}
	})

	sess.Go("read", func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				stop("")
				return
			}
			var msg wsExecMessage
			if e := json.Unmarshal(raw, &msg); e != nil {
				continue
			}
			switch msg.Type {
			case "input":
				if msg.Data != "" {
					_, _ = stdinW.Write([]byte(msg.Data))
				}
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					select {
					case sizeCh <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}:
					default:
					}
				}
			case "close":
				stop("")
				return
			default:
			}
		}
	})

	writeJSON(wsExecMessage{Type: "ready"})
	writeText("\r\n")
	audit.Info("pod exec ws session started", auditFields...)

	err = h.svc.ExecTTYStream(
		sess.Context(),
		uint(clusterID64),
		namespace,
		name,
		container,
		stdinR,
		wsWriter,
		wsWriter,
		&wsTerminalSizeQueue{ch: sizeCh},
	)
	if !stopped.Load() {
		if err != nil {
			writeJSON(wsExecMessage{Type: "error", Data: err.Error()})
		} else {
			writeJSON(wsExecMessage{Type: "exit"})
		}
	}
	durationMs := time.Since(startedAt).Milliseconds()
	reason := wsStopReason(&stopNote)
	if err != nil && !errors.Is(err, context.Canceled) {
		audit.Warn("pod exec ws session finished with error", append(auditFields, "duration_ms", durationMs, "reason", reason, "error", err)...)
	} else {
		audit.Info("pod exec ws session finished", append(auditFields, "duration_ms", durationMs, "reason", reason)...)
	}
	sess.Cancel()
	sess.Wait()
}

func wsStopReason(v *atomic.Value) string {
	if v == nil {
		return ""
	}
	s, _ := v.Load().(string)
	return s
}

func wsReauthReason(err error) string {
	switch {
	case errors.Is(err, auth.ErrWSAccountDisabled):
		return "账号已被禁用，连接已关闭"
	case errors.Is(err, auth.ErrWSSessionExpired):
		return "登录状态已失效，请重新登录"
	default:
		return "会话或集群授权已失效，请重新连接"
	}
}

type wsTextWriter struct {
	write func([]byte) (int, error)
}

func (w *wsTextWriter) Write(p []byte) (int, error) {
	if w == nil || w.write == nil {
		return len(p), nil
	}
	return w.write(p)
}
