package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/middleware"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/pkg/sshclient"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type CMDBHandler struct {
	svc *service.CMDBService
}

func NewCMDBHandler(svc *service.CMDBService) *CMDBHandler {
	return &CMDBHandler{svc: svc}
}
// ListServers 鏌ヨ鍒楄〃瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) ListServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeQuery(c, func(ctx context.Context, q service.ServerListQuery) (*pagination.Result[service.ServerItem], error) {
		q.ProjectID = projectID
		q.Actor = actor
		return h.svc.ListServers(ctx, q)
	})
}

// UpsertServer 澶勭悊瀵瑰簲鐨?HTTP 璇锋眰骞惰繑鍥炵粺涓€鍝嶅簲銆?
func (h *CMDBHandler) UpsertServer(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req service.ServerUpsertRequest) (*service.ServerItem, error) {
		req.ProjectID = projectID
		if req.ID != nil && *req.ID > 0 {
			if err := h.svc.AssertServerAccess(ctx, projectID, *req.ID, actor, "manage"); err != nil {
				return nil, err
			}
		} else if err := h.svc.AssertCanCreateServer(ctx, projectID, actor); err != nil {
			return nil, err
		}
		return h.svc.UpsertServer(ctx, req)
	})
}

// DeleteServer 鍒犻櫎瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) DeleteServer(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	id, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertServerAccess(auth.RequestContext(c), projectID, id, actor, "manage"); err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteServer(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// ServerDetail 澶勭悊瀵瑰簲鐨?HTTP 璇锋眰骞惰繑鍥炵粺涓€鍝嶅簲銆?
func (h *CMDBHandler) ServerDetail(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertServerAccess(auth.RequestContext(c), projectID, serverID, actor, "view"); err != nil {
		response.Error(c, err)
		return
	}
	data, err := h.svc.GetServer(c.Request.Context(), serverID)
	if err != nil {
		response.Error(c, err)
		return
	}
	if data.ProjectID != projectID {
		response.Error(c, constants.ErrServerNotInCurrentProject)
		return
	}
	response.Success(c, data)
}

// ExecServerCommand 澶勭悊瀵瑰簲鐨?HTTP 璇锋眰骞惰繑鍥炵粺涓€鍝嶅簲銆?
func (h *CMDBHandler) ExecServerCommand(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertServerAccess(auth.RequestContext(c), projectID, serverID, actor, "exec"); err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ServerExecRequest) (*service.ServerExecResult, error) {
		req.ProjectID = projectID
		req.ServerID = serverID
		return h.svc.ExecServerCommand(ctx, req)
	})
}

// ListServerGroups 鏌ヨ鍒楄〃瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) ListServerGroups(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, req service.ServerGroupTreeQuery) ([]service.ServerGroupItem, error) {
		req.ProjectID = projectID
		return h.svc.ListServerGroupTree(ctx, req)
	})
}

// UpsertServerGroup 澶勭悊瀵瑰簲鐨?HTTP 璇锋眰骞惰繑鍥炵粺涓€鍝嶅簲銆?
func (h *CMDBHandler) UpsertServerGroup(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ServerGroupUpsertRequest) (*service.ServerGroupItem, error) {
		req.ProjectID = projectID
		return h.svc.UpsertServerGroup(ctx, req)
	})
}

// UpdateServerGroup 鏇存柊瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) UpdateServerGroup(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	groupID, err := parseUintParam(c, "groupId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ServerGroupUpsertRequest) (*service.ServerGroupItem, error) {
		req.ProjectID = projectID
		req.ID = &groupID
		return h.svc.UpsertServerGroup(ctx, req)
	})
}

// DeleteServerGroup 鍒犻櫎瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) DeleteServerGroup(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	groupID, err := parseUintParam(c, "groupId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteServerGroup(c.Request.Context(), projectID, groupID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// ListCloudAccounts 鏌ヨ鍒楄〃瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) ListCloudAccounts(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, req service.CloudAccountListQuery) ([]service.CloudAccountItem, error) {
		req.ProjectID = projectID
		return h.svc.ListCloudAccounts(ctx, req)
	})
}

// UpsertCloudAccount 澶勭悊瀵瑰簲鐨?HTTP 璇锋眰骞惰繑鍥炵粺涓€鍝嶅簲銆?
func (h *CMDBHandler) UpsertCloudAccount(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.CloudAccountUpsertRequest) (*service.CloudAccountItem, error) {
		req.ProjectID = projectID
		return h.svc.UpsertCloudAccount(ctx, req)
	})
}

// UpdateCloudAccount 鏇存柊瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) UpdateCloudAccount(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	accountID, err := parseUintParam(c, "accountId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.CloudAccountUpsertRequest) (*service.CloudAccountItem, error) {
		req.ProjectID = projectID
		req.ID = &accountID
		return h.svc.UpsertCloudAccount(ctx, req)
	})
}

// DeleteCloudAccount 鍒犻櫎瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) DeleteCloudAccount(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	accountID, err := parseUintParam(c, "accountId")
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := h.svc.DeleteCloudAccount(c.Request.Context(), projectID, accountID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// SyncCloudAccount 鍚屾瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) SyncCloudAccount(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	accountID, err := parseUintParam(c, "accountId")
	if err != nil {
		response.Error(c, err)
		return
	}
	res, err := h.svc.SyncCloudAccount(c.Request.Context(), service.CloudSyncRequest{
		ProjectID: projectID,
		AccountID: accountID,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, res)
}

// TestServer 娴嬭瘯瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) TestServer(c *gin.Context) {
	ServeJSON(c, h.svc.TestServerConnectivity)
}

// BatchTestServers 澶勭悊瀵瑰簲鐨?HTTP 璇锋眰骞惰繑鍥炵粺涓€鍝嶅簲銆?
func (h *CMDBHandler) BatchTestServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.BatchServerTestRequest) (*service.BatchServerTestResult, error) {
		req.ProjectID = projectID
		return h.svc.BatchTestServerConnectivity(ctx, req)
	})
}

// CloudServerAction 鎵ц浜戞湇鍔″櫒鎿嶄綔锛堟敼瀵?閲嶅惎/鍏虫満锛夈€?
func (h *CMDBHandler) CloudServerAction(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.CloudServerActionRequest) (*service.CloudServerActionResult, error) {
		return h.svc.RunCloudServerAction(ctx, projectID, serverID, req)
	})
}

// SyncServers 鍚屾瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) SyncServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	ServeJSON(c, func(ctx context.Context, req service.ServerSyncRequest) (*service.ServerSyncResult, error) {
		req.ProjectID = projectID
		return h.svc.SyncProjectServers(ctx, req)
	})
}

// ImportServers 瀵煎叆瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) ImportServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.Error(c, constants.ErrUploadFailed)
		return
	}
	defer file.Close()
	n, err := h.svc.ImportServersFromExcel(c.Request.Context(), projectID, file)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"imported": n})
}

// ExportServers 瀵煎嚭瀵瑰簲鐨?HTTP 鎺ュ彛澶勭悊閫昏緫銆?
func (h *CMDBHandler) ExportServers(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	keyword := c.Query("keyword")
	f, err := h.svc.ExportServersToExcel(c.Request.Context(), projectID, keyword)
	if err != nil {
		response.Error(c, err)
		return
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		response.Error(c, err)
		return
	}
	filename := fmt.Sprintf("project-%d-servers.xlsx", projectID)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ServersImportTemplate 澶勭悊瀵瑰簲鐨?HTTP 璇锋眰骞惰繑鍥炵粺涓€鍝嶅簲銆?
func (h *CMDBHandler) ServersImportTemplate(c *gin.Context) {
	f, err := h.svc.ServersImportTemplateExcel()
	if err != nil {
		response.Error(c, err)
		return
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		response.Error(c, err)
		return
	}
	filename := "servers-import-template.xlsx"
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// ServerTerminalWS godoc
// @Summary Interactive SSH terminal (WebSocket)
// @Description WebSocket upgrade for project server SSH. Obtain ticket via POST /api/v1/auth/ws-ticket first, then pass ticket= in query string.
// @Tags CMDB
// @Produce json
// @Security BearerAuth
// @Param id path int true "Project ID"
// @Param serverId path int true "Server ID"
// @Param ticket query string true "One-time WebSocket ticket"
// @Router /api/v1/projects/{id}/servers/{serverId}/terminal/ws [get]
func (h *CMDBHandler) ServerTerminalWS(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertServerAccess(auth.RequestContext(c), projectID, serverID, actor, "exec"); err != nil {
		// WebSocket 必须立刻写响应，避免后续 Upgrade；用 AbortWithError 而非仅挂 c.Error。
		middleware.AbortWithError(c, err)
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	sess := newWSSession(c.Request.Context(), logx.With(c.Request.Context(), "component", "http.ws.terminal"))
	defer sess.Cancel()
	defer sess.Wait()

	stdinR, stdinW := io.Pipe()
	defer stdinR.Close()
	defer stdinW.Close()

	sizeCh := make(chan sshclient.TerminalSize, 10)
	defer close(sizeCh)

	var writeMu sync.Mutex
	writeJSON := func(msg wsExecMessage) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.WriteJSON(msg)
	}

	wsWriter := &wsTextWriter{write: func(p []byte) (int, error) {
		writeMu.Lock()
		defer writeMu.Unlock()
		if err := conn.WriteJSON(wsExecMessage{Type: "stdout", Data: string(p)}); err != nil {
			return 0, err
		}
		return len(p), nil
	}}

	conn.SetReadLimit(2 * 1024 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

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
					sess.Cancel()
					return
				}
			}
		}
	})

	sess.Go("read", func() {
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				sess.Cancel()
				_ = stdinW.Close()
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
					case sizeCh <- sshclient.TerminalSize{Cols: msg.Cols, Rows: msg.Rows}:
					default:
					}
				}
			case "close":
				sess.Cancel()
				_ = stdinW.Close()
				return
			default:
			}
		}
	})

	writeJSON(wsExecMessage{Type: "ready"})

	if err := h.svc.StreamServerTerminal(sess.Context(), projectID, serverID, stdinR, wsWriter, wsWriter, sizeCh); err != nil {
		writeJSON(wsExecMessage{Type: "error", Data: err.Error()})
		sess.Cancel()
		sess.Wait()
		return
	}
	writeJSON(wsExecMessage{Type: "exit"})
	sess.Cancel()
	sess.Wait()
}

func (h *CMDBHandler) ListServerGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	var userID, serverID uint
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		if parsed, e := strconv.ParseUint(v, 10, 32); e == nil {
			userID = uint(parsed)
		}
	}
	if v := strings.TrimSpace(c.Query("server_id")); v != "" {
		if parsed, e := strconv.ParseUint(v, 10, 32); e == nil {
			serverID = uint(parsed)
		}
	}
	list, err := h.svc.ListServerGrants(c.Request.Context(), projectID, actor, userID, serverID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

// MyServerAccess 当前用户对指定服务器的有效权限（含 owner/admin 隐式全量）。
func (h *CMDBHandler) MyServerAccess(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	perm, err := h.svc.EffectiveServerAccess(c.Request.Context(), projectID, serverID, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, perm)
}

func (h *CMDBHandler) UpsertServerGrant(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req service.ServerGrantUpsertRequest) (*model.ServerAccessGrant, error) {
		req.ProjectID = projectID
		if actor != nil {
			id := actor.ID
			req.CreatedBy = &id
		}
		return h.svc.UpsertServerGrant(ctx, req, actor)
	})
}

func (h *CMDBHandler) BulkUpsertServerGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	ServeJSON(c, func(ctx context.Context, req service.ServerGrantBulkRequest) (gin.H, error) {
		req.ProjectID = projectID
		if actor != nil {
			id := actor.ID
			req.CreatedBy = &id
		}
		n, err := h.svc.BulkUpsertServerGrants(ctx, req, actor)
		if err != nil {
			return nil, err
		}
		return gin.H{"upserted": n}, nil
	})
}

func (h *CMDBHandler) DeleteServerGrant(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	grantID, err := parseUintParam(c, "grantId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.DeleteServerGrant(c.Request.Context(), projectID, grantID, actor); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

func (h *CMDBHandler) BootstrapServerGrants(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	req := service.BootstrapServerGrantsRequest{ProjectID: projectID}
	if actor != nil {
		id := actor.ID
		req.CreatedBy = &id
	}
	stats, err := h.svc.BootstrapServerGrantsForMembers(c.Request.Context(), req, actor)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, stats)
}

// ListServerFiles 列出服务器远端目录。
func (h *CMDBHandler) ListServerFiles(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertServerAccess(auth.RequestContext(c), projectID, serverID, actor, "exec"); err != nil {
		response.Error(c, err)
		return
	}
	ServeQuery(c, func(ctx context.Context, q service.ServerFileListQuery) (gin.H, error) {
		q.ProjectID = projectID
		q.ServerID = serverID
		list, err := h.svc.ListServerFiles(ctx, q)
		if err != nil {
			return nil, err
		}
		return gin.H{
			"list":              list,
			"path":              q.Path,
			"max_transfer_mb":   h.svc.MaxTransferFileMB(ctx),
		}, nil
	})
}

// UploadServerFile 上传文件到服务器。
func (h *CMDBHandler) UploadServerFile(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertServerAccess(auth.RequestContext(c), projectID, serverID, actor, "exec"); err != nil {
		response.Error(c, err)
		return
	}
	remoteDir := strings.TrimSpace(c.PostForm("path"))
	if remoteDir == "" {
		remoteDir = "/"
	}
	fh, err := c.FormFile("file")
	if err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg("请选择上传文件"))
		return
	}
	file, err := fh.Open()
	if err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg("打开上传文件失败"))
		return
	}
	defer file.Close()
	if err := h.svc.UploadServerFile(c.Request.Context(), projectID, serverID, remoteDir, fh.Filename, file, fh.Size); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "uploaded", "max_transfer_mb": h.svc.MaxTransferFileMB(c.Request.Context())})
}

// DownloadServerFile 下载服务器文件。
func (h *CMDBHandler) DownloadServerFile(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertServerAccess(auth.RequestContext(c), projectID, serverID, actor, "exec"); err != nil {
		response.Error(c, err)
		return
	}
	remotePath := strings.TrimSpace(c.Query("path"))
	if remotePath == "" {
		response.Error(c, constants.ErrBadRequestWithMsg("path 必填"))
		return
	}
	filename := path.Base(strings.ReplaceAll(remotePath, "\\", "/"))
	if filename == "" || filename == "." || filename == "/" {
		filename = "download.bin"
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(filename))
	c.Status(http.StatusOK)
	if _, err := h.svc.DownloadServerFile(c.Request.Context(), projectID, serverID, remotePath, c.Writer); err != nil {
		// headers may already be flushed; best-effort JSON error for early failures
		if !c.Writer.Written() {
			response.Error(c, err)
		}
		return
	}
}

// DeleteServerFile 删除服务器远端文件。
func (h *CMDBHandler) DeleteServerFile(c *gin.Context) {
	projectID, err := parseUintParam(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}
	serverID, err := parseUintParam(c, "serverId")
	if err != nil {
		response.Error(c, err)
		return
	}
	actor, _ := auth.CurrentUserFromContext(c)
	if err := h.svc.AssertServerAccess(auth.RequestContext(c), projectID, serverID, actor, "exec"); err != nil {
		response.Error(c, err)
		return
	}
	ServeJSONOK(c, gin.H{"message": "deleted"}, func(ctx context.Context, req service.ServerFilePathQuery) error {
		req.ProjectID = projectID
		req.ServerID = serverID
		return h.svc.DeleteServerFile(ctx, projectID, serverID, req.Path)
	})
}

