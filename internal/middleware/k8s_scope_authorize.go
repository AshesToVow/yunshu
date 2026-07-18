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

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/k8sauth"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/response"
	"yunshu/internal/interfaces"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

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

func (c *k8sScopeCatalogCache) get(routePath, method string) (string, bool) {
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
		c.refresh()
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

func (c *k8sScopeCatalogCache) permissions() []model.Permission {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.perms == nil {
		return nil
	}
	out := make([]model.Permission, len(c.perms))
	copy(out, c.perms)
	return out
}

func (c *k8sScopeCatalogCache) unavailable() bool {
	if c == nil || c.repo == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.loadedAt.IsZero() && c.loadErr != nil
}

func (c *k8sScopeCatalogCache) refresh() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.repo == nil {
		c.actionByKey = map[string]string{}
		c.scopedKeys = map[string]bool{}
		c.perms = nil
		c.loadedAt = time.Now()
		return
	}
	perms, err := c.repo.ListAll(context.Background())
	if err != nil {
		c.loadErr = err
		logx.With(context.Background(), "component", "http.k8s_scope").Error("permission catalog refresh failed", "error", err)
		return
	}
	c.loadErr = nil
	actionByKey, scopedKeys := service.BuildK8sScopeMappings(perms)
	c.actionByKey = actionByKey
	c.scopedKeys = scopedKeys
	c.perms = perms
	c.loadedAt = time.Now()
}

// K8sScopeAuthorize:
// 1) 只要请求带具体 namespace：始终做 NS deny/allow（含未纳入目录的 GET；super-admin 仍受约束）
// 2) 纳入目录 / forceTier / K8s 只读路径：再做集群档位校验
// 3) 档位由 DB 表 k8s_cluster_access_grants 控制（不经 Casbin）
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
		actionCode, tracked := catalog.get(routePath, method)
		forceTier := k8sScopeForceTierCheck(routePath, method)
		needTier := tracked || forceTier ||
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
		c.Request = c.Request.WithContext(k8sauth.WithRequestScope(c.Request.Context(), k8sauth.RequestScope{
			ClusterID: clusterID,
			Namespace: concreteNS,
			Pack:      pack,
		}))

		if clusterID == 0 {
			if forceTier {
				msg := "须在请求中携带 cluster_id，以便集群档位校验"
				if strings.HasSuffix(strings.TrimSpace(routePath), "/ingresses/nginx/restart") {
					msg = "重启 Ingress-Nginx 须在请求体中携带 cluster_id，且需集群 admin 档位"
				} else if strings.Contains(strings.TrimSpace(routePath), "exec") {
					msg = "Pod Exec 须在请求中携带 cluster_id（及 namespace），以便集群档位校验"
				}
				response.Error(c, constants.ErrBadRequestWithMsg(msg))
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if concreteNS == "" {
			if forceTier && strings.Contains(strings.TrimSpace(routePath), "exec") {
				response.Error(c, constants.ErrBadRequestWithMsg("Pod Exec 须指定 namespace"))
				c.Abort()
				return
			}
		}

		if needNS {
			if nsDenyRepo != nil {
				denied, err := nsDenyRepo.IsDenied(c.Request.Context(), pack, clusterID, concreteNS)
				if err != nil {
					logx.With(c.Request.Context(), "component", "http.k8s_scope").Error("namespace deny check failed", "error", err)
					response.Error(c, constants.ErrInternal)
					c.Abort()
					return
				}
				if denied {
					response.Error(c, constants.ErrForbiddenWithMsg("当前主体在此集群下禁止访问命名空间「"+concreteNS+"」"))
					c.Abort()
					return
				}
			}
			if nsAllowRepo != nil {
				active, err := nsAllowRepo.WhitelistActiveForCluster(c.Request.Context(), pack, clusterID)
				if err != nil {
					logx.With(c.Request.Context(), "component", "http.k8s_scope").Error("namespace allow check failed", "error", err)
					response.Error(c, constants.ErrInternal)
					c.Abort()
					return
				}
				if active {
					ok, err := nsAllowRepo.NamespaceAllowed(c.Request.Context(), pack, clusterID, concreteNS)
					if err != nil {
						logx.With(c.Request.Context(), "component", "http.k8s_scope").Error("namespace allow match failed", "error", err)
						response.Error(c, constants.ErrInternal)
						c.Abort()
						return
					}
					if !ok {
						response.Error(c, constants.ErrForbiddenWithMsg("当前主体在此集群下仅允许访问白名单内的命名空间"))
						c.Abort()
						return
					}
				}
			}
		}

		if !needTier {
			c.Next()
			return
		}

		for _, rc := range user.RoleCodes {
			if strings.TrimSpace(rc) == "super-admin" {
				c.Next()
				return
			}
		}

		if catalog.unavailable() {
			response.Error(c, constants.ErrInternal)
			c.Abort()
			return
		}
		if accessRepo == nil {
			response.Error(c, constants.ErrInternal)
			c.Abort()
			return
		}

		perms := catalog.permissions()
		required := service.RequiredK8sAccessRank(perms, routePath, method, actionCode)
		if required <= 0 {
			required = service.K8sAccessRankAdmin
		}

		rank := accessRepo.EffectiveTier(c.Request.Context(), pack, clusterID)
		if rank < required {
			response.Error(c, constants.ErrForbidden)
			c.Abort()
			return
		}

		c.Next()
	}
}

// k8sScopeForceTierCheck Pod Exec 等为高危：无论 API 管理是否勾选「纳入 K8s 范围校验」，均按集群档位与命名空间策略校验（仍需 Casbin 授权）。
func k8sScopeForceTierCheck(routePath, method string) bool {
	p := strings.TrimSpace(routePath)
	m := strings.ToUpper(strings.TrimSpace(method))
	switch m {
	case "POST":
		return strings.HasSuffix(p, "/pods/exec") || strings.HasSuffix(p, "/ingresses/nginx/restart")
	case "GET":
		return strings.HasSuffix(p, "/pods/exec/ws")
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
