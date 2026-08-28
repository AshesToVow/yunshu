package apiroute

import (
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// Entry 表示一条需登记到权限目录的 HTTP 路由。
type Entry struct {
	Method string
	Path   string
}

const autoSyncDescPrefix = "auto-synced"

// AutoSyncDescription 标记由路由扫描自动登记的权限描述。
func AutoSyncDescription(pluginName string) string {
	pluginName = strings.TrimSpace(pluginName)
	if pluginName == "" {
		return autoSyncDescPrefix
	}
	return autoSyncDescPrefix + ": " + pluginName
}

// IsAutoSyncedDescription 判断权限描述是否来自路由自动登记。
func IsAutoSyncedDescription(desc string) bool {
	return strings.HasPrefix(strings.TrimSpace(desc), autoSyncDescPrefix)
}

// Collect 从 Gin 引擎收集已注册路由（仅含带 path 模板的业务路由）。
func Collect(engine *gin.Engine) []Entry {
	if engine == nil {
		return nil
	}
	routes := engine.Routes()
	out := make([]Entry, 0, len(routes))
	seen := map[string]struct{}{}
	for _, rt := range routes {
		method := strings.ToUpper(strings.TrimSpace(rt.Method))
		path := normalizePath(rt.Path)
		if method == "" || path == "" || ShouldSkip(method, path) {
			continue
		}
		key := method + " " + path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, Entry{Method: method, Path: path})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path == out[j].Path {
			return out[i].Method < out[j].Method
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// ShouldSkip 跳过无需写入权限目录的路由（公开探活、登录、Swagger、Webhook 等）。
func ShouldSkip(method, path string) bool {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizePath(path)
	if path == "" {
		return true
	}
	if !strings.HasPrefix(path, "/api/v1") {
		return true
	}
	for _, rule := range skipRules {
		if rule.match(method, path) {
			return true
		}
	}
	return false
}

// DefaultName 根据 method + path 生成可读权限名称。
func DefaultName(method, path string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizePath(path)
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) >= 3 && segments[0] == "api" && segments[1] == "v1" {
		segments = segments[2:]
	}
	label := strings.Join(segments, " ")
	label = strings.NewReplacer(":", "", "-", " ").Replace(label)
	switch method {
	case "GET":
		return "查看 " + label
	case "POST":
		return "创建/提交 " + label
	case "PUT", "PATCH":
		return "更新 " + label
	case "DELETE":
		return "删除 " + label
	default:
		return method + " " + label
	}
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	// Gin 路由模板统一为 :param
	path = strings.ReplaceAll(path, "*filepath", ":filepath")
	return path
}

type skipRule struct {
	method string
	path   string
	prefix bool
}

func (r skipRule) match(method, path string) bool {
	if r.method != "" && r.method != method {
		return false
	}
	if r.prefix {
		return strings.HasPrefix(path, r.path)
	}
	return path == r.path
}

var skipRules = []skipRule{
	{method: "GET", path: "/api/v1/health"},
	{method: "GET", path: "/api/v1/ready"},
	{method: "GET", path: "/api/v1/menus/tree"},
	{method: "POST", path: "/api/v1/auth/", prefix: true},
	{method: "GET", path: "/api/v1/auth/password-policy"},
	{method: "POST", path: "/api/v1/cicd/jenkins/callback"},
	{method: "POST", path: "/api/v1/alerts/webhook/"},
	{method: "POST", path: "/api/v1/loggie/", prefix: true},
	{method: "GET", path: "/swagger/"},
}
