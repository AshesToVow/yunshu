package runbooks

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed *.md
var fs embed.FS

// Names 返回内置剧本名（不含扩展名）。
func Names() []string {
	entries, err := fs.ReadDir(".")
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".md") {
			out = append(out, strings.TrimSuffix(name, ".md"))
		}
	}
	return out
}

// Load 读取剧本正文。
func Load(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("runbook name required")
	}
	if !strings.HasSuffix(name, ".md") {
		name += ".md"
	}
	raw, err := fs.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("load runbook %s: %w", name, err)
	}
	return string(raw), nil
}

// MatchByReason 按容器/诊断 reason 选择剧本。
func MatchByReason(reason, summary string) string {
	blob := strings.ToLower(reason + " " + summary)
	switch {
	case strings.Contains(blob, "crashloopbackoff"), strings.Contains(blob, "crash loop"):
		return "CrashLoopBackOff"
	case strings.Contains(blob, "imagepullbackoff"), strings.Contains(blob, "errimagepull"), strings.Contains(blob, "image pull"):
		return "ImagePullBackOff"
	case strings.Contains(blob, "pending"), strings.Contains(blob, "unschedulable"), strings.Contains(blob, "insufficient"):
		return "PendingUnschedulable"
	default:
		return "CrashLoopBackOff"
	}
}
