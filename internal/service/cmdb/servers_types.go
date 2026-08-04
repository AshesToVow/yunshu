package cmdb

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
)

type ServerItem struct {
	ID                     uint    `json:"id"`
	ProjectID              uint    `json:"project_id"`
	GroupID                *uint   `json:"group_id,omitempty"`
	Name                   string  `json:"name"`
	Host                   string  `json:"host"`
	Port                   int     `json:"port"`
	OSType                 string  `json:"os_type"`
	OSArch                 string  `json:"os_arch"`
	Tags                   string  `json:"tags"`
	SourceType             string  `json:"source_type"`
	Provider               string  `json:"provider,omitempty"`
	CloudInstanceID        string  `json:"cloud_instance_id,omitempty"`
	CloudRegion            string  `json:"cloud_region,omitempty"`
	CloudZone              string  `json:"cloud_zone,omitempty"`
	CloudSpec              string  `json:"cloud_spec,omitempty"`
	CloudConfigInfo        string  `json:"cloud_config_info,omitempty"`
	CloudOSName            string  `json:"cloud_os_name,omitempty"`
	CloudNetworkInfo       string  `json:"cloud_network_info,omitempty"`
	CloudChargeType        string  `json:"cloud_charge_type,omitempty"`
	CloudNetworkChargeType string  `json:"cloud_network_charge_type,omitempty"`
	CloudTagsJSON          string  `json:"cloud_tags_json,omitempty"`
	CloudPublicIP          string  `json:"cloud_public_ip,omitempty"`
	CloudPrivateIP         string  `json:"cloud_private_ip,omitempty"`
	CloudStatusText        string  `json:"cloud_status_text,omitempty"`
	LastTestAt             *string `json:"last_test_at"`
	LastTestErr            *string `json:"last_test_error"`
	CreatedAt              string  `json:"created_at"`
	LastSeenAt             *string `json:"last_seen_at"`
	Status                 int     `json:"status"`
}

// ServerDetailItem 在 ServerItem 基础上附带 SSH 凭据摘要（不含明文）。
type ServerDetailItem struct {
	ServerItem
	AuthType          string  `json:"auth_type,omitempty"`
	Username          string  `json:"username,omitempty"`
	PasswordSet       bool    `json:"password_set"`
	PrivateKeySet     bool    `json:"private_key_set"`
	UsernameDictLabel *string `json:"username_dict_label,omitempty"`
	PasswordDictLabel *string `json:"password_dict_label,omitempty"`
}

type ServerListQuery struct {
	ProjectID      uint   `form:"project_id" binding:"required"`
	Keyword        string `form:"keyword"`
	GroupID        *uint  `form:"group_id"`
	CloudAccountID *uint  `form:"cloud_account_id"`
	SourceType     string `form:"source_type"`
	Provider       string `form:"provider"`
	Page           int    `form:"page"`
	PageSize       int    `form:"page_size"`
	// Actor 由 handler 注入，用于资源 ACL 过滤。
	Actor *auth.CurrentUser `form:"-"`
}

type ServerUpsertRequest struct {
	ID              *uint  `json:"id"`
	ProjectID       uint   `json:"project_id" binding:"required"`
	GroupID         *uint  `json:"group_id,omitempty"`
	Name            string `json:"name" binding:"required"`
	Host            string `json:"host" binding:"required"`
	Port            int    `json:"port"`
	OSType          string `json:"os_type"`
	Tags            string `json:"tags"`
	Status          int    `json:"status"`
	SourceType      string `json:"source_type"`
	Provider        string `json:"provider"`
	CloudInstanceID string `json:"cloud_instance_id"`
	CloudRegion     string `json:"cloud_region"`

	AuthType   string  `json:"auth_type"` // password/key
	Username   string  `json:"username"`
	Password   *string `json:"password,omitempty"`
	PrivateKey *string `json:"private_key,omitempty"`
	Passphrase *string `json:"passphrase,omitempty"`

	UsernameDictLabel string `json:"username_dict_label"`
	PasswordDictLabel string `json:"password_dict_label"`
}

type ServerExecRequest struct {
	ProjectID  uint   `json:"project_id"`
	ServerID   uint   `json:"server_id"`
	Command    string `json:"command" binding:"required"`
	TimeoutSec int    `json:"timeout_sec"`
}

type ServerExecResult struct {
	ServerID   uint   `json:"server_id"`
	Command    string `json:"command"`
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMS int64  `json:"duration_ms"`
	Truncated  bool   `json:"truncated"`
}

type ServerGroupItem struct {
	ID        uint              `json:"id"`
	ProjectID uint              `json:"project_id"`
	ParentID  *uint             `json:"parent_id,omitempty"`
	Name      string            `json:"name"`
	Category  string            `json:"category"`
	Provider  string            `json:"provider"`
	Sort      int               `json:"sort"`
	Status    int               `json:"status"`
	Children  []ServerGroupItem `json:"children,omitempty"`
}

type ServerGroupUpsertRequest struct {
	ID        *uint  `json:"id,omitempty"`
	ProjectID uint   `json:"project_id" binding:"required"`
	ParentID  *uint  `json:"parent_id,omitempty"`
	Name      string `json:"name" binding:"required"`
	Category  string `json:"category"`
	Provider  string `json:"provider"`
	Sort      int    `json:"sort"`
	Status    int    `json:"status"`
}

type ServerGroupTreeQuery struct {
	ProjectID uint `form:"project_id" binding:"required"`
}

type CloudAccountItem struct {
	ID            uint    `json:"id"`
	ProjectID     uint    `json:"project_id"`
	GroupID       uint    `json:"group_id"`
	Provider      string  `json:"provider"`
	AccountName   string  `json:"account_name"`
	RegionScope   string  `json:"region_scope"`
	Status        int     `json:"status"`
	LastSyncAt    *string `json:"last_sync_at,omitempty"`
	LastSyncError *string `json:"last_sync_error,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type CloudAccountListQuery struct {
	ProjectID uint  `form:"project_id" binding:"required"`
	GroupID   *uint `form:"group_id"`
}

type CloudAccountUpsertRequest struct {
	ID          *uint  `json:"id,omitempty"`
	ProjectID   uint   `json:"project_id" binding:"required"`
	GroupID     uint   `json:"group_id" binding:"required"`
	Provider    string `json:"provider" binding:"required"`
	AccountName string `json:"account_name" binding:"required"`
	RegionScope string `json:"region_scope"`
	AK          string `json:"ak,omitempty"`
	SK          string `json:"sk,omitempty"`
	Status      int    `json:"status"`
}

type CloudSyncRequest struct {
	ProjectID uint `json:"project_id" binding:"required"`
	AccountID uint `json:"account_id" binding:"required"`
}

type CloudSyncResult struct {
	Total     int    `json:"total"`
	Added     int    `json:"added"`
	Updated   int    `json:"updated"`
	Disabled  int    `json:"disabled"`
	Unchanged int    `json:"unchanged"`
	Message   string `json:"message"`
}

type CloudInstance struct {
	InstanceID        string
	Name              string
	Host              string
	Region            string
	Zone              string
	Spec              string
	ConfigInfo        string
	OSName            string
	NetworkInfo       string
	ChargeType        string
	NetworkChargeType string
	TagsJSON          string
	PublicIP          string
	PrivateIP         string
	StatusText        string
	OSType            string
	Status            int
}

type CloudProvider interface {
	ListInstances(ctx context.Context, ak, sk, regionScope string) ([]CloudInstance, error)
	QueryInstanceExpireAt(ctx context.Context, ak, sk, region, instanceID string) (*time.Time, error)
	ResetInstancePassword(ctx context.Context, ak, sk, region, instanceID, newPassword string) error
	RebootInstance(ctx context.Context, ak, sk, region, instanceID string) error
	ShutdownInstance(ctx context.Context, ak, sk, region, instanceID string) error
	SyncInstanceTags(ctx context.Context, ak, sk, region, instanceID string, oldTags, newTags map[string]string) error
}

type ServerTestRequest struct {
	ServerID uint `json:"server_id" binding:"required"`
}

type ServerTestResult struct {
	ServerID uint   `json:"server_id,omitempty"`
	OK       bool   `json:"ok"`
	Message  string `json:"message"`
}

type BatchServerTestRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	ServerIDs []uint `json:"server_ids"`
	Parallel  int    `json:"parallel"`
}

type BatchServerTestResult struct {
	Total   int                `json:"total"`
	Success int                `json:"success"`
	Failed  int                `json:"failed"`
	Results []ServerTestResult `json:"results"`
}

type ServerSyncRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	ServerIDs []uint `json:"server_ids"`
	Parallel  int    `json:"parallel"`
}

type ServerSyncResult struct {
	Total       int                `json:"total"`
	Online      int                `json:"online"`
	Offline     int                `json:"offline"`
	UpdatedAt   string             `json:"updated_at"`
	TestResults []ServerTestResult `json:"test_results"`
}

type CloudServerActionRequest struct {
	Action      string `json:"action" binding:"required,oneof=reset_password reboot shutdown"`
	NewPassword string `json:"new_password"`
}

type CloudServerActionResult struct {
	ServerID uint   `json:"server_id"`
	Action   string `json:"action"`
	Message  string `json:"message"`
}

func isCloudServerSourceType(sourceType string) bool {
	t := strings.TrimSpace(strings.ToLower(sourceType))
	return t == model.ServerGroupCategoryCloud || t == "cloud"
}

func toServerItem(sv model.Server) ServerItem {
	var lastTestAt *string
	if sv.LastTestAt != nil {
		x := sv.LastTestAt.Format(time.RFC3339)
		lastTestAt = &x
	}
	var lastSeenAt *string
	if sv.LastSeenAt != nil {
		x := sv.LastSeenAt.Format(time.RFC3339)
		lastSeenAt = &x
	}
	isCloud := isCloudServerSourceType(sv.SourceType)
	return ServerItem{
		ID:                     sv.ID,
		ProjectID:              sv.ProjectID,
		GroupID:                sv.GroupID,
		Name:                   sv.Name,
		Host:                   sv.Host,
		Port:                   sv.Port,
		OSType:                 sv.OSType,
		OSArch:                 sv.OSArch,
		Tags:                   sv.Tags,
		SourceType:             sv.SourceType,
		Provider:               pickCloudString(isCloud, sv.Provider),
		CloudInstanceID:        pickCloudString(isCloud, sv.CloudInstanceID),
		CloudRegion:            pickCloudString(isCloud, sv.CloudRegion),
		CloudZone:              pickCloudString(isCloud, sv.CloudZone),
		CloudSpec:              pickCloudString(isCloud, sv.CloudSpec),
		CloudConfigInfo:        pickCloudString(isCloud, sv.CloudConfigInfo),
		CloudOSName:            pickCloudString(isCloud, sv.CloudOSName),
		CloudNetworkInfo:       pickCloudString(isCloud, sv.CloudNetworkInfo),
		CloudChargeType:        pickCloudString(isCloud, sv.CloudChargeType),
		CloudNetworkChargeType: pickCloudString(isCloud, sv.CloudNetworkChargeType),
		CloudTagsJSON:          pickCloudString(isCloud, sv.CloudTagsJSON),
		CloudPublicIP:          pickCloudString(isCloud, sv.CloudPublicIP),
		CloudPrivateIP:         pickCloudString(isCloud, sv.CloudPrivateIP),
		CloudStatusText:        pickCloudString(isCloud, sv.CloudStatusText),
		LastTestAt:             lastTestAt,
		LastTestErr:            sv.LastTestError,
		CreatedAt:              sv.CreatedAt.Format(time.RFC3339),
		LastSeenAt:             lastSeenAt,
		Status:                 sv.Status,
	}
}

func pickCloudString(isCloud bool, v string) string {
	if !isCloud {
		return ""
	}
	return v
}
