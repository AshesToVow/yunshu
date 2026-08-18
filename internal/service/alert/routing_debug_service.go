package alert

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

type AlertRoutingDebugRequest struct {
	ProjectID uint              `json:"project_id"`
	Labels    map[string]string `json:"labels" binding:"required"`
	Severity  string            `json:"severity"`
	Status    string            `json:"status"`
}

type AlertRoutingDebugChannel struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type AlertRoutingDebugResult struct {
	Matched               bool                       `json:"matched"`
	MatchedPath           string                     `json:"matched_path,omitempty"`
	MatchedNodeNames      []string                   `json:"matched_node_names,omitempty"`
	ReceiverGroupIDs      []uint                     `json:"receiver_group_ids,omitempty"`
	SilenceSeconds        int                        `json:"silence_seconds,omitempty"`
	Channels              []AlertRoutingDebugChannel `json:"channels,omitempty"`
	MatchedFromProject    bool                       `json:"matched_from_project,omitempty"`
	MatchedFromGlobal     bool                       `json:"matched_from_global,omitempty"`
	Silenced              bool                       `json:"silenced"`
	SilenceID             uint                       `json:"silence_id,omitempty"`
	MaintenanceSuppressed bool                       `json:"maintenance_suppressed"`
	MaintenanceID         uint                       `json:"maintenance_id,omitempty"`
	Hint                  string                     `json:"hint,omitempty"`
}

func (s *AlertService) DebugRouting(ctx context.Context, req AlertRoutingDebugRequest) (*AlertRoutingDebugResult, error) {
	if s.subscriptionSvc == nil {
		return nil, constants.ErrInternalWithMsg("订阅路由服务未初始化")
	}
	severity := strings.TrimSpace(req.Severity)
	if severity == "" {
		severity = "warning"
	}
	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status == "" {
		status = "firing"
	}
	labels := map[string]string{}
	for k, v := range req.Labels {
		labels[k] = strings.TrimSpace(v)
	}
	out := &AlertRoutingDebugResult{}
	now := time.Now()
	if s.maintenanceSvc != nil {
		if mid, ok, err := s.maintenanceSvc.FirstMatchingID(ctx, labels, now); err == nil && ok {
			out.MaintenanceSuppressed = true
			out.MaintenanceID = mid
			out.Silenced = true
		}
	}
	if s.silenceSvc != nil {
		if sid, ok, err := s.silenceSvc.FirstMatchingSilenceID(ctx, labels, now); err == nil && ok {
			out.Silenced = true
			out.SilenceID = sid
		}
	}

	// 与真实投递 channelIDSetForAlert 一致：当前项目 + 全局 project_id=0 都会参与匹配并合并。
	var merged AlertRouteResult
	try := func(pid uint, tag string) {
		route, ok := s.subscriptionSvc.MatchRouteDetailed(ctx, pid, labels, severity, status)
		if !ok || len(route.ReceiverGroupIDs) == 0 {
			return
		}
		if pid == 0 {
			out.MatchedFromGlobal = true
		} else {
			out.MatchedFromProject = true
		}
		for _, n := range route.MatchedNodeNames {
			merged.MatchedNodeNames = append(merged.MatchedNodeNames, fmt.Sprintf("%s:%s", tag, n))
		}
		merged.MatchedNodeIDs = append(merged.MatchedNodeIDs, route.MatchedNodeIDs...)
		merged.ReceiverGroupIDs = append(merged.ReceiverGroupIDs, route.ReceiverGroupIDs...)
		if route.SilenceSeconds > merged.SilenceSeconds {
			merged.SilenceSeconds = route.SilenceSeconds
		}
		if merged.MatchedPath == "" && route.MatchedPath != "" {
			merged.MatchedPath = route.MatchedPath
		}
	}
	if req.ProjectID > 0 {
		try(req.ProjectID, fmt.Sprintf("project=%d", req.ProjectID))
	}
	try(0, "global")

	merged.ReceiverGroupIDs = uniqUint(merged.ReceiverGroupIDs)
	merged.MatchedNodeIDs = uniqUint(merged.MatchedNodeIDs)
	merged.MatchedNodeNames = uniqStrings(merged.MatchedNodeNames)
	out.Matched = len(merged.ReceiverGroupIDs) > 0
	out.MatchedPath = merged.MatchedPath
	out.MatchedNodeNames = merged.MatchedNodeNames
	out.ReceiverGroupIDs = merged.ReceiverGroupIDs
	out.SilenceSeconds = merged.SilenceSeconds
	if out.MatchedFromGlobal && !out.MatchedFromProject {
		out.Hint = "命中来自全局订阅(project_id=0)。当前项目树里停用节点不会影响全局路由，请检查全局/其它项目的订阅树。"
	} else if out.MatchedFromGlobal && out.MatchedFromProject {
		out.Hint = "当前项目与全局订阅均命中，通道会合并投递。"
	}

	if out.Matched {
		channels, err := s.loadEnabledChannels(ctx)
		if err != nil {
			return nil, err
		}
		byID := map[uint]model.AlertChannel{}
		for _, ch := range channels {
			byID[ch.ID] = ch
		}
		seen := map[uint]struct{}{}
		if s.receiverGroupCache != nil {
			for _, gid := range merged.ReceiverGroupIDs {
				g, err := s.receiverGroupCache.Get(gid)
				if err != nil || g == nil || !g.IsActiveNow() {
					continue
				}
				for _, cid := range g.ChannelIDs {
					if cid == 0 {
						continue
					}
					if _, ok := seen[cid]; ok {
						continue
					}
					seen[cid] = struct{}{}
					if ch, ok := byID[cid]; ok {
						out.Channels = append(out.Channels, AlertRoutingDebugChannel{ID: ch.ID, Name: ch.Name, Type: ch.Type})
					}
				}
			}
		}
	}
	return out, nil
}
