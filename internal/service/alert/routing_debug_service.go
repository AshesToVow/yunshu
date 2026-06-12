package alert

import (
	"context"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

type AlertRoutingDebugRequest struct {
	ProjectID uint              `json:"project_id" binding:"required"`
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
	Matched              bool                       `json:"matched"`
	MatchedPath          string                     `json:"matched_path,omitempty"`
	MatchedNodeNames     []string                   `json:"matched_node_names,omitempty"`
	ReceiverGroupIDs     []uint                     `json:"receiver_group_ids,omitempty"`
	SilenceSeconds       int                        `json:"silence_seconds,omitempty"`
	Channels             []AlertRoutingDebugChannel `json:"channels,omitempty"`
	Silenced             bool                       `json:"silenced"`
	SilenceID            uint                       `json:"silence_id,omitempty"`
	MaintenanceSuppressed bool                      `json:"maintenance_suppressed"`
	MaintenanceID        uint                       `json:"maintenance_id,omitempty"`
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
	route, matched := s.subscriptionSvc.MatchRouteDetailed(ctx, req.ProjectID, labels, severity, status)
	out.Matched = matched
	if matched {
		out.MatchedPath = route.MatchedPath
		out.MatchedNodeNames = route.MatchedNodeNames
		out.ReceiverGroupIDs = route.ReceiverGroupIDs
		out.SilenceSeconds = route.SilenceSeconds
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
			for _, gid := range route.ReceiverGroupIDs {
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
