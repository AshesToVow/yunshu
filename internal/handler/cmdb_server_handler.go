package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/exportutil"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

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

// ProbeServer 远端只读资源探测（SSH 白名单命令）。
func (h *CMDBHandler) ProbeServer(c *gin.Context) {
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
	ServeJSON(c, func(ctx context.Context, req service.HostProbeRequest) (*service.HostProbeResult, error) {
		req.ProjectID = projectID
		req.ServerID = serverID
		req.Actor = actor
		return h.svc.ProbeHostMetrics(ctx, req)
	})
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
	result, err := h.svc.ImportServersFromExcel(c.Request.Context(), projectID, exportutil.LimitedImportReader(file))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
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
	filename := fmt.Sprintf("project-%d-servers.xlsx", projectID)
	if err := exportutil.ServeExcel(c, filename, f); err != nil {
		response.Error(c, err)
	}
}

// ServersImportTemplate 澶勭悊瀵瑰簲鐨?HTTP 璇锋眰骞惰繑鍥炵粺涓€鍝嶅簲銆?
func (h *CMDBHandler) ServersImportTemplate(c *gin.Context) {
	f, err := h.svc.ServersImportTemplateExcel()
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := exportutil.ServeExcel(c, "servers-import-template.xlsx", f); err != nil {
		response.Error(c, err)
	}
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
			"list":            list,
			"path":            q.Path,
			"max_transfer_mb": h.svc.MaxTransferFileMB(ctx),
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
