package logplatform

import (
	"fmt"
	"strings"

	"yunshu/internal/model"
)

// pipelineParseProfile 描述 Loggie multi + regex + timestamp 解析规则。
// Loggie file source 使用 multi.active + pattern（匹配行=新事件，否则并入上一条）。
type pipelineParseProfile struct {
	name                string
	multilinePattern    string
	regexPattern        string
	extraRegexPattern   string
	timestampFromLayout string
	timestampLocation   string
	maxLines            int
	criPassThroughTS    bool
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
	"kafka":         profileKafka(),
	"redis":         profileRedis(),
	"mysql":         profileMySQLError(),
	"mysql_error":   profileMySQLError(),
	"mysql_slow":    profileMySQLSlow(),
	"zookeeper":     profileZookeeper(),
	"zk":            profileZookeeper(),
	"cityeyes":      profileCityEyesVap(),
	"cityeyes-vap":  profileCityEyesVap(),
	"plain":         {name: "plain"},
}

// ParseProfileOption UI 可选解析模板。
type ParseProfileOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// ListParseProfileOptions 日志源 multiline_rule 下拉选项。
func ListParseProfileOptions() []ParseProfileOption {
	return []ParseProfileOption{
		{Value: "elasticsearch", Label: "Elasticsearch / Java [ERROR ] 方括号堆栈"},
		{Value: "spring", Label: "Spring Boot 2024-01-01 INFO（含 Logback）"},
		{Value: "cityeyes-vap", Label: "CityEyesVap 管道分隔（含 JSON/堆栈多行）"},
		{Value: "kafka", Label: "Kafka [时间] INFO 方括号"},
		{Value: "redis", Label: "Redis 1:M 日期 * / # 级别"},
		{Value: "mysql", Label: "MySQL 5.7 error log"},
		{Value: "mysql_slow", Label: "MySQL slow query log（# Time:）"},
		{Value: "zookeeper", Label: "Zookeeper 2024-01-01,123 [myid:1] - INFO"},
		{Value: "cri", Label: "K8s CRI 容器日志（/var/log/pods）"},
		{Value: "syslog", Label: "Syslog /var/log/messages"},
		{Value: "nginx_access", Label: "Nginx access"},
	}
}

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
	return pipelineParseProfile{
		name: "elasticsearch",
		// 行首 [ 为新事件；at / Caused by / 缩进行并入上一条（ERROR/WARN/INFO 均适用，非仅 WARN）
		multilinePattern:    `^\[`,
		regexPattern:        `(?s)^\[(?P<ts>[^\]]+)\]\[(?P<level>TRACE|DEBUG|INFO|WARN|ERROR|FATAL)\s*\]\[(?P<component>[^\]]*)\]\s*(?:\[(?P<node>[^\]]+)\]\s*)?(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02T15:04:05,000",
		timestampLocation:   "Local",
		maxLines:            500,
	}
}

func profileSpringLog() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "spring",
		multilinePattern:    `^\d{4}-\d{2}-\d{2}`,
		regexPattern:        `(?s)^(?P<ts>\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:[.,]\d{3})?(?:Z|[+-]\d{2}:?\d{2})?)\s+(?P<level>TRACE|DEBUG|INFO|WARN|ERROR|FATAL|PANIC)\s+(?:\[[^\]]+\]\s+)?(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02 15:04:05.000",
		timestampLocation:   "Local",
		maxLines:            300,
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

func profileKafka() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "kafka",
		multilinePattern:    `^\[`,
		regexPattern:        `(?s)^\[(?P<ts>[^\]]+)\]\s+(?P<level>TRACE|DEBUG|INFO|WARN|ERROR|FATAL)\s+(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02 15:04:05,000",
		timestampLocation:   "Local",
		maxLines:            300,
	}
}

func profileRedis() pipelineParseProfile {
	return pipelineParseProfile{
		name:             "redis",
		multilinePattern: `^\d+:[CM]`,
		// 1:M 16 Jan 2026 18:46:12.965 * message  /  # Warning
		regexPattern:        `^(?P<pid>\d+):(?P<role>[CM])\s+(?P<ts>\d{1,2} \w{3} \d{4} \d{2}:\d{2}:\d{2}\.\d+)\s+(?P<rlevel>[*#-])\s+(?P<message>.*)$`,
		timestampFromLayout: "02 Jan 2006 15:04:05.000",
		timestampLocation:   "Local",
		maxLines:            200,
	}
}

func profileMySQLError() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "mysql",
		multilinePattern:    `^\d{4}-\d{2}-\d{2}T`,
		regexPattern:        `(?s)^(?P<ts>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s+\d+\s+\[(?P<level>Note|Warning|ERROR|System)\]\s+(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02T15:04:05.999999Z",
		timestampLocation:   "UTC",
		maxLines:            300,
	}
}

func profileMySQLSlow() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "mysql_slow",
		multilinePattern:    `^# Time:`,
		regexPattern:        `(?s)^# Time:\s+(?P<ts>\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z).*(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02T15:04:05.999999Z",
		timestampLocation:   "UTC",
		maxLines:            100,
	}
}

func profileZookeeper() pipelineParseProfile {
	return pipelineParseProfile{
		name:                "zookeeper",
		multilinePattern:    `^\d{4}-\d{2}-\d{2}`,
		regexPattern:        `(?s)^(?P<ts>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2},\d{3})\s+\[(?P<myid>[^\]]+)\]\s+-\s+(?P<level>TRACE|DEBUG|INFO|WARN|ERROR|FATAL)\s+(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02 15:04:05.000",
		timestampLocation:   "Local",
		maxLines:            300,
	}
}

func profileCityEyesVap() pipelineParseProfile {
	return pipelineParseProfile{
		name: "cityeyes-vap",
		// App|service|ip|time| LEVEL|thread|class:line|msg；JSON/堆栈行不以该模式开头则并入
		multilinePattern: `^[^|{].+\|[^|]+\|[\d.]+\|\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}`,
		regexPattern:     `(?s)^(?P<app>[^|]+)\|(?P<service>[^|]+)\|(?P<host>[^|]+)\|(?P<ts>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\|\s*(?P<level>TRACE|DEBUG|INFO|WARN|ERROR|FATAL)\|(?P<thread>[^|]+)\|(?P<source>[^|]+)\|(?P<message>.*)$`,
		timestampFromLayout: "2006-01-02 15:04:05.000",
		timestampLocation:   "Local",
		maxLines:            500,
	}
}

func profileCRI() pipelineParseProfile {
	return pipelineParseProfile{
		name:             "cri",
		multilinePattern: `^\d{4}-\d{2}-\d{2}T`,
		regexPattern:     `^(?P<ts>\d{4}-\d{2}-\d{2}T\S+)\s+(?P<stream>stdout|stderr)\s+(?P<flag>\S)\s+(?P<message>.*)$`,
		extraRegexPattern: `^(?P<klevel>[IWEF])(?P<md>\d{4})\s+(?P<ktime>\d{2}:\d{2}:\d{2}\.\d+)\s+(?P<pid>\d+)\s+(?P<source>[^\]]+)\]\s*(?P<kmsg>.*)$`,
		maxLines:          200,
		criPassThroughTS:  true,
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
	case strings.Contains(p, "cityeyes"), strings.Contains(p, "cityeyes-vap"):
		return profileCityEyesVap()
	case strings.Contains(p, "kafka"), strings.Contains(p, "server.log") && strings.Contains(p, "kafka"):
		return profileKafka()
	case strings.Contains(p, "redis"):
		return profileRedis()
	case strings.Contains(p, "slow.log"), strings.Contains(p, "slow-query"):
		return profileMySQLSlow()
	case strings.Contains(p, "mysql"), strings.Contains(p, "mysqld"), strings.Contains(p, "mariadb"):
		return profileMySQLError()
	case strings.Contains(p, "zookeeper"), strings.Contains(p, "/zk/"), strings.HasSuffix(p, "zookeeper.log"):
		return profileZookeeper()
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

	// Redis：* → INFO，# → WARN
	if p.name == "redis" {
		b.WriteString("          - if: equal(rlevel, *)\n")
		b.WriteString("            then:\n")
		b.WriteString("              - action: add(level, INFO)\n")
		b.WriteString("          - if: equal(rlevel, #)\n")
		b.WriteString("            then:\n")
		b.WriteString("              - action: add(level, WARN)\n")
		b.WriteString("          - if: equal(rlevel, -)\n")
		b.WriteString("            then:\n")
		b.WriteString("              - action: add(level, DEBUG)\n")
	}

	if p.criPassThroughTS {
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
		// nginx 等非 klog 行不会产出 kmsg；无条件 move 会把 message/body 写成 null → ES 出现 "<nil>"
		b.WriteString("          - if: exist(kmsg)\n")
		b.WriteString("            then:\n")
		b.WriteString("              - action: move(kmsg, message)\n")
		b.WriteString("                ignoreError: true\n")
	}

	b.WriteString("          - if: exist(message)\n")
	b.WriteString("            then:\n")
	b.WriteString("              - action: move(message, body)\n")
	b.WriteString("                ignoreError: true\n")
	return b.String()
}
