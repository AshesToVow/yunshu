package handler

import (
	"context"

	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

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
