package logplatform

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAgentIndexPrefix = "yunshu-agent"
	defaultK8sIndexPrefix   = "yunshu-k8s"
	maxSearchIndexServers   = 80
)

var (
	// yunshu-agent-10-10-10-5-2026.07.17
	agentNameDateSuffix = regexp.MustCompile(`^(.+)-(\d{4}\.\d{2}\.\d{2})$`)
	// legacy yunshu-agent-8
	agentLegacyIDTopic = regexp.MustCompile(`^(\d+)$`)
)

// SanitizeHostForName 将服务器 IP/主机名转为 Topic/索引安全片段（点、冒号等替换为 -）。
func SanitizeHostForName(host string) string {
	h := strings.TrimSpace(host)
	if h == "" {
		return "unknown"
	}
	var b strings.Builder
	b.Grow(len(h))
	for _, r := range h {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return out
}

func agentNamePrefix(prefix string) string {
	p := strings.Trim(strings.TrimSpace(prefix), "-")
	if p == "" {
		p = defaultAgentIndexPrefix
	}
	return p
}

// AgentIndexSink 生成 Loggie ES sink 按日索引：yunshu-agent-{ip}-${+YYYY.MM.DD}
func AgentIndexSink(serverHost string) string {
	return fmt.Sprintf("%s-%s-${+YYYY.MM.DD}", defaultAgentIndexPrefix, SanitizeHostForName(serverHost))
}

// AgentIndexSinkByServerID 兼容旧版按 server_id 的索引模板（仅测试/回退）。
func AgentIndexSinkByServerID(serverID uint) string {
	return fmt.Sprintf("%s-%d-${+YYYY.MM.DD}", defaultAgentIndexPrefix, serverID)
}

// AgentIndexForDay 消费端按日索引名（IP + 日期）。
func AgentIndexForDay(serverHost string, day time.Time) string {
	if day.IsZero() {
		day = time.Now().UTC()
	}
	return fmt.Sprintf("%s-%s-%s", defaultAgentIndexPrefix, SanitizeHostForName(serverHost), day.UTC().Format("2006.01.02"))
}

// AgentIndexForDayByServerID 兼容旧索引 yunshu-agent-{id}-YYYY.MM.DD。
func AgentIndexForDayByServerID(serverID uint, day time.Time) string {
	if day.IsZero() {
		day = time.Now().UTC()
	}
	return fmt.Sprintf("%s-%d-%s", defaultAgentIndexPrefix, serverID, day.UTC().Format("2006.01.02"))
}

// AgentKafkaTopicTemplate Loggie Kafka sink topic（含日期占位）：yunshu-agent-{ip}-${+YYYY.MM.DD}
func AgentKafkaTopicTemplate(serverHost, prefix string) string {
	return fmt.Sprintf("%s-%s-${+YYYY.MM.DD}", agentNamePrefix(prefix), SanitizeHostForName(serverHost))
}

// AgentKafkaTopicForDay 具体某日 Topic 名（自动建 Topic 用）。
func AgentKafkaTopicForDay(serverHost, prefix string, day time.Time) string {
	if day.IsZero() {
		day = time.Now().UTC()
	}
	return fmt.Sprintf("%s-%s-%s", agentNamePrefix(prefix), SanitizeHostForName(serverHost), day.UTC().Format("2006.01.02"))
}

// AgentKafkaTopic 兼容旧命名 yunshu-agent-{server_id}。
func AgentKafkaTopic(serverID uint, prefix string) string {
	return fmt.Sprintf("%s-%d", agentNamePrefix(prefix), serverID)
}

// IsAgentKafkaTopic 判断是否为平台 Agent Topic（新 IP+日期 或旧 server_id）。
func IsAgentKafkaTopic(topic, prefix string) bool {
	p := agentNamePrefix(prefix)
	topic = strings.TrimSpace(topic)
	head := p + "-"
	if !strings.HasPrefix(topic, head) {
		return false
	}
	rest := strings.TrimPrefix(topic, head)
	if rest == "" {
		return false
	}
	if agentLegacyIDTopic.MatchString(rest) {
		return true
	}
	if m := agentNameDateSuffix.FindStringSubmatch(rest); len(m) == 3 {
		return SanitizeHostForName(m[1]) == m[1] || m[1] != ""
	}
	// 无日期的 IP 片段（容错）
	return strings.ContainsAny(rest, ".-_") || len(rest) > 0
}

// ParseServerIDFromAgentKafkaTopic 从旧 Topic yunshu-agent-7 解析 server_id。
func ParseServerIDFromAgentKafkaTopic(topic, prefix string) (uint, bool) {
	p := agentNamePrefix(prefix)
	topic = strings.TrimSpace(topic)
	head := p + "-"
	if !strings.HasPrefix(topic, head) {
		return 0, false
	}
	rest := strings.TrimPrefix(topic, head)
	if !agentLegacyIDTopic.MatchString(rest) {
		return 0, false
	}
	id, err := strconv.ParseUint(rest, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// ParseHostKeyFromAgentName 从 Topic/索引名解析主机片段（去日期后缀）。
func ParseHostKeyFromAgentName(name, prefix string) (hostKey string, ok bool) {
	p := agentNamePrefix(prefix)
	name = strings.TrimSpace(name)
	head := p + "-"
	if !strings.HasPrefix(name, head) {
		return "", false
	}
	rest := strings.TrimPrefix(name, head)
	if agentLegacyIDTopic.MatchString(rest) {
		return "", false
	}
	if m := agentNameDateSuffix.FindStringSubmatch(rest); len(m) == 3 {
		return m[1], m[1] != ""
	}
	if rest != "" {
		return rest, true
	}
	return "", false
}

// AgentIndexPattern 单服务器检索通配（按 IP）。
func AgentIndexPattern(serverHost string) string {
	return fmt.Sprintf("%s-%s-*", defaultAgentIndexPrefix, SanitizeHostForName(serverHost))
}

// AgentIndexPatternByServerID 旧版按 ID 通配。
func AgentIndexPatternByServerID(serverID uint) string {
	return fmt.Sprintf("%s-%d-*", defaultAgentIndexPrefix, serverID)
}

// GlobalAgentIndexPattern 全量 Agent 索引通配。
func GlobalAgentIndexPattern() string {
	return defaultAgentIndexPrefix + "-*"
}

// ResolveSearchIndices 按服务器 ID 列表拼索引（仅旧格式回退）；优先用 ResolveSearchIndicesByHosts。
func ResolveSearchIndices(serverID *uint, projectServerIDs []uint) string {
	if serverID != nil && *serverID > 0 {
		return AgentIndexPatternByServerID(*serverID)
	}
	if len(projectServerIDs) == 0 {
		return GlobalAgentIndexPattern()
	}
	if len(projectServerIDs) > maxSearchIndexServers {
		return GlobalAgentIndexPattern()
	}
	parts := make([]string, 0, len(projectServerIDs)*2)
	for _, id := range projectServerIDs {
		if id == 0 {
			continue
		}
		parts = append(parts, AgentIndexPatternByServerID(id))
	}
	if len(parts) == 0 {
		return GlobalAgentIndexPattern()
	}
	return strings.Join(parts, ",")
}

// ResolveSearchIndicesByHosts 按主机拼索引，并附带旧 ID 通配（过渡期检索兼容）。
func ResolveSearchIndicesByHosts(hosts []string, serverIDs []uint) string {
	parts := make([]string, 0, len(hosts)+len(serverIDs))
	seen := map[string]struct{}{}
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		parts = append(parts, p)
	}
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		add(AgentIndexPattern(h))
	}
	for _, id := range serverIDs {
		if id == 0 {
			continue
		}
		add(AgentIndexPatternByServerID(id))
	}
	if len(parts) == 0 {
		return GlobalAgentIndexPattern()
	}
	if len(parts) > maxSearchIndexServers*2 {
		return GlobalAgentIndexPattern()
	}
	return strings.Join(parts, ",")
}

// AgentIndexPrefixForServer 旧版前缀 yunshu-agent-7-
func AgentIndexPrefixForServer(serverID uint) string {
	return fmt.Sprintf("%s-%d-", defaultAgentIndexPrefix, serverID)
}

// AgentIndexPrefixForHost 新版前缀 yunshu-agent-10-10-10-5-
func AgentIndexPrefixForHost(serverHost string) string {
	return fmt.Sprintf("%s-%s-", defaultAgentIndexPrefix, SanitizeHostForName(serverHost))
}

func k8sNamePrefix(prefix string) string {
	p := strings.Trim(strings.TrimSpace(prefix), "-")
	if p == "" {
		p = defaultK8sIndexPrefix
	}
	return p
}

// ClusterLoggieAppName 项目隔离的 DaemonSet / SA 名。
func ClusterLoggieAppName(projectID uint) string {
	return fmt.Sprintf("yunshu-loggie-p%d", projectID)
}

// ClusterLoggieConfigMapName 项目隔离的 ConfigMap 名。
func ClusterLoggieConfigMapName(projectID uint) string {
	return fmt.Sprintf("yunshu-loggie-config-p%d", projectID)
}

// K8sIndexSink Loggie ES sink：yunshu-k8s-{clusterId}-p{projectId}-${+YYYY.MM.DD}
func K8sIndexSink(clusterID, projectID uint) string {
	if projectID == 0 {
		return fmt.Sprintf("%s-%d-${+YYYY.MM.DD}", defaultK8sIndexPrefix, clusterID)
	}
	return fmt.Sprintf("%s-%d-p%d-${+YYYY.MM.DD}", defaultK8sIndexPrefix, clusterID, projectID)
}

// K8sIndexForDay 某日索引名（含项目隔离；projectID=0 为旧格式）。
func K8sIndexForDay(clusterID, projectID uint, day time.Time) string {
	if day.IsZero() {
		day = time.Now().UTC()
	}
	d := day.UTC().Format("2006.01.02")
	if projectID == 0 {
		return fmt.Sprintf("%s-%d-%s", defaultK8sIndexPrefix, clusterID, d)
	}
	return fmt.Sprintf("%s-%d-p%d-%s", defaultK8sIndexPrefix, clusterID, projectID, d)
}

// K8sIndexPattern 单集群检索通配（含各项目分片与旧索引）。
func K8sIndexPattern(clusterID uint) string {
	return fmt.Sprintf("%s-%d-*", defaultK8sIndexPrefix, clusterID)
}

// K8sIndexPatternByProject 单集群单项目检索通配。
func K8sIndexPatternByProject(clusterID, projectID uint) string {
	if projectID == 0 {
		return K8sIndexPattern(clusterID)
	}
	return fmt.Sprintf("%s-%d-p%d-*", defaultK8sIndexPrefix, clusterID, projectID)
}

// GlobalK8sIndexPattern 全量 K8s 日志索引通配。
func GlobalK8sIndexPattern() string {
	return defaultK8sIndexPrefix + "-*"
}

// K8sKafkaTopicTemplate Kafka sink topic 模板（含项目隔离）。
func K8sKafkaTopicTemplate(clusterID, projectID uint, prefix string) string {
	if projectID == 0 {
		return fmt.Sprintf("%s-%d-${+YYYY.MM.DD}", k8sNamePrefix(prefix), clusterID)
	}
	return fmt.Sprintf("%s-%d-p%d-${+YYYY.MM.DD}", k8sNamePrefix(prefix), clusterID, projectID)
}

// K8sKafkaTopicForDay 具体某日 Topic。
func K8sKafkaTopicForDay(clusterID, projectID uint, prefix string, day time.Time) string {
	if day.IsZero() {
		day = time.Now().UTC()
	}
	d := day.UTC().Format("2006.01.02")
	if projectID == 0 {
		return fmt.Sprintf("%s-%d-%s", k8sNamePrefix(prefix), clusterID, d)
	}
	return fmt.Sprintf("%s-%d-p%d-%s", k8sNamePrefix(prefix), clusterID, projectID, d)
}

var (
	// yunshu-k8s-3-p12-2026.08.11 或 yunshu-k8s-3-2026.08.11 或无日期
	k8sTopicRestRE = regexp.MustCompile(`^(\d+)(?:-p(\d+))?(?:-(\d{4}\.\d{2}\.\d{2}))?$`)
)

// IsK8sKafkaTopic 判断是否为集群采集 Topic（含项目隔离与旧格式）。
func IsK8sKafkaTopic(topic, prefix string) bool {
	_, _, ok := ParseK8sKafkaTopicMeta(topic, prefix)
	return ok
}

// ParseK8sKafkaTopicMeta 从 Topic 解析 cluster_id / project_id（旧 Topic projectID=0）。
func ParseK8sKafkaTopicMeta(topic, prefix string) (clusterID, projectID uint, ok bool) {
	p := k8sNamePrefix(prefix)
	topic = strings.TrimSpace(topic)
	head := p + "-"
	if !strings.HasPrefix(topic, head) {
		return 0, 0, false
	}
	rest := strings.TrimPrefix(topic, head)
	m := k8sTopicRestRE.FindStringSubmatch(rest)
	if len(m) < 2 {
		return 0, 0, false
	}
	cid, err := strconv.ParseUint(m[1], 10, 64)
	if err != nil || cid == 0 {
		return 0, 0, false
	}
	if len(m) >= 3 && m[2] != "" {
		pid, err := strconv.ParseUint(m[2], 10, 64)
		if err != nil {
			return 0, 0, false
		}
		projectID = uint(pid)
	}
	return uint(cid), projectID, true
}

// ParseClusterIDFromK8sKafkaTopic 从 Topic 解析 cluster_id。
func ParseClusterIDFromK8sKafkaTopic(topic, prefix string) (uint, bool) {
	cid, _, ok := ParseK8sKafkaTopicMeta(topic, prefix)
	return cid, ok
}
