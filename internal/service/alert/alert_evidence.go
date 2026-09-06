package alert

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/logplatform"
)

// AlertEvidenceResult 告警证据包（日志采样 + 变更 + 维度提示）。
type AlertEvidenceResult struct {
	Fingerprint string            `json:"fingerprint"`
	ProjectID   uint              `json:"project_id,omitempty"`
	Alertname   string            `json:"alertname,omitempty"`
	Severity    string            `json:"severity,omitempty"`
	Status      string            `json:"status,omitempty"` // firing|resolved|unknown
	StartsAt    string            `json:"starts_at,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Dims        alertnotify.Dims  `json:"dims"`

	RecentChanges []model.ChangeEvent     `json:"recent_changes,omitempty"`
	LogOverview   *logplatform.LogOverviewResult `json:"log_overview,omitempty"`
	LogSamples    []AlertEvidenceLogSample `json:"log_samples,omitempty"`
	LogHint       string                  `json:"log_hint,omitempty"`

	// Pod 诊断由前端按 cluster_id/namespace/pod 调用；此处仅给出建议参数。
	PodDiagnoseHint *AlertPodDiagnoseHint `json:"pod_diagnose_hint,omitempty"`
}

type AlertEvidenceLogSample struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level,omitempty"`
	Message   string `json:"message"`
	Host      string `json:"host,omitempty"`
	Pod       string `json:"pod,omitempty"`
}

type AlertPodDiagnoseHint struct {
	ClusterID   uint   `json:"cluster_id,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	Namespace   string `json:"namespace,omitempty"`
	Pod         string `json:"pod,omitempty"`
	Available   bool   `json:"available"`
	Reason      string `json:"reason,omitempty"`
}

// CollectEvidence 按 fingerprint 组装证据包。
func (s *AlertService) CollectEvidence(ctx context.Context, fingerprint string) (*AlertEvidenceResult, error) {
	fp := strings.TrimSpace(fingerprint)
	if fp == "" {
		return nil, constants.ErrBadRequestWithMsg("fingerprint required")
	}
	if s.db == nil {
		return nil, constants.ErrBadRequestWithMsg("db unavailable")
	}

	out := &AlertEvidenceResult{Fingerprint: fp, Labels: map[string]string{}, Status: "unknown"}

	var cur model.AlertCurEvent
	err := s.db.WithContext(ctx).Where("fingerprint = ?", fp).First(&cur).Error
	if err == nil {
		out.Status = "firing"
		out.ProjectID = cur.ProjectID
		out.Alertname = cur.Alertname
		out.Severity = cur.Severity
		out.StartsAt = cur.StartsAt.UTC().Format(time.RFC3339)
		out.Labels = parseLabelsJSON(cur.LabelsJSON)
	} else {
		var his model.AlertHisEvent
		if err2 := s.db.WithContext(ctx).Where("fingerprint = ?", fp).Order("id DESC").First(&his).Error; err2 == nil {
			out.Status = "resolved"
			out.ProjectID = his.ProjectID
			out.Alertname = his.Alertname
			out.Severity = his.Severity
			out.StartsAt = his.StartsAt.UTC().Format(time.RFC3339)
			out.Labels = parseLabelsJSON(his.LabelsJSON)
		}
	}
	if out.ProjectID == 0 {
		if v := strings.TrimSpace(out.Labels["project_id"]); v != "" {
			if n, e := strconv.ParseUint(v, 10, 64); e == nil {
				out.ProjectID = uint(n)
			}
		}
	}
	if out.Alertname == "" {
		out.Alertname = out.Labels["alertname"]
	}
	out.Dims = alertnotify.ExtractDims(out.Labels)
	out.PodDiagnoseHint = buildPodDiagnoseHint(out.Labels, out.Dims)

	if out.ProjectID > 0 {
		anchor := time.Now().UTC()
		if out.StartsAt != "" {
			if t, e := time.Parse(time.RFC3339, out.StartsAt); e == nil {
				anchor = t
			}
		}
		from := anchor.Add(-30 * time.Minute)
		var changes []model.ChangeEvent
		_ = s.db.WithContext(ctx).
			Where("project_id = ? AND started_at >= ? AND started_at <= ?", out.ProjectID, from, anchor.Add(30*time.Minute)).
			Order("id DESC").Limit(10).Find(&changes).Error
		out.RecentChanges = changes

		if s.logSearch != nil {
			sq := logplatform.LogSearchQuery{
				ProjectID: out.ProjectID,
				Level:     "ERROR",
				From:      from.Format(time.RFC3339),
				To:        anchor.Add(5 * time.Minute).Format(time.RFC3339),
				Page:      1,
				PageSize:  8,
			}
			if out.Dims.Namespace != "" {
				sq.Namespace = out.Dims.Namespace
			}
			if out.Dims.Pod != "" {
				sq.Pod = out.Dims.Pod
			}
			if out.Dims.Service != "" {
				sq.ServiceName = out.Dims.Service
			}
			if ov, e := s.logSearch.Overview(ctx, sq); e == nil {
				out.LogOverview = ov
			}
			if res, e := s.logSearch.Search(ctx, sq); e == nil && res != nil {
				for _, it := range res.List {
					out.LogSamples = append(out.LogSamples, AlertEvidenceLogSample{
						Timestamp: it.Timestamp,
						Level:     it.Level,
						Message:   it.Message,
						Host:      firstNonEmpty(it.Host, it.ServerHost),
						Pod:       firstNonEmpty(it.Pod, it.PodName),
					})
				}
			} else if e != nil {
				out.LogHint = e.Error()
			}
			if len(out.LogSamples) == 0 && out.LogHint == "" {
				out.LogHint = "窗口内无 ERROR 日志采样（可点「关联日志」扩大检索）"
			}
		} else {
			out.LogHint = "日志检索服务未接入证据包"
		}
	} else {
		out.LogHint = "告警未绑定 project_id，无法拉取项目日志/变更"
	}

	return out, nil
}

func parseLabelsJSON(raw string) map[string]string {
	out := map[string]string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func buildPodDiagnoseHint(labels map[string]string, dims alertnotify.Dims) *AlertPodDiagnoseHint {
	hint := &AlertPodDiagnoseHint{
		ClusterName: dims.Cluster,
		Namespace:   dims.Namespace,
		Pod:         dims.Pod,
	}
	if v := strings.TrimSpace(labels["cluster_id"]); v != "" {
		if n, e := strconv.ParseUint(v, 10, 64); e == nil {
			hint.ClusterID = uint(n)
		}
	}
	if hint.Namespace != "" && hint.Pod != "" && hint.ClusterID > 0 {
		hint.Available = true
		return hint
	}
	hint.Available = false
	if hint.Pod == "" || hint.Namespace == "" {
		hint.Reason = "labels 缺少 namespace/pod"
	} else if hint.ClusterID == 0 {
		hint.Reason = "缺少 cluster_id（有 cluster 名时请在前端选择集群后诊断）"
	}
	return hint
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
