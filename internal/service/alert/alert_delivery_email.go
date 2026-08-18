package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

const (
	RecipientModeAssigneeOnly  = "assignee_only"
	RecipientModeAssigneeAndCC = "assignee_and_cc"
	RecipientModeChannelOnly   = "channel_only"
)

func normalizeRecipientMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case RecipientModeAssigneeOnly:
		return RecipientModeAssigneeOnly
	case RecipientModeChannelOnly:
		return RecipientModeChannelOnly
	case RecipientModeAssigneeAndCC, "assignee+cc", "handler_and_cc", "", "<nil>":
		// 空值默认：处理人 + 抄送通道
		return RecipientModeAssigneeAndCC
	default:
		return RecipientModeAssigneeAndCC
	}
}

func payloadRecipientMode(payload map[string]interface{}) string {
	if payload == nil {
		return RecipientModeAssigneeAndCC
	}
	return normalizeRecipientMode(fmt.Sprintf("%v", payload["recipient_mode"]))
}

func extractPayloadEmails(payload map[string]interface{}, key string) []string {
	if payload == nil {
		return nil
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return nil
	}
	var out []string
	seen := map[string]struct{}{}
	for _, e := range normalizeRecipientList(raw) {
		s := strings.TrimSpace(strings.ToLower(e))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func dedupeEmailList(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
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

func mergeEmailLists(parts ...[]string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(e string) {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			return
		}
		if _, ok := seen[e]; ok {
			return
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	for _, part := range parts {
		for _, e := range part {
			add(e)
		}
	}
	return out
}

func mergeAssigneeEmails(recipients []string, payload map[string]interface{}) []string {
	mode := payloadRecipientMode(payload)
	assignee := extractPayloadEmails(payload, "assignee_emails")
	channel := dedupeEmailList(recipients)

	switch mode {
	case RecipientModeChannelOnly:
		return channel
	case RecipientModeAssigneeOnly:
		if len(assignee) > 0 {
			return assignee
		}
		return channel
	default: // assignee_and_cc
		if len(assignee) == 0 {
			return channel
		}
		return mergeEmailLists(assignee, channel)
	}
}

func payloadHasAssigneeEmails(payload map[string]interface{}) bool {
	return len(extractPayloadEmails(payload, "assignee_emails")) > 0
}

// mergeAssigneeEmailsWithReceiverGroup 合并接收组抄送。
// assignee_only 且已有处理人时不合并；assignee_and_cc / channel_only 会合并。
func mergeAssigneeEmailsWithReceiverGroup(recipients []string, payload map[string]interface{}) []string {
	mode := payloadRecipientMode(payload)
	if mode == RecipientModeAssigneeOnly && payloadHasAssigneeEmails(payload) {
		return recipients
	}
	extra := extractPayloadEmails(payload, "receiver_group_emails")
	if len(extra) == 0 {
		return recipients
	}
	return mergeEmailLists(recipients, extra)
}

func (s *AlertService) sendEmailChannel(ctx context.Context, channel *model.AlertChannel, source, title, severity, status string, payload map[string]interface{}) (int, string, error) {
	recipients, err := parseEmailRecipients(channel.HeadersJSON)
	if err != nil {
		return 0, "", bizerrors.Pass(ctx, "alert.delivery", "sendEmailChannel", err)
	}
	recipients = mergeAssigneeEmails(recipients, payload)
	recipients = mergeAssigneeEmailsWithReceiverGroup(recipients, payload)
	if len(recipients) == 0 {
		return 0, "", constants.ErrBadRequestWithMsg(constants.ErrMsgc47e8ed41463)
	}
	if s.mailer == nil || !s.mailer.Enabled() {
		msg := constants.ErrMsg71c5fe1e9994
		return 0, "", bizerrors.InternalCtx(ctx, fmt.Errorf("%s", msg), "api: "+msg)
	}
	settings, err := parseChannelSettings(channel.HeadersJSON)
	if err != nil {
		return 0, "", bizerrors.Pass(ctx, "alert.delivery", "sendEmailChannel", err)
	}
	subject := strings.TrimSpace(title)
	mdBody := s.renderChannelMessage(ctx, title, severity, status, payload, settings)
	htmlBody := alertnotify.MarkdownToHTML(mdBody)
	var failMsgs []string
	okCount := 0
	for _, to := range recipients {
		if err := s.mailer.SendMultipart(ctx, to, subject, mdBody, htmlBody); err != nil {
			failMsgs = append(failMsgs, fmt.Sprintf("%s: %v", to, err))
		} else {
			okCount++
		}
	}
	var sendErr error
	if okCount == 0 && len(recipients) > 0 {
		sendErr = fmt.Errorf("%s", strings.Join(failMsgs, "; "))
	} else if len(failMsgs) > 0 {
		sendErr = fmt.Errorf("partial failure: %s", strings.Join(failMsgs, "; "))
	}
	storeMap := make(map[string]interface{}, len(payload)+4)
	for k, v := range payload {
		storeMap[k] = v
	}
	storeMap["to"] = recipients
	alertdispatch.SlimOutgoingPayloadForHistory(storeMap, s.cfg.MaxPayloadChars)
	reqBytes, _ := json.Marshal(storeMap)
	respNote := "email sent"
	if okCount > 0 && len(failMsgs) > 0 {
		respNote = fmt.Sprintf("email sent: %d ok, %d failed", okCount, len(failMsgs))
	}
	event := model.AlertEvent{
		Source:             source,
		Title:              title,
		Severity:           severity,
		Status:             status,
		Cluster:            alertnotify.StringFromPayload(payload, "cluster"),
		MonitorPipeline:    strings.TrimSpace(alertnotify.StringFromPayload(payload, "monitorPipeline")),
		GroupKey:           alertnotify.StringFromPayload(payload, "groupKey"),
		LabelsDigest:       alertnotify.StringFromPayload(payload, "labelsDigest"),
		MatchedPolicyIDs:   alertnotify.StringFromPayload(payload, "matchedPolicyIds"),
		MatchedPolicyNames: alertnotify.StringFromPayload(payload, "matchedPolicyNames"),
		ChannelID:          channel.ID,
		ChannelName:        channel.Name,
		Success:            okCount > 0,
		HTTPStatusCode:     200,
		RequestPayload:     truncateText(string(buildEventPayloadBytes(reqBytes, payload, s.cfg.MaxPayloadChars)), s.cfg.MaxPayloadChars),
		ResponsePayload:    truncateText(respNote, s.cfg.MaxPayloadChars),
	}
	fillAlertEventDatasourceFromPayload(&event, payload)
	if sendErr != nil && okCount == 0 {
		event.HTTPStatusCode = 500
		event.Success = false
		event.ErrorMessage = truncateText(sendErr.Error(), 1000)
		event.ResponsePayload = ""
	}
	_ = s.persistAlertEvent(ctx, &event)
	if okCount == 0 && sendErr != nil {
		return 500, "", sendErr
	}
	return 200, respNote, nil
}
