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
	regexPattern        string
	timestampFromLayout string // 空则不写 timestamp action
	timestampLocation   string // Local / Asia/Shanghai / UTC
}

var parseProfileByName = map[string]pipelineParseProfile{
	"syslog":        profileSyslog(),
	"elasticsearch": profileElasticsearch(),
	"java_bracket":  profileElasticsearch(),
	"spring":        profileSpringLog(),
	"microservice":  profileSpringLog(),
	"cri":           profileCRI(),
	"k8s":           profileCRI(),
	"nginx_access":  profileNginxAccess(),
	"plain":         {name: "plain"},
}

func profileSyslog() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "syslog",
		multilinePattern:    `^\w{3}\s+\d{1,2}\s+`,
		regexPattern:        `^(?P<ts>\w+\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(?P<host>\S+)\s+(?P<message>.*)$`,
		timestampFromLayout: "Jan _2 15:04:05",
		timestampLocation:   "Local",
	}
}

func profileElasticsearch() pipelineParseProfile {
	return pipelineParseProfile{
		name:             "elasticsearch",
		multilinePattern: `^\[`,
		// [2026-07-13T23:42:49,235][WARN ][o.e.t.ThreadPool ] [yunshuNode] message
		regexPattern:        `^\[(?P<ts>[^\]]+)\]\[(?P<level>\w+)\s*\]\[(?P<component>[^\]]*)\]\s*(?:\[(?P<node>[^\]]+)\]\s*)?(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02T15:04:05,000",
		timestampLocation:   "Local",
	}
}

func profileSpringLog() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "spring",
		multilinePattern:    `^\d{4}-\d{2}-\d{2}`,
		regexPattern:        `^(?P<ts>\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:[.,]\d{3})?(?:Z|[+-]\d{2}:?\d{2})?)\s+(?P<level>TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC)\s+(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02 15:04:05",
		timestampLocation:   "Local",
	}
}

func profileNginxAccess() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "nginx_access",
		multilinePattern:    `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`,
		regexPattern:        `^(?P<remote>\S+)\s+-\s+-\s+\[(?P<ts>[^\]]+)\]\s+"(?P<request>[^"]*)"\s+(?P<status>\d{3})\s+(?P<bytes>\S+)(?:\s+"(?P<referrer>[^"]*)"\s+"(?P<agent>[^"]*)")?\s*$`,
		timestampFromLayout: "02/Jan/2006:15:04:05 -0700",
		timestampLocation:   "Local",
	}
}

func profileCRI() pipelineParseProfile {
	return pipelineParseProfile{
		name:             "cri",
		multilinePattern: `^\d{4}-\d{2}-\d{2}T`,
		// 2024-07-14T02:32:33.306038878+00:00 stderr F Trace[...]
		regexPattern:        `^(?P<ts>\S+)\s+(?P<stream>stdout|stderr)\s+(?P<flag>\S)\s+(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02T15:04:05.999999999Z07:00",
		timestampLocation:   "UTC",
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
	case strings.Contains(p, "/var/log/pods/"), strings.Contains(p, "/pods/kube-"), strings.Contains(p, "kube-apiserver"):
		return profileCRI()
	case strings.Contains(p, "elasticsearch"), strings.Contains(p, "/es/"), strings.HasSuffix(p, "es.log"):
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

// renderTransformerActions 生成 Loggie v1.5 transformer actions：
// regex 提取 level/ts/message → timestamp 写回 @timestamp → move 规范字段。
func (p pipelineParseProfile) renderTransformerActions() string {
	if !p.hasTransformer() {
		return ""
	}
	loc := strings.TrimSpace(p.timestampLocation)
	if loc == "" {
		loc = "Local"
	}
	var b strings.Builder
	b.WriteString("    interceptors:\n")
	b.WriteString("      - type: transformer\n")
	b.WriteString("        actions:\n")
	b.WriteString("          - action: regex(body)\n")
	fmt.Fprintf(&b, "            pattern: '%s'\n", p.regexPattern)
	b.WriteString("            ignoreError: true\n")
	if layout := strings.TrimSpace(p.timestampFromLayout); layout != "" {
		b.WriteString("          - action: timestamp(ts)\n")
		fmt.Fprintf(&b, "            fromLayout: %q\n", layout)
		fmt.Fprintf(&b, "            fromLocation: %s\n", loc)
		b.WriteString("            toLayout: \"2006-01-02T15:04:05.000Z07:00\"\n")
		b.WriteString("            toLocation: UTC\n")
		b.WriteString("            ignoreError: true\n")
		b.WriteString("          - action: move(ts, @timestamp)\n")
		b.WriteString("            ignoreError: true\n")
	}
	b.WriteString("          - action: move(message, body)\n")
	b.WriteString("            ignoreError: true\n")
	return b.String()
}
