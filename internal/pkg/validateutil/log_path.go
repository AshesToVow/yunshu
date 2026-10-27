package validateutil

import (
	"fmt"
	"strings"
)

var allowedLogPathRoots = []string{
	"/var/log/",
	"/var/lib/",
	"/opt/",
	"/data/",
	"/app/",
	"/srv/",
	"/usr/local/",
	"/home/",
	"/tmp/",
	"/logs/",
	"/workspace/",
}

var deniedLogPathSegments = map[string]bool{
	".ssh":    true,
	".aws":    true,
	".kube":   true,
	".docker": true,
	".gnupg":  true,
	"secrets": true,
	"secret":  true,
}

var deniedLogPathNames = map[string]bool{
	"shadow":      true,
	"gshadow":     true,
	"sudoers":     true,
	"id_rsa":      true,
	"id_dsa":      true,
	"id_ecdsa":    true,
	"id_ed25519":  true,
	".env":        true,
	"credentials": true,
}

// ValidateLogCollectPath 限制 Loggie 采集路径只能落在常见日志目录，避免读取系统凭据文件。
func ValidateLogCollectPath(raw string) error {
	p := strings.TrimSpace(raw)
	if p == "" {
		return fmt.Errorf("日志路径为空")
	}
	if strings.ContainsRune(p, '\x00') {
		return fmt.Errorf("日志路径包含非法字符")
	}
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("日志路径必须为绝对路径")
	}
	if strings.Contains(p, "..") {
		return fmt.Errorf("日志路径不允许包含 ..")
	}
	staticPrefix := p
	if i := strings.IndexAny(staticPrefix, "*?["); i >= 0 {
		staticPrefix = staticPrefix[:i]
	}
	allowed := false
	for _, root := range allowedLogPathRoots {
		if strings.HasPrefix(staticPrefix, root) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Errorf("日志路径必须位于允许的日志目录（%s）", strings.Join(allowedLogPathRoots, "、"))
	}
	lower := strings.ToLower(p)
	for _, segment := range strings.Split(lower, "/") {
		if segment == "" {
			continue
		}
		if deniedLogPathSegments[segment] {
			return fmt.Errorf("日志路径命中敏感目录: %s", segment)
		}
		if deniedLogPathNames[segment] {
			return fmt.Errorf("日志路径命中敏感文件: %s", segment)
		}
	}
	return nil
}
