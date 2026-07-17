package logplatform

import (
	"fmt"
	"strings"
	"time"
)

const (
	defaultAgentIndexPrefix = "yunshu-agent"
	maxSearchIndexServers   = 80
)

// AgentIndexSink 生成 Loggie ES sink 的按日索引名（单服务器）。
func AgentIndexSink(serverID uint) string {
	return fmt.Sprintf("%s-%d-${+YYYY.MM.DD}", defaultAgentIndexPrefix, serverID)
}

// AgentIndexForDay 消费端按日索引名（已解析日期）。
func AgentIndexForDay(serverID uint, day time.Time) string {
	if day.IsZero() {
		day = time.Now().UTC()
	}
	return fmt.Sprintf("%s-%d-%s", defaultAgentIndexPrefix, serverID, day.UTC().Format("2006.01.02"))
}

// AgentKafkaTopic 每个 Agent 独立 Kafka Topic：{prefix}-{server_id}。
func AgentKafkaTopic(serverID uint, prefix string) string {
	p := strings.Trim(strings.TrimSpace(prefix), "-")
	if p == "" {
		p = defaultAgentIndexPrefix
	}
	return fmt.Sprintf("%s-%d", p, serverID)
}

// ParseServerIDFromAgentKafkaTopic 从 yunshu-agent-7 解析 server_id。
func ParseServerIDFromAgentKafkaTopic(topic, prefix string) (uint, bool) {
	p := strings.Trim(strings.TrimSpace(prefix), "-")
	if p == "" {
		p = defaultAgentIndexPrefix
	}
	topic = strings.TrimSpace(topic)
	head := p + "-"
	if !strings.HasPrefix(topic, head) {
		return 0, false
	}
	rest := strings.TrimPrefix(topic, head)
	if rest == "" {
		return 0, false
	}
	var id uint64
	for _, ch := range rest {
		if ch < '0' || ch > '9' {
			return 0, false
		}
		id = id*10 + uint64(ch-'0')
	}
	if id == 0 {
		return 0, false
	}
	return uint(id), true
}

// AgentIndexPattern 单服务器检索通配。
func AgentIndexPattern(serverID uint) string {
	return fmt.Sprintf("%s-%d-*", defaultAgentIndexPrefix, serverID)
}

// GlobalAgentIndexPattern 全量 Agent 索引通配（保留策略 / 兜底检索）。
func GlobalAgentIndexPattern() string {
	return defaultAgentIndexPrefix + "-*"
}

// ResolveSearchIndices 按服务器筛选拼 ES multi-index；超限时回退全局通配。
func ResolveSearchIndices(serverID *uint, projectServerIDs []uint) string {
	if serverID != nil && *serverID > 0 {
		return AgentIndexPattern(*serverID)
	}
	if len(projectServerIDs) == 0 {
		return GlobalAgentIndexPattern()
	}
	if len(projectServerIDs) > maxSearchIndexServers {
		return GlobalAgentIndexPattern()
	}
	parts := make([]string, 0, len(projectServerIDs))
	for _, id := range projectServerIDs {
		if id == 0 {
			continue
		}
		parts = append(parts, AgentIndexPattern(id))
	}
	if len(parts) == 0 {
		return GlobalAgentIndexPattern()
	}
	return strings.Join(parts, ",")
}

// AgentIndexPrefixForServer 项目级保留策略过滤前缀，如 yunshu-agent-7-
func AgentIndexPrefixForServer(serverID uint) string {
	return fmt.Sprintf("%s-%d-", defaultAgentIndexPrefix, serverID)
}
