package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/projectaccess"
	"yunshu/internal/repository"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
)

type K8sClusterListQuery struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type K8sClusterCreateRequest struct {
	Name                      string        `json:"name" binding:"required,max=128"`
	ConnectionMode            string        `json:"connection_mode,omitempty" binding:"omitempty,oneof=kubeconfig direct"`
	Kubeconfig                string        `json:"kubeconfig,omitempty"`
	KubeconfigReadonly        string        `json:"kubeconfig_readonly,omitempty"`
	DirectConfig              *DirectConfig `json:"direct_config,omitempty"`
	OwningProjectID           *uint         `json:"owning_project_id"`
	ImpersonateEnabled        *bool         `json:"impersonate_enabled"`
	ImpersonateUserPrefix     string        `json:"impersonate_user_prefix"`
	RequireDestructiveConfirm *bool         `json:"require_destructive_confirm"`
}

type DirectConfig struct {
	Server                string `json:"server" binding:"omitempty,url"`
	InsecureSkipTLSVerify bool   `json:"insecure_skip_tls_verify,omitempty"`
	CAData                string `json:"ca_data,omitempty"`
	Token                 string `json:"token,omitempty"`
	Username              string `json:"username,omitempty"`
	Password              string `json:"password,omitempty"`
	ClientCertData        string `json:"client_cert_data,omitempty"`
	ClientKeyData         string `json:"client_key_data,omitempty"`
	// DictConfigKey 从数据字典读取的配置键
	DictConfigKey string `json:"dict_config_key,omitempty"`
}

type K8sClusterUpdateRequest struct {
	Name                      *string       `json:"name,omitempty" binding:"omitempty,max=128"`
	ConnectionMode            *string       `json:"connection_mode,omitempty" binding:"omitempty,oneof=kubeconfig direct"`
	Kubeconfig                *string       `json:"kubeconfig,omitempty"`
	KubeconfigReadonly        *string       `json:"kubeconfig_readonly,omitempty"`
	DirectConfig              *DirectConfig `json:"direct_config,omitempty"`
	OwningProjectID           *uint         `json:"owning_project_id"`
	ImpersonateEnabled        *bool         `json:"impersonate_enabled"`
	ImpersonateUserPrefix     *string       `json:"impersonate_user_prefix"`
	RequireDestructiveConfirm *bool         `json:"require_destructive_confirm"`
}

type K8sClusterSetStatusRequest struct {
	Status int `json:"status" binding:"oneof=0 1"`
}

type K8sClusterItem struct {
	ID                        uint          `json:"id"`
	Name                      string        `json:"name"`
	OwningProjectID           *uint         `json:"owning_project_id,omitempty"`
	ConnectionMode            string        `json:"connection_mode,omitempty"`
	Kubeconfig                string        `json:"kubeconfig,omitempty"`
	KubeconfigConfigured      bool          `json:"kubeconfig_configured,omitempty"`
	KubeconfigReadonlyConfigured bool       `json:"kubeconfig_readonly_configured,omitempty"`
	DirectConfig              *DirectConfig `json:"direct_config,omitempty"`
	ImpersonateEnabled        bool          `json:"impersonate_enabled"`
	ImpersonateUserPrefix     string        `json:"impersonate_user_prefix,omitempty"`
	RequireDestructiveConfirm bool          `json:"require_destructive_confirm"`
	Status                    int           `json:"status"`
	AccessPreset              string        `json:"access_preset,omitempty"`
	AccessRank                int           `json:"access_rank,omitempty"`
	CreatedAt                 time.Time     `json:"created_at"`
	UpdatedAt                 time.Time     `json:"updated_at"`
}

type K8sClusterListResponse struct {
	List     []K8sClusterItem `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type K8sClusterStatusResponse struct {
	ServerVersion       string    `json:"server_version"`
	ConnectionState     string    `json:"connection_state"`
	LastError           string    `json:"last_error,omitempty"`
	LastAttemptAt       time.Time `json:"last_attempt_at,omitempty"`
	LastSuccessAt       time.Time `json:"last_success_at,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
}

type NamespaceItem struct {
	Name  string `json:"name"`
	Phase string `json:"phase"`
}

type ComponentStatusItem struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Healthy     bool   `json:"healthy"`
	Message     string `json:"message,omitempty"`
	Error       string `json:"error,omitempty"`
	LastProbeAt string `json:"last_probe_at,omitempty"`
}

type K8sClusterService struct {
	repo        interfaces.K8sClusterRepository
	dictRepo    interfaces.DictEntryRepository
	runtime     *K8sRuntimeService
	nsDenyRepo  interfaces.K8sNamespaceDenyRepository
	nsAllowRepo interfaces.K8sNamespaceAllowRepository
	memberRepo  interfaces.ProjectMemberRepository
	accessRepo  interfaces.K8sClusterAccessRepository
}

// NewK8sClusterService 创建相关逻辑。
func NewK8sClusterService(
	repo interfaces.K8sClusterRepository,
	dictRepo interfaces.DictEntryRepository,
	runtime *K8sRuntimeService,
	nsDeny interfaces.K8sNamespaceDenyRepository,
	nsAllow interfaces.K8sNamespaceAllowRepository,
	memberRepo interfaces.ProjectMemberRepository,
	accessRepo interfaces.K8sClusterAccessRepository,
) *K8sClusterService {
	return &K8sClusterService{
		repo:        repo,
		dictRepo:    dictRepo,
		runtime:     runtime,
		nsDenyRepo:  nsDeny,
		nsAllowRepo: nsAllow,
		memberRepo:  memberRepo,
		accessRepo:  accessRepo,
	}
}

func (s *K8sClusterService) ensureClusterOwningProjectAccess(ctx context.Context, cl *model.K8sCluster) error {
	if cl == nil || cl.OwningProjectID == nil || *cl.OwningProjectID == 0 {
		return nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return nil
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return constants.ErrInternal
	}
	_, err := s.memberRepo.GetByProjectAndUser(ctx, *cl.OwningProjectID, u.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrK8sClusterProjectAccessDenied
		}
		return bizerrors.Pass(ctx, "k8s.cluster", "ensureClusterOwningProjectAccess", err)
	}
	return nil
}

// ensureClusterGrantAccess 非超管需至少具备只读集群档位。
func (s *K8sClusterService) ensureClusterGrantAccess(ctx context.Context, clusterID uint) error {
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return nil
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.accessRepo == nil {
		return constants.ErrForbidden
	}
	pack := k8sauth.PackFromCurrentUser(u)
	if s.accessRepo.EffectiveTier(ctx, pack, clusterID) < K8sAccessRankReadonly {
		return constants.ErrForbidden
	}
	return nil
}

func accessPresetFromRank(rank int) string {
	switch rank {
	case K8sAccessRankReadonly:
		return string(PresetK8sReadonly)
	case K8sAccessRankReadonlyExec:
		return string(PresetK8sReadonlyExec)
	case K8sAccessRankAdmin:
		return string(PresetK8sAdmin)
	default:
		return ""
	}
}

func (s *K8sClusterService) fillAccessFields(ctx context.Context, item *K8sClusterItem, clusterID uint) {
	if item == nil {
		return
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		item.AccessRank = K8sAccessRankAdmin
		item.AccessPreset = string(PresetK8sAdmin)
		return
	}
	if s.accessRepo == nil {
		return
	}
	rank := s.accessRepo.EffectiveTier(ctx, k8sauth.PackFromCurrentUser(u), clusterID)
	item.AccessRank = rank
	item.AccessPreset = accessPresetFromRank(rank)
}

func (s *K8sClusterService) validateAssignOwningProject(ctx context.Context, pid uint) error {
	if pid == 0 {
		return nil
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return constants.ErrUnauthorized
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return nil
	}
	if s.memberRepo == nil {
		return constants.ErrInternal
	}
	m, err := s.memberRepo.GetByProjectAndUser(ctx, pid, u.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrK8sClusterProjectAccessDenied
		}
		return bizerrors.Pass(ctx, "k8s.cluster", "validateAssignOwningProject", err)
	}
	if !projectaccess.RoleAtLeast(m.Role, "admin") {
		return constants.ErrProjectAdminRequired
	}
	return nil
}

// List 查询列表相关的业务逻辑。
func (s *K8sClusterService) List(ctx context.Context, query K8sClusterListQuery) (*K8sClusterListResponse, error) {
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	params := repository.K8sClusterListParams{
		Keyword:  query.Keyword,
		Page:     page,
		PageSize: pageSize,
	}
	u, ok := auth.RequestUserFromContext(ctx)
	needGrantFilter := ok && u != nil && !auth.IsSuperAdminRole(u.RoleCodes)
	if needGrantFilter && s.memberRepo != nil {
		ids, err := s.memberRepo.ListProjectIDsByUser(ctx, u.ID)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "k8s.cluster", "List", err)
		}
		params.ProjectMemberFilter = true
		params.ProjectMemberIDs = ids
	}
	// 非超管需按档位过滤：先取足量再内存过滤，保证 total/分页正确。
	if needGrantFilter {
		params.Page = 1
		params.PageSize = 1000
	}
	clusters, total, err := s.repo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	var tiersIdx repository.EffectiveTierIndex
	if needGrantFilter && s.accessRepo != nil {
		idx, err := s.accessRepo.BuildEffectiveTierIndex(ctx, k8sauth.PackFromCurrentUser(u))
		if err != nil {
			return nil, bizerrors.Pass(ctx, "k8s.cluster", "List", err)
		}
		tiersIdx = idx
		filtered := make([]model.K8sCluster, 0, len(clusters))
		for _, c := range clusters {
			if tiersIdx.ClusterAccessible(c.ID, K8sAccessRankReadonly) {
				filtered = append(filtered, c)
			}
		}
		clusters = filtered
		total = int64(len(clusters))
		start := (page - 1) * pageSize
		if start > len(clusters) {
			start = len(clusters)
		}
		end := start + pageSize
		if end > len(clusters) {
			end = len(clusters)
		}
		clusters = clusters[start:end]
	}

	items := make([]K8sClusterItem, 0, len(clusters))
	for _, c := range clusters {
		item := s.buildClusterItem(c, false)
		if needGrantFilter {
			rank := 0
			if s.accessRepo != nil {
				rank = tiersIdx.GlobalRank
				if r, ok := tiersIdx.PerCluster[c.ID]; ok && r > rank {
					rank = r
				}
			}
			item.AccessRank = rank
			item.AccessPreset = accessPresetFromRank(rank)
		} else if ok && u != nil && auth.IsSuperAdminRole(u.RoleCodes) {
			item.AccessRank = K8sAccessRankAdmin
			item.AccessPreset = string(PresetK8sAdmin)
		}
		items = append(items, item)
	}
	return &K8sClusterListResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Create 创建相关的业务逻辑。
func (s *K8sClusterService) Create(ctx context.Context, req K8sClusterCreateRequest) (*K8sClusterItem, error) {
	// 设置默认连接模式
	connectionMode := req.ConnectionMode
	if connectionMode == "" {
		connectionMode = "kubeconfig"
	}
	if connectionMode == "kubeconfig" && strings.TrimSpace(req.Kubeconfig) == "" {
		return nil, constants.ErrBadRequestWithMsg("kubeconfig 模式必须提供 kubeconfig 内容")
	}
	if connectionMode == "direct" {
		if req.DirectConfig == nil || strings.TrimSpace(req.DirectConfig.Server) == "" {
			if req.DirectConfig == nil || strings.TrimSpace(req.DirectConfig.DictConfigKey) == "" {
				return nil, constants.ErrBadRequestWithMsg("直连模式必须提供 API Server 或数据字典配置键")
			}
		}
	}

	c := &model.K8sCluster{
		Name:                      req.Name,
		ConnectionMode:            connectionMode,
		Status:                    1,
		RequireDestructiveConfirm: true,
		ImpersonateEnabled:        false,
		ImpersonateUserPrefix:     "yunshu:",
	}
	if req.RequireDestructiveConfirm != nil {
		c.RequireDestructiveConfirm = *req.RequireDestructiveConfirm
	}
	// Impersonation 已下线：忽略请求中的 impersonate_*，一律关闭。

	// 处理直连配置
	if connectionMode == "direct" && req.DirectConfig != nil {
		// 如果从字典读取配置
		if req.DirectConfig.DictConfigKey != "" && s.dictRepo != nil {
			dictConfig, err := getDirectConfigFromDict(ctx, s.dictRepo, req.DirectConfig.DictConfigKey)
			if err != nil {
				return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmte5d845e17676, err))
			}
			// 合并字典配置和用户配置（用户配置优先）
			mergeDirectConfig(dictConfig, req.DirectConfig)
			req.DirectConfig = dictConfig
		}

		directConfigJSON, err := json.Marshal(req.DirectConfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg(constants.ErrMsg2569b002d990)
		}
		sealedDC, err := s.runtime.SealCredential(string(directConfigJSON))
		if err != nil {
			return nil, constants.ErrInternalWithMsg("加密直连配置失败")
		}
		c.DirectConfig = sealedDC
		// 为直连模式生成兼容的kubeconfig（库内仍存密封副本，运行时以 DirectConfig 为准）
		kubeconfig, err := buildKubeconfigFromDirectConfig(req.DirectConfig)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmt92e759c1fa53, err))
		}
		sealedKC, err := s.runtime.SealCredential(kubeconfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg("加密 kubeconfig 失败")
		}
		c.Kubeconfig = sealedKC
	} else {
		sealedKC, err := s.runtime.SealCredential(req.Kubeconfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg("加密 kubeconfig 失败")
		}
		c.Kubeconfig = sealedKC
	}

	if ro := strings.TrimSpace(req.KubeconfigReadonly); ro != "" {
		sealedRO, err := s.runtime.SealCredential(ro)
		if err != nil {
			return nil, constants.ErrInternalWithMsg("加密只读 kubeconfig 失败")
		}
		c.KubeconfigReadonly = sealedRO
	}

	if req.OwningProjectID != nil && *req.OwningProjectID > 0 {
		if err := s.validateAssignOwningProject(ctx, *req.OwningProjectID); err != nil {
			return nil, err
		}
		v := *req.OwningProjectID
		c.OwningProjectID = &v
	}

	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return &K8sClusterItem{
		ID:              c.ID,
		Name:            c.Name,
		OwningProjectID: c.OwningProjectID,
		ConnectionMode:  c.ConnectionMode,
		Status:          c.Status,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}, nil
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sClusterService) Detail(ctx context.Context, id uint) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cluster); err != nil {
		return nil, err
	}
	if err := s.ensureClusterGrantAccess(ctx, id); err != nil {
		return nil, err
	}
	item := s.buildClusterItem(*cluster, true)
	s.fillAccessFields(ctx, &item, id)
	return &item, nil
}

// Update 更新相关的业务逻辑。
func (s *K8sClusterService) Update(ctx context.Context, id uint, req K8sClusterUpdateRequest) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cluster); err != nil {
		return nil, err
	}
	if req.Name != nil {
		cluster.Name = *req.Name
	}

	// 处理连接模式变更
	if req.ConnectionMode != nil {
		cluster.ConnectionMode = *req.ConnectionMode
		if *req.ConnectionMode == "kubeconfig" {
			cluster.DirectConfig = ""
		}
	}

	// 处理直连配置更新
	if cluster.ConnectionMode == "direct" && req.DirectConfig != nil {
		storedPlain, err := s.runtime.OpenCredential(cluster.DirectConfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg("解密已存直连配置失败")
		}
		preserveDirectAuthFromStored(storedPlain, req.DirectConfig)
		// 如果从字典读取配置
		if req.DirectConfig.DictConfigKey != "" && s.dictRepo != nil {
			dictConfig, err := getDirectConfigFromDict(ctx, s.dictRepo, req.DirectConfig.DictConfigKey)
			if err != nil {
				return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmte5d845e17676, err))
			}
			// 合并字典配置和用户配置（用户配置优先）
			mergeDirectConfig(dictConfig, req.DirectConfig)
			req.DirectConfig = dictConfig
		}

		directConfigJSON, err := json.Marshal(req.DirectConfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg(constants.ErrMsg2569b002d990)
		}
		sealedDC, err := s.runtime.SealCredential(string(directConfigJSON))
		if err != nil {
			return nil, constants.ErrInternalWithMsg("加密直连配置失败")
		}
		cluster.DirectConfig = sealedDC

		// 为直连模式生成兼容的kubeconfig
		kubeconfig, err := buildKubeconfigFromDirectConfig(req.DirectConfig)
		if err != nil {
			return nil, constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmt92e759c1fa53, err))
		}
		sealedKC, err := s.runtime.SealCredential(kubeconfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg("加密 kubeconfig 失败")
		}
		cluster.Kubeconfig = sealedKC
		s.runtime.DeleteRegisterCache(cluster.ID)
	} else if req.Kubeconfig != nil {
		sealedKC, err := s.runtime.SealCredential(*req.Kubeconfig)
		if err != nil {
			return nil, constants.ErrInternalWithMsg("加密 kubeconfig 失败")
		}
		cluster.Kubeconfig = sealedKC
		s.runtime.DeleteRegisterCache(cluster.ID)
	}

	if req.KubeconfigReadonly != nil {
		ro := strings.TrimSpace(*req.KubeconfigReadonly)
		if ro == "" {
			cluster.KubeconfigReadonly = ""
		} else {
			sealedRO, err := s.runtime.SealCredential(ro)
			if err != nil {
				return nil, constants.ErrInternalWithMsg("加密只读 kubeconfig 失败")
			}
			cluster.KubeconfigReadonly = sealedRO
		}
		s.runtime.DeleteRegisterCache(cluster.ID)
	}
	// Impersonation 已下线：更新时强制关闭，清理历史开启状态
	if cluster.ImpersonateEnabled {
		cluster.ImpersonateEnabled = false
		s.runtime.DeleteRegisterCache(cluster.ID)
	}
	if req.RequireDestructiveConfirm != nil {
		cluster.RequireDestructiveConfirm = *req.RequireDestructiveConfirm
	}

	if req.OwningProjectID != nil {
		if *req.OwningProjectID == 0 {
			cluster.OwningProjectID = nil
		} else {
			if err := s.validateAssignOwningProject(ctx, *req.OwningProjectID); err != nil {
				return nil, err
			}
			v := *req.OwningProjectID
			cluster.OwningProjectID = &v
		}
	}

	if err := s.repo.Update(ctx, cluster); err != nil {
		return nil, err
	}
	out := s.buildClusterItem(*cluster, true)
	return &out, nil
}

// Delete 删除相关的业务逻辑。
func (s *K8sClusterService) Delete(ctx context.Context, id uint) error {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return k8sRepoErr(ctx, "k8s.cluster", "Delete", err, "cluster_id", id)
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cl); err != nil {
		return err
	}
	s.runtime.DeleteRegisterCache(id)
	return s.repo.Delete(ctx, id)
}

// SetStatus 设置相关的业务逻辑。
func (s *K8sClusterService) SetStatus(ctx context.Context, id uint, status int) (*K8sClusterItem, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cluster); err != nil {
		return nil, err
	}
	if status != 0 && status != 1 {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg394db01d16f3)
	}
	if cluster.Status != status {
		cluster.Status = status
		// 停用后立即让 kom 重新注册失效，避免继续复用旧连接
		if status == 0 {
			s.runtime.DeleteRegisterCache(id)
		}
		if err := s.repo.Update(ctx, cluster); err != nil {
			return nil, err
		}
	}
	out := s.buildClusterItem(*cluster, true)
	return &out, nil
}

func (s *K8sClusterService) buildClusterItem(c model.K8sCluster, forDetail bool) K8sClusterItem {
	item := K8sClusterItem{
		ID:                        c.ID,
		Name:                      c.Name,
		OwningProjectID:           c.OwningProjectID,
		ConnectionMode:            c.ConnectionMode,
		Status:                    c.Status,
		ImpersonateEnabled:        false,
		ImpersonateUserPrefix:     "",
		RequireDestructiveConfirm: c.RequireDestructiveConfirm,
		CreatedAt:                 c.CreatedAt,
		UpdatedAt:                 c.UpdatedAt,
	}
	if strings.TrimSpace(c.Kubeconfig) != "" {
		item.KubeconfigConfigured = true
	}
	if strings.TrimSpace(c.KubeconfigReadonly) != "" {
		item.KubeconfigReadonlyConfigured = true
	}
	if c.ConnectionMode == "direct" && strings.TrimSpace(c.DirectConfig) != "" {
		raw := c.DirectConfig
		if s != nil && s.runtime != nil {
			if plain, err := s.runtime.OpenCredential(c.DirectConfig); err == nil {
				raw = plain
			}
		}
		var dc DirectConfig
		if err := json.Unmarshal([]byte(raw), &dc); err == nil {
			item.DirectConfig = maskDirectConfigForAPI(&dc)
		}
	}
	if forDetail && item.KubeconfigConfigured {
		// 不回传完整 kubeconfig，避免凭证经 API 泄露；更新时重新粘贴即可。
		item.Kubeconfig = ""
	}
	return item
}

// Status 探测连通性：失败时仍返回状态体（含 last_error），避免前端只看到笼统 500。
func (s *K8sClusterService) Status(ctx context.Context, id uint) (*K8sClusterStatusResponse, error) {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cl); err != nil {
		return nil, err
	}
	ver, state, err := s.runtime.CheckClusterHeartbeat(ctx, id)
	resp := &K8sClusterStatusResponse{
		ServerVersion:       ver,
		ConnectionState:     state.State,
		LastError:           state.LastError,
		LastAttemptAt:       state.LastAttemptAt,
		LastSuccessAt:       state.LastSuccessAt,
		ConsecutiveFailures: state.ConsecutiveFailures,
	}
	if err != nil {
		if strings.TrimSpace(resp.LastError) == "" {
			resp.LastError = classifyClusterConnectError(err)
		}
		if resp.ConnectionState == "" || resp.ConnectionState == "unknown" || resp.ConnectionState == "ready" {
			resp.ConnectionState = "degraded"
		}
		return resp, nil
	}
	return resp, nil
}

// ListNamespaces 查询列表相关的业务逻辑；若传入 pack，则按命名空间黑/白名单过滤（与控制台下拉、K8sScopeAuthorize 对齐）。
func (s *K8sClusterService) ListNamespaces(ctx context.Context, id uint, pack *k8sauth.PrincipalPack) ([]NamespaceItem, error) {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cl); err != nil {
		return nil, err
	}
	nsList, err := s.runtime.ListNamespacesViaKom(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]NamespaceItem, 0, len(nsList))
	for _, ns := range nsList {
		out = append(out, NamespaceItem{Name: ns.Name, Phase: string(ns.Status.Phase)})
	}
	if pack == nil || len(pack.PrincipalRows()) == 0 {
		return out, nil
	}
	names := make([]string, len(out))
	for i := range out {
		names[i] = out[i].Name
	}
	names, err = FilterNamespaceNamesByPolicy(ctx, s.nsDenyRepo, s.nsAllowRepo, *pack, id, names)
	if err != nil {
		return nil, err
	}
	keep := make(map[string]struct{}, len(names))
	for _, n := range names {
		keep[n] = struct{}{}
	}
	filtered := out[:0]
	for _, it := range out {
		if _, ok := keep[it.Name]; ok {
			filtered = append(filtered, it)
		}
	}
	return filtered, nil
}

// ListComponentStatuses 查询列表相关的业务逻辑。
func (s *K8sClusterService) ListComponentStatuses(ctx context.Context, id uint) ([]ComponentStatusItem, error) {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.ensureClusterOwningProjectAccess(ctx, cl); err != nil {
		return nil, err
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, id)
	if err != nil {
		return nil, err
	}
	probedAt := time.Now().Format(time.RFC3339)
	out := make([]ComponentStatusItem, 0)

	var nodes []corev1.Node
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).List(&nodes).Error; err != nil {
		return nil, bizerrors.Internalf(ctx, "k8s.cluster", "api", err, constants.ErrFmt559cb56d5b9d)
	}
	for _, n := range nodes {
		out = append(out, nodeToComponentStatusItem(n, probedAt))
	}

	var pods []corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace("kube-system").List(&pods).Error; err == nil {
		for _, p := range pods {
			if isKubeControlPlanePod(p) {
				out = append(out, podToComponentStatusItem(p, probedAt))
			}
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}
