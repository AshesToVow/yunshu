package logplatform

import (
	"fmt"
	"strings"

	"yunshu/internal/model"
)

// pipelineParseProfile 描述 Loggie multiline + regex + timestamp 解析规则。
type pipelineParseProfile struct {
	name                string
	multilinePattern    string
	multilineNegate     *bool  // nil 表示 true（新事件匹配 pattern）
	regexPattern        string
	extraRegexPattern   string // 二次 regex（如 CRI 后再解析 klog）
	timestampFromLayout string
	timestampLocation   string // 空=不写 fromLocation（保留串内时区）
	maxLines            int
	criPassThroughTS    bool // CRI 头时间已是 RFC3339，直接 move 不做 layout 转换
}

var parseProfileByName = map[string]pipelineParseProfile{
	"syslog":        profileSyslog(),
	"elasticsearch": profileElasticsearch(),
	"java_bracket":  profileElasticsearch(),
	"spring":        profileSpringLog(),
	"microservice":  profileSpringLog(),
	"cri":           profileCRI(),
	"k8s":           profileCRI(),
	"klog":          profileCRI(),
	"nginx_access":  profileNginxAccess(),
	"plain":         {name: "plain"},
}

func boolPtr(v bool) *bool { return &v }

func profileSyslog() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "syslog",
		multilinePattern:    `^\w{3}\s+\d{1,2}\s+`,
		regexPattern:        `^(?P<ts>\w+\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(?P<host>\S+)\s+(?P<message>.*)$`,
		timestampFromLayout: "Jan _2 15:04:05",
		timestampLocation:   "Local",
		maxLines:            200,
	}
}

func profileElasticsearch() pipelineParseProfile {
	neg := true
	return pipelineParseProfile{
		name: "elasticsearch",
		// 以 [yyyy-MM-dd 开头的行为新事件；堆栈 at / Caused by / 缩进行并入上一行
		multilinePattern:    `^\[\d{4}-\d{2}-\d{2}`,
		multilineNegate:     &neg,
		regexPattern:        `(?s)^\[(?P<ts>[^\]]+)\]\[(?P<level>\w+)\s*\]\[(?P<component>[^\]]*)\]\s*(?:\[(?P<node>[^\]]+)\]\s*)?(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02T15:04:05,000",
		timestampLocation:   "Local",
		maxLines:            500,
	}
}

func profileSpringLog() pipelineParseProfile {
	neg := true
	return pipelineParseProfile{
		name:                "spring",
		multilinePattern:    `^\d{4}-\d{2}-\d{2}`,
		multilineNegate:     &neg,
		regexPattern:        `(?s)^(?P<ts>\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:[.,]\d{3})?(?:Z|[+-]\d{2}:?\d{2})?)\s+(?P<level>TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC)\s+(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02 15:04:05",
		timestampLocation:   "Local",
		maxLines:            200,
	}
}

func profileNginxAccess() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "nginx_access",
		multilinePattern:    `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`,
		regexPattern:        `^(?P<remote>\S+)\s+-\s+-\s+\[(?P<ts>[^\]]+)\]\s+"(?P<request>[^"]*)"\s+(?P<status>\d{3})\s+(?P<bytes>\S+)(?:\s+"(?P<referrer>[^"]*)"\s+"(?P<agent>[^"]*)")?\s*$`,
		timestampFromLayout: "02/Jan/2006:15:04:05 -0700",
		maxLines:            50,
	}
}

func profileCRI() pipelineParseProfile {
	neg := true
	return pipelineParseProfile{
		name: "cri",
		// CRI：2026-07-16T10:51:52.902943486+08:00 stderr F <payload>
		multilinePattern: `^\d{4}-\d{2}-\d{2}T`,
		multilineNegate:  &neg,
		regexPattern:     `^(?P<ts>\d{4}-\d{2}-\d{2}T\S+)\s+(?P<stream>stdout|stderr)\s+(?P<flag>\S)\s+(?P<message>.*)$`,
		// 二次解析 klog：I0716 02:51:52.902837       1 httplog.go:132] ...
		extraRegexPattern: `^(?P<klevel>[IWEF])(?P<md>\d{4})\s+(?P<ktime>\d{2}:\d{2}:\d{2}\.\d+)\s+(?P<pid>\d+)\s+(?P<source>[^\]]+)\]\s*(?P<kmsg>.*)$`,
		// CRI 头已是 RFC3339Nano+offset，直接 move 到 @timestamp，避免 fromLocation 误解析
		timestampFromLayout: "",
		timestampLocation:   "",
		maxLines:            200,
		criPassThroughTS:    true,
	}
}

func usesSyslogFormat(paths []string) bool {
	for _, p := range paths {
		p = strings.ToLower(p)
		for _, name := range []string{"/messages", "/syslog", "/secure", "/maillog", "/cron"} {
			if strings.Contains(p, name) {
				return true
			}
		}
	}
	return false
}

func pipelineParseProfileFor(paths []string) pipelineParseProfile {
	if len(paths) == 0 {
		return profileSpringLog()
	}
	return detectParseProfile(paths[0], nil)
}

func detectParseProfile(path string, multilineRule *string) pipelineParseProfile {
	if multilineRule != nil {
		rule := strings.TrimSpace(*multilineRule)
		if rule != "" {
			if p, ok := parseProfileByName[strings.ToLower(rule)]; ok {
				return p
			}
			if strings.HasPrefix(rule, "^") {
				base := inferProfileFromPath(path)
				base.multilinePattern = rule
				return base
			}
		}
	}
	return inferProfileFromPath(path)
}

func inferProfileFromPath(path string) pipelineParseProfile {
	p := strings.ToLower(strings.TrimSpace(path))
	switch {
	case strings.Contains(p, "/var/log/pods"), strings.Contains(p, "/var/log/containers"),
		strings.Contains(p, "/pods/"), strings.Contains(p, "kube-apiserver"),
		strings.Contains(p, "metrics-server"):
		return profileCRI()
	case strings.Contains(p, "elasticsearch"), strings.Contains(p, "/es/"), strings.Contains(p, "yunshu.log"),
		strings.HasSuffix(p, "es.log"), strings.Contains(p, "elasticsearch-"),
		strings.Contains(p, "/logs/yunshu"):
		return profileElasticsearch()
	case usesSyslogFormat([]string{path}):
		return profileSyslog()
	case strings.Contains(p, "nginx") && strings.Contains(p, "access"):
		return profileNginxAccess()
	case strings.HasSuffix(p, ".log"), strings.Contains(p, "*.log"):
		return profileSpringLog()
	default:
		return profileSpringLog()
	}
}

func parseProfileForLogSource(src model.ServiceLogSource) pipelineParseProfile {
	path := strings.TrimSpace(src.Path)
	glob := logSourceToGlobPath(src)
	if glob != "" {
		path = glob
	}
	return detectParseProfile(path, src.MultilineRule)
}

func (p pipelineParseProfile) hasTransformer() bool {
	return strings.TrimSpace(p.regexPattern) != ""
}

func (p pipelineParseProfile) multilineNegateValue() bool {
	if p.multilineNegate == nil {
		return true
	}
	return *p.multilineNegate
}

func (p pipelineParseProfile) multilineMaxLines() int {
	if p.maxLines > 0 {
		return p.maxLines
	}
	return 200
}

// renderTransformerActions 生成 Loggie v1.5 transformer actions。
func (p pipelineParseProfile) renderTransformerActions() string {
	if !p.hasTransformer() {
		return ""
	}
	var b strings.Builder
	b.WriteString("    interceptors:\n")
	b.WriteString("      - type: transformer\n")
	b.WriteString("        actions:\n")
	b.WriteString("          - action: regex(body)\n")
	fmt.Fprintf(&b, "            pattern: '%s'\n", p.regexPattern)
	b.WriteString("            ignoreError: true\n")
	if p.criPassThroughTS {
		// 已是 RFC3339Nano（含时区），直接覆盖 schema 写入的采集时间
		b.WriteString("          - action: move(ts, @timestamp)\n")
		b.WriteString("            ignoreError: true\n")
	} else if layout := strings.TrimSpace(p.timestampFromLayout); layout != "" {
		b.WriteString("          - action: timestamp(ts)\n")
		fmt.Fprintf(&b, "            fromLayout: %q\n", layout)
		if loc := strings.TrimSpace(p.timestampLocation); loc != "" {
			fmt.Fprintf(&b, "            fromLocation: %s\n", loc)
		}
		b.WriteString("            toLayout: \"2006-01-02T15:04:05.000Z07:00\"\n")
		b.WriteString("            toLocation: UTC\n")
		b.WriteString("            ignoreError: true\n")
		b.WriteString("          - action: move(ts, @timestamp)\n")
		b.WriteString("            ignoreError: true\n")
	}
	if extra := strings.TrimSpace(p.extraRegexPattern); extra != "" {
		// CRI payload → klog：I/W/E/F → level，正文写入 message
		b.WriteString("          - action: regex(message)\n")
		fmt.Fprintf(&b, "            pattern: '%s'\n", extra)
		b.WriteString("            ignoreError: true\n")
		b.WriteString("          - if: equal(klevel, I)\n")
		b.WriteString("            then:\n")
		b.WriteString("              - action: add(level, INFO)\n")
		b.WriteString("          - if: equal(klevel, W)\n")
		b.WriteString("            then:\n")
		b.WriteString("              - action: add(level, WARN)\n")
		b.WriteString("          - if: equal(klevel, E)\n")
		b.WriteString("            then:\n")
		b.WriteString("              - action: add(level, ERROR)\n")
		b.WriteString("          - if: equal(klevel, F)\n")
		b.WriteString("            then:\n")
		b.WriteString("              - action: add(level, FATAL)\n")
		b.WriteString("          - action: move(kmsg, message)\n")
		b.WriteString("            ignoreError: true\n")
	}
	b.WriteString("          - action: move(message, body)\n")
	b.WriteString("            ignoreError: true\n")
	return b.String()
}
