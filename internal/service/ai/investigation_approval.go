package ai

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
)

// attachInvestigationApprovals 将调查报告中的写动作转为审批单，闭合「采集→分析→待审批」。
// 成功创建至少一张审批单时状态为 awaiting_approval，并写入 ApprovalID（首张）。
func (s *Service) attachInvestigationApprovals(
	ctx context.Context,
	userID uint,
	actor *auth.CurrentUser,
	row *model.AiInvestigation,
	report *InvestigationReport,
	req StartInvestigationRequest,
) {
	if s.repo == nil || row == nil || report == nil {
		return
	}
	cands := extractInvestigationWriteCandidates(row.Kind, req, report)
	if len(cands) == 0 {
		return
	}
	var firstID uint
	for _, c := range cands {
		argsRaw, _ := json.Marshal(c.Args)
		out, err := s.createToolApproval(ctx, userID, c.Tool, string(argsRaw), c.ClusterID, c.Namespace, c.Resource, c.Reason)
		if err != nil {
			report.Actions = append(report.Actions, map[string]any{
				"action": "approval_failed", "tool": c.Tool, "error": err.Error(),
			})
			continue
		}
		aid := uintFromAny(out["approval_id"], 0)
		if firstID == 0 && aid > 0 {
			firstID = aid
		}
		report.Actions = append(report.Actions, map[string]any{
			"action":      "pending_approval",
			"tool":        c.Tool,
			"approval_id": aid,
			"hint":        "已创建审批单，通过后可在 AI 操作审批或统一待办中执行",
		})
	}
	if firstID == 0 {
		return
	}
	row.ApprovalID = &firstID
	row.Status = "awaiting_approval"
	row.UpdatedAt = time.Now()
	// AnalysisJSON / ReportJSON 由 StartInvestigation 统一落库，避免被覆盖丢 approvals
	_ = actor
}

type investigationWriteCandidate struct {
	Tool      string
	Args      map[string]any
	ClusterID uint
	Namespace string
	Resource  string
	Reason    string
}

func extractInvestigationWriteCandidates(
	_ string,
	req StartInvestigationRequest,
	report *InvestigationReport,
) []investigationWriteCandidate {
	seen := map[string]struct{}{}
	var out []investigationWriteCandidate
	add := func(c investigationWriteCandidate) {
		key := c.Tool + "|" + c.Resource + "|" + c.Namespace
		if fp, ok := c.Args["fingerprint"].(string); ok && fp != "" {
			key = c.Tool + "|" + fp
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}

	// 告警调查：不再默认建静默审批（避免噪音）；仅当报告 actions 显式要求时建单
	for _, a := range report.Actions {
		tool := normalizeWriteToolName(actionField(a, "tool", "action", "name"))
		if tool == "" || !isApprovalBackedWriteTool(tool) {
			continue
		}
		args := map[string]any{}
		for k, v := range a {
			args[k] = v
		}
		clusterID := uintFromAny(a["cluster_id"], req.ClusterID)
		ns := strings.TrimSpace(actionField(a, "namespace"))
		if ns == "" {
			ns = strings.TrimSpace(req.Namespace)
		}
		name := strings.TrimSpace(actionField(a, "name", "resource", "deployment", "pod"))
		if name == "" {
			name = strings.TrimSpace(req.Resource)
		}
		fp := strings.TrimSpace(actionField(a, "fingerprint"))
		if fp == "" {
			fp = strings.TrimSpace(req.Fingerprint)
		}
		reason := strings.TrimSpace(actionField(a, "reason", "hint", "comment"))
		if reason == "" {
			reason = "调查建议执行 " + tool
		}
		switch tool {
		case "create_alert_silence":
			pid := uintFromAny(a["project_id"], req.ProjectID)
			if fp == "" || pid == 0 {
				continue
			}
			args["project_id"] = pid
			args["fingerprint"] = fp
			if _, ok := args["hours"]; !ok {
				args["hours"] = 2
			}
			add(investigationWriteCandidate{
				Tool: tool, Args: args, ClusterID: clusterID, Resource: fp, Reason: reason,
			})
		case "scale_deployment", "restart_deployment", "delete_pod":
			if clusterID == 0 || ns == "" || name == "" {
				continue
			}
			args["cluster_id"] = clusterID
			args["namespace"] = ns
			args["name"] = name
			if tool == "scale_deployment" {
				if _, ok := args["replicas"]; !ok {
					continue // 缺副本数不建单
				}
			}
			args["reason"] = reason
			add(investigationWriteCandidate{
				Tool: tool, Args: args, ClusterID: clusterID, Namespace: ns, Resource: name, Reason: reason,
			})
		default:
			// 脚本等：保留 args，resource 用 tool 名
			add(investigationWriteCandidate{
				Tool: tool, Args: args, ClusterID: clusterID, Namespace: ns, Resource: name, Reason: reason,
			})
		}
	}
	return out
}

func isApprovalBackedWriteTool(name string) bool {
	switch name {
	case "scale_deployment", "restart_deployment", "delete_pod", "create_alert_silence":
		return true
	default:
		return false
	}
}

func normalizeWriteToolName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	switch s {
	case "silence", "alert_silence", "create_silence", "静默":
		return "create_alert_silence"
	case "scale", "扩缩容", "scale_deploy":
		return "scale_deployment"
	case "restart", "重启", "restart_deploy":
		return "restart_deployment"
	case "delete", "删pod", "deletepod":
		return "delete_pod"
	default:
		return s
	}
}

func actionField(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}
	return ""
}

func uintFromAny(v any, fallback uint) uint {
	switch n := v.(type) {
	case float64:
		return uint(n)
	case int:
		return uint(n)
	case int64:
		return uint(n)
	case uint:
		return n
	case json.Number:
		i, _ := n.Int64()
		return uint(i)
	default:
		return fallback
	}
}
