package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

const routingWizardTreeProjectID uint = 0

type AlertRoutingWizardRequest struct {
	ProjectID   uint     `json:"project_id"`
	Severity    string   `json:"severity"`
	ChannelIDs  []uint   `json:"channel_ids"`
	ExtraEmails []string `json:"extra_emails"`
	Name        string   `json:"name"`
}

type AlertRoutingWizardResult struct {
	ReceiverGroup *model.AlertReceiverGroup  `json:"receiver_group"`
	Node          *model.AlertSubscriptionNode `json:"node"`
	RootCreated   bool                       `json:"root_created"`
}

func (s *AlertSubscriptionService) ApplyRoutingWizard(ctx context.Context, req AlertRoutingWizardRequest) (*AlertRoutingWizardResult, error) {
	if s == nil || s.groups == nil {
		return nil, constants.ErrInternal
	}
	channelIDs := uniquePositiveUints(req.ChannelIDs)
	if len(channelIDs) == 0 {
		return nil, constants.ErrBadRequestWithMsg("请至少选择一个通知通道")
	}
	severity := normalizeWizardSeverity(req.Severity)
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = wizardNodeName(req.ProjectID, severity)
	}
	chJSON, err := json.Marshal(channelIDs)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.wizard", "ApplyRoutingWizard", err)
	}
	emailJSON, err := json.Marshal(uniqueEmails(req.ExtraEmails))
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.wizard", "ApplyRoutingWizard", err)
	}
	enabled := true
	group, err := s.groups.Create(ctx, AlertReceiverGroupUpsertRequest{
		ProjectID:           routingWizardTreeProjectID,
		Name:                name + " 接收组",
		Description:         "由路由向导创建",
		ChannelIDsJSON:      string(chJSON),
		EmailRecipientsJSON: string(emailJSON),
		Enabled:             &enabled,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.wizard", "ApplyRoutingWizard", err)
	}
	groupIDsJSON, err := json.Marshal([]uint{group.ID})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.wizard", "ApplyRoutingWizard", err)
	}

	rootID, rootCreated, err := s.ensureGlobalRoutingRoot(ctx)
	if err != nil {
		return nil, err
	}
	parentID := rootID
	labels := map[string]string{}
	if req.ProjectID > 0 {
		labels["project_id"] = fmt.Sprintf("%d", req.ProjectID)
	}
	labelsJSON := "{}"
	if len(labels) > 0 {
		b, mErr := json.Marshal(labels)
		if mErr != nil {
			return nil, bizerrors.Pass(ctx, "alert.wizard", "ApplyRoutingWizard", mErr)
		}
		labelsJSON = string(b)
	}
	notifyResolved := true
	node, err := s.CreateNode(ctx, AlertSubscriptionNodeUpsertRequest{
		ProjectID:            routingWizardTreeProjectID,
		ParentID:             &parentID,
		Name:                 name,
		MatchLabelsJSON:      labelsJSON,
		MatchRegexJSON:       "{}",
		MatchSeverity:        severity,
		Continue:             true,
		Enabled:              &enabled,
		ReceiverGroupIDsJSON: string(groupIDsJSON),
		NotifyResolved:       &notifyResolved,
	})
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.wizard", "ApplyRoutingWizard", err)
	}
	return &AlertRoutingWizardResult{
		ReceiverGroup: group,
		Node:          node,
		RootCreated:   rootCreated,
	}, nil
}

func (s *AlertSubscriptionService) ensureGlobalRoutingRoot(ctx context.Context) (uint, bool, error) {
	tree, err := s.GetNodeTree(ctx, routingWizardTreeProjectID)
	if err != nil {
		return 0, false, bizerrors.Pass(ctx, "alert.wizard", "ensureGlobalRoutingRoot", err)
	}
	for _, n := range tree {
		if n.ID > 0 {
			return n.ID, false, nil
		}
	}
	enabled := true
	notifyResolved := true
	root, err := s.CreateNode(ctx, AlertSubscriptionNodeUpsertRequest{
		ProjectID:            routingWizardTreeProjectID,
		Name:                 "全局根",
		Enabled:              &enabled,
		Continue:             true,
		MatchLabelsJSON:      "{}",
		MatchRegexJSON:       "{}",
		ReceiverGroupIDsJSON: "[]",
		NotifyResolved:       &notifyResolved,
	})
	if err != nil {
		return 0, false, bizerrors.Pass(ctx, "alert.wizard", "ensureGlobalRoutingRoot", err)
	}
	return root.ID, true, nil
}

func wizardNodeName(projectID uint, severity string) string {
	sev := severity
	if sev == "" {
		sev = "全部级别"
	}
	if projectID == 0 {
		return "向导 · " + sev
	}
	return fmt.Sprintf("向导 · 项目 %d · %s", projectID, sev)
}

func normalizeWizardSeverity(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch s {
	case "critical", "warning", "info":
		return s
	case "critical,warning", "warning,critical":
		return "critical,warning"
	default:
		return strings.TrimSpace(raw)
	}
}

func uniquePositiveUints(in []uint) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(in))
	for _, id := range in {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func uniqueEmails(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, e := range in {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}
