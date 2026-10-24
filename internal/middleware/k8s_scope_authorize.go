package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8srole"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

// k8sScopeCatalogLoadTimeout 限定权限目录刷新的单次查询上界，避免慢 DB 拖垮请求链路。
const k8sScopeCatalogLoadTimeout = 5 * time.Second

type k8sScopeCatalogCache struct {
	repo        interfaces.PermissionRepository
	ttl         time.Duration
	mu          sync.RWMutex
	loadedAt    time.Time
	loadErr     error
	actionByKey map[string]string
	scopedKeys  map[string]bool
	perms       []model.Permission
}

func newK8sScopeCatalogCache(repo interfaces.PermissionRepository) *k8sScopeCatalogCache {
	return &k8sScopeCatalogCache{
		repo: repo,
		ttl:  60 * time.Second,
	}
}

func (c *k8sScopeCatalogCache) get(ctx context.Context, routePath, method string) (string, bool) {
	if c == nil {
		return "", false
	}
	now := time.Now()
	c.mu.RLock()
	ready := !c.loadedAt.IsZero() && now.Sub(c.loadedAt) < c.ttl
	actionByKey := c.actionByKey
	scopedKeys := c.scopedKeys
	c.mu.RUnlock()
	if !ready {
		c.refresh(ctx)
		c.mu.RLock()
		actionByKey = c.actionByKey
		scopedKeys = c.scopedKeys
		c.mu.RUnlock()
	}
	key := service.K8sScopeRouteKey(routePath, method)
	if scopedKeys == nil || !scopedKeys[key] {
		return "", false
	}
	return strings.TrimSpace(actionByKey[key]), true
}

func (c *k8sScopeCatalogCache) refresh(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.repo == nil {
		c.actionByKey = map[string]string{}
		c.scopedKeys = map[string]bool{}
		c.perms = nil
		c.loadedAt = time.Now()
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// 目录刷新走请求链路 ctx（保留 request_id / user 等日志字段），并加超时上界避免拖垮请求
	loadCtx, cancel := context.WithTimeout(ctx, k8sScopeCatalogLoadTimeout)
	defer cancel()
	perms, err := c.repo.ListAll(loadCtx)
	if err != nil {
		c.loadErr = err
		logx.With(loadCtx, "component", "http.k8s_scope").Error("permission catalog refresh failed", "error", err)
		return
	}
	c.loadErr = nil
	actionByKey, scopedKeys := service.BuildK8sScopeMappings(perms)
	c.actionByKey = actionByKey
	c.scopedKeys = scopedKeys
	c.perms = perms
	c.loadedAt = time.Now()
}

// K8sScopeAuthorize 对齐 k8m CheckPermissionLogic：
// 1) 解析 cluster_id / namespace 后加载 k8s_cluster_access_grants
// 2) 平台超管旁路（与 k8m RolePlatformAdmin 一致）
// 3) 其余用户按三角色 + 授权行上的 NS 黑白名单校验（写操作含重启均需 cluster_admin）
func K8sScopeAuthorize(
	logger *logx.Logger,
	permRepo interfaces.PermissionRepository,
	accessRepo interfaces.K8sClusterAccessRepository,
	nsDenyRepo interfaces.K8sNamespaceDenyRepository,
	nsAllowRepo interfaces.K8sNamespaceAllowRepository,
) gin.HandlerFunc {
	catalog := newK8sScopeCatalogCache(permRepo)
	return func(c *gin.Context) {
		routePath := c.FullPath()
		if strings.TrimSpace(routePath) == "" {
			routePath = c.Request.URL.Path
		}
		method := c.Request.Method
		_, tracked := catalog.get(c.Request.Context(), routePath, method)
		forceTier := k8sScopeForceTierCheck(routePath, method)
		grantRequiredWrite := k8sScopeRequiresClusterGrant(routePath, method)
		needTier := tracked || forceTier || grantRequiredWrite ||
			(strings.EqualFold(method, "GET") && service.IsK8sReadAPIPath(routePath))

		clusterID, namespace := extractClusterNamespaceFromRequest(c)
		concreteNS := strings.TrimSpace(namespace)
		needNS := clusterID > 0 && concreteNS != "" && concreteNS != "_cluster"

		if !needTier && !needNS {
			c.Next()
			return
		}

		user, ok := auth.CurrentUserFromContext(c)
		if !ok {
			response.Error(c, constants.ErrNotLoggedIn)
			c.Abort()
			return
		}
		pack := k8sauth.PackFromCurrentUser(user)
		intent := k8sauth.AccessIntentWrite
		if strings.EqualFold(method, "GET") && service.IsK8sReadAPIPath(routePath) && !forceTier {
			intent = k8sauth.AccessIntentRead
		}
		if forceTier && k8sRouteIsExecLike(routePath) {
			intent = k8sauth.AccessIntentExec
		}
		if strings.EqualFold(method, "GET") && !forceTier && !tracked {
			intent = k8sauth.AccessIntentRead
		}
		bypass := auth.IsSuperAdminRole(user.RoleCodes)
		scope := k8sauth.RequestScope{
			ClusterID:  clusterID,
			Namespace:  concreteNS,
			Pack:       pack,
			Intent:     intent,
			BypassAuth: bypass,
			Username:   strings.TrimSpace(user.Username),
		}
		if scope.Username == "" {
			scope.Username = k8sauth.UserRefString(user.ID)
		}
		c.Request = c.Request.WithContext(k8sauth.WithRequestScope(c.Request.Context(), scope))
		// 从 body 提取 confirm 供高危操作
		if m := extractConfirmFromRequest(c); m {
			c.Request = c.Request.WithContext(k8sauth.WithDestructiveConfirm(c.Request.Context(), true))
		}

		if clusterID == 0 {
			if forceTier || grantRequiredWrite {
				msg := "须在请求中携带 cluster_id，以便集群档位校验"
				if strings.HasSuffix(strings.TrimSpace(routePath), "/ingresses/nginx/restart") {
					msg = "重启 Ingress-Nginx 须在请求体中携带 cluster_id，且需集群 admin 档位"
				} else if k8sRouteIsExecLike(routePath) {
					msg = "Pod Exec / 文件操作须在请求中携带 cluster_id（及 namespace），以便集群档位校验"
				}
				response.Error(c, constants.ErrBadRequestWithMsg(msg))
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if concreteNS == "" {
			if forceTier && k8sRouteIsExecLike(routePath) {
				response.Error(c, constants.ErrBadRequestWithMsg("Pod Exec / 文件操作须指定 namespace"))
				c.Abort()
				return
			}
		}

		if bypass {
			c.Next()
			return
		}
		if accessRepo == nil {
			response.Error(c, constants.ErrInternal)
			c.Abort()
			return
		}
		grants, err := accessRepo.ListForPack(c.Request.Context(), pack, clusterID)
		if err != nil {
			logx.With(c.Request.Context(), "component", "http.k8s_scope").Error("list k8s grants failed", "error", err)
			response.Error(c, constants.ErrInternal)
			c.Abort()
			return
		}
		scope.Grants = grants
		scope.AccessRank = k8srole.RankFromGrants(grants)
		c.Request = c.Request.WithContext(k8sauth.WithRequestScope(c.Request.Context(), scope))

		var nsList []string
		if needNS {
			nsList = []string{concreteNS}
		}
		action := k8srole.ActionFromHTTP(method, routePath)
		if err := k8srole.Check(grants, strconv.FormatUint(uint64(clusterID), 10), scope.Username, action, nsList); err != nil {
			response.Error(c, constants.ErrForbiddenWithMsg(err.Error()))
			c.Abort()
			return
		}
		c.Next()
	}
}

// k8sScopeRequiresClusterGrant 将 K8s 资源写操作统一纳入集群档位校验，避免未登记到权限目录的路由绕过授权。
// /clusters 下的接口管理平台中的集群连接记录，由 Casbin 控制，不属于目标集群资源操作。
func k8sScopeRequiresClusterGrant(routePath, method string) bool {
	p := strings.TrimSpace(routePath)
	if strings.Contains(p, "/clusters") {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

// k8sScopeForceTierCheck 高危写操作强制集群档位校验（仍需 Casbin 授权）。
func k8sScopeForceTierCheck(routePath, method string) bool {
	p := strings.TrimSpace(routePath)
	m := strings.ToUpper(strings.TrimSpace(method))
	switch m {
	case "POST":
		if strings.HasSuffix(p, "/pods/exec") ||
			strings.HasSuffix(p, "/pods/debug") ||
			strings.HasSuffix(p, "/ingresses/nginx/restart") ||
			strings.HasSuffix(p, "/nodes/drain") ||
			strings.HasSuffix(p, "/rbac/apply") {
			return true
		}
		if strings.Contains(p, "/helm/") && (strings.HasSuffix(p, "/uninstall") || strings.HasSuffix(p, "/install") || strings.HasSuffix(p, "/upgrade")) {
			return true
		}
		if strings.HasSuffix(p, "/pods/file/upload") || strings.HasSuffix(p, "/pods/file/delete") {
			return true
		}
		return false
	case "DELETE":
		return strings.Contains(p, "/helm/")
	case "GET":
		return strings.HasSuffix(p, "/pods/exec/ws") || strings.HasSuffix(p, "/secrets/reveal")
	default:
		return false
	}
}

// k8sRouteIsExecLike Pod 终端与容器文件变更同属 exec 档位意图。
func k8sRouteIsExecLike(routePath string) bool {
	p := strings.TrimSpace(routePath)
	if p == "" {
		return false
	}
	if strings.Contains(p, "exec") || strings.HasSuffix(p, "/pods/debug") {
		return true
	}
	return strings.Contains(p, "/pods/file/upload") || strings.Contains(p, "/pods/file/delete")
}

func extractConfirmFromRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("confirm")), "true") || strings.TrimSpace(c.Query("confirm")) == "1" {
		return true
	}
	if c.Request.Body == nil {
		return false
	}
	raw, _ := io.ReadAll(c.Request.Body)
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if len(raw) == 0 {
		return false
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return false
	}
	switch v := m["confirm"].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
	case float64:
		return v != 0
	default:
		return false
	}
}

func extractClusterNamespaceFromRequest(c *gin.Context) (uint, string) {
	clusterID := parseUint(strings.TrimSpace(c.Query("cluster_id")))
	namespace := strings.TrimSpace(c.Query("namespace"))

	if clusterID == 0 {
		clusterID = parseUint(strings.TrimSpace(c.Param("id")))
	}

	if (clusterID == 0 || namespace == "") && c.Request.Body != nil {
		raw, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		if len(raw) > 0 {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err == nil {
				if clusterID == 0 {
					clusterID = toUint(m["cluster_id"])
					if clusterID == 0 {
						clusterID = toUint(m["clusterId"])
					}
				}
				if namespace == "" {
					if s, ok := m["namespace"].(string); ok {
						namespace = strings.TrimSpace(s)
					}
				}
			}
		}
	}

	if clusterID == 0 {
		clusterID = parseUint(strings.TrimSpace(c.PostForm("cluster_id")))
	}
	if namespace == "" {
		namespace = strings.TrimSpace(c.PostForm("namespace"))
	}

	return clusterID, namespace
}

func parseUint(v string) uint {
	if strings.TrimSpace(v) == "" {
		return 0
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

func toUint(v any) uint {
	switch t := v.(type) {
	case float64:
		return uint(t)
	case int:
		return uint(t)
	case int64:
		return uint(t)
	case uint:
		return t
	case string:
		return parseUint(t)
	default:
		return 0
	}
}
