// Package permcatalog 是平台 API 权限清单的唯一来源。
//
// 路由注册表仍是「接口是否存在」的权威（启动期扫描 + 孤儿回收），本包只提供
// 人类可读的权限名、分组与描述，供 seed、启动同步、权限目录接口与前端目录生成共用，
// 避免 seed / 路由扫描 / 前端静态目录三份清单各写一遍互相漂移。
package permcatalog

import (
	"strings"
	"sync"

	"yunshu/internal/model"
)

// Entry 一条 API 权限目录项。
type Entry struct {
	Group       string `json:"group"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	// Auth 是否需要登录鉴权；公开接口（探活/登录/回调）为 false。
	Auth bool `json:"auth"`
}

// Group 按业务域分组的权限清单。
type Group struct {
	Title   string           `json:"title"`
	Entries []model.Permission `json:"entries"`
}

// Groups 返回可 seed、可授权的权限分组（不含无 Casbin 鉴权的公开接口）。
func Groups() []Group {
	return []Group{
		{Title: "系统", Entries: systemEntries()},
		{Title: "K8s 集群与资源", Entries: k8sEntries()},
		{Title: "告警中心", Entries: alertEntries()},
		{Title: "项目管理", Entries: projectEntries()},
		{Title: "CI/CD", Entries: cicdEntries()},
		{Title: "数据库管理", Entries: dbmgmtEntries()},
		{Title: "AI 能力", Entries: aiEntries()},
		{Title: "日志", Entries: logEntries()},
		{Title: "Kafka", Entries: kafkamgmtEntries()},
		{Title: "巡检", Entries: inspectEntries()},
	}
}

// AllPermissions 返回全部可授权权限点（seed 与目录接口共用）。
func AllPermissions() []model.Permission {
	groups := Groups()
	out := make([]model.Permission, 0, 720)
	for _, g := range groups {
		out = append(out, g.Entries...)
	}
	return out
}

// publicEntries 无 Casbin 鉴权的公开/登录态接口，仅供目录展示，不参与授权与 seed。
func publicEntries() []Entry {
	return []Entry{
		{Name: "健康检查", Description: "进程状态与运行信息", Method: "GET", Path: "/api/v1/health"},
		{Name: "就绪探活", Description: "兼容旧探针的就绪检查", Method: "GET", Path: "/api/v1/ready"},
		{Name: "发送邮箱验证码", Description: "注册/找回密码验证码", Method: "POST", Path: "/api/v1/auth/verification-code"},
		{Name: "用户名发登录验证码", Description: "向绑定邮箱发送登录验证码", Method: "POST", Path: "/api/v1/auth/login-code"},
		{Name: "图形验证码", Description: "获取登录图形验证码", Method: "POST", Path: "/api/v1/auth/password-login-code"},
		{Name: "用户名密码登录", Description: "账号密码登录", Method: "POST", Path: "/api/v1/auth/login"},
		{Name: "邮箱验证码登录", Description: "邮箱验证码登录", Method: "POST", Path: "/api/v1/auth/email-login"},
		{Name: "提交注册申请", Description: "自助注册申请", Method: "POST", Path: "/api/v1/auth/register"},
		{Name: "密码策略", Description: "登录页读取密码强度策略", Method: "GET", Path: "/api/v1/auth/password-policy"},
		{Name: "刷新令牌", Description: "刷新访问令牌", Method: "POST", Path: "/api/v1/auth/refresh"},
		{Name: "退出登录", Description: "注销当前会话", Method: "POST", Path: "/api/v1/auth/logout", Auth: true},
		{Name: "WebSocket 票据", Description: "换取 WebSocket 一次性票据", Method: "POST", Path: "/api/v1/auth/ws-ticket", Auth: true},
		{Name: "当前用户", Description: "读取当前登录用户资料", Method: "GET", Path: "/api/v1/auth/me", Auth: true},
		{Name: "更新个人资料", Description: "修改昵称等个人资料", Method: "PUT", Path: "/api/v1/auth/me", Auth: true},
		{Name: "修改密码", Description: "修改本人登录密码", Method: "PUT", Path: "/api/v1/auth/password", Auth: true},
		{Name: "当前用户能力", Description: "当前用户可调用的 API 能力（按钮级权限）", Method: "GET", Path: "/api/v1/auth/permissions", Auth: true},
		{Name: "登录二次验证", Description: "TOTP/恢复码换取会话", Method: "POST", Path: "/api/v1/auth/mfa/verify"},
		{Name: "生成 TOTP 密钥", Description: "启用二次验证前生成密钥", Method: "POST", Path: "/api/v1/auth/mfa/setup", Auth: true},
		{Name: "启用二次验证", Description: "校验 TOTP 后启用并返回恢复码", Method: "POST", Path: "/api/v1/auth/mfa/enable", Auth: true},
		{Name: "关闭二次验证", Description: "校验密码与 TOTP 后关闭", Method: "POST", Path: "/api/v1/auth/mfa/disable", Auth: true},
		{Name: "侧栏菜单树", Description: "按角色过滤的侧栏菜单", Method: "GET", Path: "/api/v1/menus/tree", Auth: true},
		{Name: "Jenkins 回调", Description: "CI/CD 流水线回调", Method: "POST", Path: "/api/v1/cicd/jenkins/callback"},
		{Name: "告警回调", Description: "Alertmanager Webhook", Method: "POST", Path: "/api/v1/alerts/webhook/alertmanager"},
		{Name: "Loggie 心跳", Description: "日志采集器心跳上报", Method: "POST", Path: "/api/v1/loggie/heartbeat/report"},
	}
}

// Entries 返回完整目录：业务权限 + 公开接口。
func Entries() []Entry {
	out := make([]Entry, 0, 760)
	for _, g := range Groups() {
		for _, p := range g.Entries {
			out = append(out, Entry{
				Group:       g.Title,
				Name:        p.Name,
				Description: p.Description,
				Method:      strings.ToUpper(strings.TrimSpace(p.Action)),
				Path:        strings.TrimSpace(p.Resource),
				Auth:        true,
			})
		}
	}
	out = append(out, publicEntries()...)
	return out
}

var (
	indexOnce sync.Once
	index     map[string]Entry
)

func buildIndex() {
	index = make(map[string]Entry, 760)
	for _, e := range Entries() {
		index[Key(e.Method, e.Path)] = e
	}
}

// Key 归一化 method + path，作为目录查表键。
func Key(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

// Lookup 按 method + path 查目录项；未登记的路由返回 ok=false。
func Lookup(method, path string) (Entry, bool) {
	indexOnce.Do(buildIndex)
	e, ok := index[Key(method, path)]
	return e, ok
}

// pseudoKeys 无同名路由的伪权限：用于把「敏感凭据类字典」与普通数据字典拆开授权。
// 升级环境由启动同步幂等补齐，因此它们不受「无路由即回收」影响。
var pseudoKeys = []string{
	"GET /api/v1/dict/sensitive",
	"PUT /api/v1/dict/sensitive",
}

// PseudoPermissions 返回无路由但仍需登记/授权的权限点。
func PseudoPermissions() []model.Permission {
	out := make([]model.Permission, 0, len(pseudoKeys))
	for _, key := range pseudoKeys {
		method, path, _ := strings.Cut(key, " ")
		e, ok := Lookup(method, path)
		if !ok {
			// 目录漏登记时兜底，避免启动同步静默丢失敏感字典权限点。
			out = append(out, model.Permission{
				Name:     path + " " + method,
				Resource: path,
				Action:   method,
				Description: "伪权限（目录未登记）",
			})
			continue
		}
		out = append(out, model.Permission{
			Name:        e.Name,
			Resource:    e.Path,
			Action:      e.Method,
			Description: e.Description,
		})
	}
	return out
}

// IsPseudoResource 判断资源是否属于「无路由但需保留」的伪权限，回收逻辑据此跳过。
func IsPseudoResource(resource string) bool {
	r := strings.TrimSpace(resource)
	for _, key := range pseudoKeys {
		_, path, _ := strings.Cut(key, " ")
		if r == path {
			return true
		}
	}
	return false
}

// SnapshotGroup 目录快照分组。
type SnapshotGroup struct {
	Title   string  `json:"title"`
	Entries []Entry `json:"entries"`
}

// SnapshotDoc 目录快照文档，由 `yunshu permission-catalog` 生成，用于评审与版本差异对比。
type SnapshotDoc struct {
	Total  int             `json:"total"`
	Groups []SnapshotGroup `json:"groups"`
}

// Snapshot 输出完整目录快照（含公开接口）。
func Snapshot() SnapshotDoc {
	groups := Groups()
	doc := SnapshotDoc{Groups: make([]SnapshotGroup, 0, len(groups)+1)}
	total := 0
	for _, g := range groups {
		entries := make([]Entry, 0, len(g.Entries))
		for _, p := range g.Entries {
			entries = append(entries, Entry{
				Group:       g.Title,
				Name:        p.Name,
				Description: p.Description,
				Method:      strings.ToUpper(strings.TrimSpace(p.Action)),
				Path:        strings.TrimSpace(p.Resource),
				Auth:        true,
			})
		}
		total += len(entries)
		doc.Groups = append(doc.Groups, SnapshotGroup{Title: g.Title, Entries: entries})
	}
	public := publicEntries()
	total += len(public)
	doc.Groups = append(doc.Groups, SnapshotGroup{Title: "公开接口（无需授权）", Entries: public})
	doc.Total = total
	return doc
}
