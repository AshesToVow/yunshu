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
