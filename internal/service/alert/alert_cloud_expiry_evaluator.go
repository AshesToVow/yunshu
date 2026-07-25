package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/service/cmdb"
)

func (s *AlertService) tickCloudExpiryRules(ctx context.Context) error {
	return s.tickCloudExpiryRulesWithMode(ctx, false)
}

// cloudExpiryEvalLeaderKey 云到期定时评估的集群单实例锁。与内置监控规则的 alert:monitor:eval:leader 同理：
// 多副本部署时若无此锁，每个副本都会各自扫描同一规则、调用云 API 并推送告警，导致重复通知
//（云到期告警 SkipGroupTiming=true，会绕过分组节流，重复无法在下游被抑制）。
const cloudExpiryEvalLeaderKey = "alert:cloud-expiry:eval:leader"

// acquireCloudExpiryLeader 尝试成为本轮定时评估的唯一执行者。无 Redis（单机）时恒为 true。
// TTL 仅作为持锁副本崩溃时的兜底，正常路径由 release 在扫描结束后主动释放，不阻塞下一分钟的调度。
func (s *AlertService) acquireCloudExpiryLeader(ctx context.Context) (bool, func()) {
	if s.redis == nil {
		return true, func() {}
	}
	ttl := 5 * time.Minute
	ok, err := s.redis.SetNX(ctx, cloudExpiryEvalLeaderKey, "1", ttl).Result()
	if err != nil || !ok {
		return false, func() {}
	}
	return true, func() {
		// 用独立 context 释放，避免调度 ctx 已取消时无法删除锁（否则要等 TTL 才过期）。
		relCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.redis.Del(relCtx, cloudExpiryEvalLeaderKey).Err()
	}
}

func (s *AlertService) tickCloudExpiryRulesWithMode(ctx context.Context, force bool) error {
	if !force {
		// 定时评估：集群内仅一个副本执行本轮扫描；手动「立即评估」(force) 不受此限制，直接在处理请求的副本上运行。
		ok, release := s.acquireCloudExpiryLeader(ctx)
		if !ok {
			return nil
		}
		defer release()
	}
	rules, err := s.cloudExpiryRepo.ListEnabled(ctx)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.cloud-expiry", "tickCloudExpiryRulesWithMode", err)
	}
	if !force && s.aead == nil {
		if len(rules) > 0 {
			alertLog().Info("Skipped cloud expiry tick", "reason", "no_encryption_key", "enabled_rules", len(rules))
		}
		// 定时评估依赖解密云账号 AK/SK；未配置 encryption_key 时不跑规则，也不推进 last_eval，避免「看起来在调度、实际无拉云」。
		return nil
	}
	now := time.Now()
	var skipNoCron int
	for i := range rules {
		rule := &rules[i]
		if !force && !rule.ScheduleEnabled {
			continue
		}
		syntheticID := uint(1000000) + rule.ID
		if !force {
			cronSpec := strings.TrimSpace(rule.EvalCronSpec)
			if cronSpec == "" {
				skipNoCron++
				continue
			}
			var last time.Time
			var hasLast bool
			if s.redis != nil {
				last, hasLast = s.redisLastEvalTime(ctx, syntheticID)
			} else {
				last, hasLast = s.cloudExpiryLocalLastEval(syntheticID)
			}
			if !ShouldEvalCloudExpiryByCron(cronSpec, last, hasLast, now) {
				continue
			}
		}
		if !force {
			alertLog().Info("Scheduled cloud expiry rule evaluation", "rule_id", rule.ID, "name", rule.Name, "cron", strings.TrimSpace(rule.EvalCronSpec))
		}
		s.evaluateOneCloudExpiryRule(ctx, rule, now, force)
		if !force {
			if s.redis != nil {
				s.redisTouchLastEval(ctx, syntheticID, now)
			} else {
				s.touchCloudExpiryNoRedisLastEval(syntheticID, now)
			}
		}
	}
	if !force && skipNoCron > 0 {
		alertLog().Info("Cloud expiry tick completed", "skipped_empty_cron", skipNoCron)
	}
	return nil
}

// EvaluateCloudExpiryRulesNow 手动触发一次云到期规则评估。
func (s *AlertService) EvaluateCloudExpiryRulesNow(ctx context.Context) error {
	if s.aead == nil {
		return constants.ErrBadRequestWithMsg(
			"未配置 security.encryption_key（或与保存云账号凭据时使用的密钥不一致），无法解密 AK/SK，云到期规则不会拉取云实例。配置密钥后重试「立即评估」。")
	}
	return s.tickCloudExpiryRulesWithMode(ctx, true)
}

func (s *AlertService) evaluateOneCloudExpiryRule(ctx context.Context, rule *model.CloudExpiryRule, now time.Time, manualEval bool) {
	if s.aead == nil {
		return
	}
	alertLog().Info("Started cloud expiry rule evaluation", "rule_id", rule.ID, "name", rule.Name, "manual", manualEval)
	instScanned := 0
	providerFilter := strings.TrimSpace(rule.Provider)
	regionFilter := parseRegionSet(rule.RegionScope)
	accounts, err := s.cloudAccountRepo.ListEnabledByProject(ctx, rule.ProjectID, providerFilter)
	if err != nil {
		return
	}
	for i := range accounts {
		acc := &accounts[i]
		if acc.EncAK == nil || acc.EncSK == nil {
			continue
		}
		ak, err := cryptox.DecryptString(s.aead, *acc.EncAK)
		if err != nil {
			continue
		}
		sk, err := cryptox.DecryptString(s.aead, *acc.EncSK)
		if err != nil {
			continue
		}
		provider, err := cmdb.CloudProviderByName(strings.TrimSpace(acc.Provider))
		if err != nil {
			continue
		}
		scope := strings.TrimSpace(acc.RegionScope)
		if ruleScope := strings.TrimSpace(rule.RegionScope); ruleScope != "" {
			scope = ruleScope
		}
		instances, err := provider.ListInstances(ctx, ak, sk, scope)
		if err != nil {
			continue
		}
		for _, ins := range instances {
			instanceID := strings.TrimSpace(ins.InstanceID)
			if instanceID == "" {
				continue
			}
			region := strings.TrimSpace(ins.Region)
			if len(regionFilter) > 0 {
				if strings.EqualFold(strings.TrimSpace(acc.Provider), "tencent") {
					if !cmdb.InstanceMatchesTencentRegionFilter(region, regionFilter) {
						continue
					}
				} else if _, ok := regionFilter[region]; !ok {
					continue
				}
			}
			expireAt, err := provider.QueryInstanceExpireAt(ctx, ak, sk, region, instanceID)
			if err != nil || expireAt == nil {
				continue
			}
			instScanned++
			daysLeft := int(math.Ceil(expireAt.Sub(now).Hours() / 24))
			firing := daysLeft <= maxInt(1, rule.AdvanceDays)
			fp := fmt.Sprintf("cloud_expiry_rule_%d_%s", rule.ID, instanceID)
			labels := map[string]string{
				"alertname":        strings.TrimSpace(rule.Name),
				"severity":         strings.TrimSpace(rule.Severity),
				"source":           "cloud_expiry",
				"project_id":       fmt.Sprintf("%d", rule.ProjectID),
				"provider":         strings.TrimSpace(acc.Provider),
				"cloud_account_id": fmt.Sprintf("%d", acc.ID),
				"instance_id":      instanceID,
				"instance_name":    strings.TrimSpace(ins.Name),
				"region":           region,
			}
			if labels["severity"] == "" {
				labels["severity"] = "warning"
			}
			if raw := strings.TrimSpace(rule.LabelsJSON); raw != "" && raw != "{}" {
				var obj map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &obj); err == nil {
					for k, v := range obj {
						labels[strings.TrimSpace(k)] = strings.TrimSpace(fmt.Sprintf("%v", v))
					}
				}
			}
			annotations := map[string]string{
				"summary":     fmt.Sprintf("云服务器到期提醒：%s/%s 剩余 %d 天", strings.TrimSpace(acc.Provider), instanceID, daysLeft),
				"description": fmt.Sprintf("实例=%s(%s)，区域=%s，到期时间=%s，剩余天数=%d", strings.TrimSpace(ins.Name), instanceID, region, expireAt.Format(time.RFC3339), daysLeft),
				"value":       fmt.Sprintf("%d", daysLeft),
			}
			cloud := &CloudExpiryExtension{
				Provider:     strings.TrimSpace(acc.Provider),
				AccountID:    acc.ID,
				InstanceID:   instanceID,
				InstanceName: strings.TrimSpace(ins.Name),
				Region:       region,
				ExpiresAt:    *expireAt,
				DaysLeft:     daysLeft,
				ProjectID:    rule.ProjectID,
			}
			s.emitCloudExpiryAlert(ctx, fp, firing, labels, annotations, now, cloud)
		}
	}
	alertLog().Info("Finished cloud expiry rule evaluation", "rule_id", rule.ID, "instances_checked", instScanned)
}

func (s *AlertService) emitCloudExpiryAlert(ctx context.Context, fp string, firing bool, labels, annotations map[string]string, now time.Time, cloud *CloudExpiryExtension) {
	s.monitorEvalMu.Lock()
	active := s.cloudExpiryState[fp]
	if firing {
		// 不在此处短路「已 firing」：否则首次入站若未匹配订阅/通道失败，会永久不再重试。
		// 持续 firing 时的外发频率由 ingest 层对 cloud_expiry + SkipGroupTiming 叠加 repeat_interval 控制。
		s.cloudExpiryState[fp] = true
		s.monitorEvalMu.Unlock()
		item := NewCanonicalAlert(
			IngressSourceCloudExpiry,
			"cloud-expiry",
			"firing",
			map[string]string{"alertname": labels["alertname"]},
			labels,
			IngressAlertDetail{
				Status:          "firing",
				Labels:          labels,
				Annotations:     annotations,
				StartsAt:        now,
				EndsAt:          now.Add(24 * time.Hour),
				Fingerprint:     fp,
				SkipGroupTiming: true,
			},
		)
		item.Cloud = cloud
		_ = s.receiveCanonicalSync(ctx, item)
		alertLog().Info("Emitted cloud expiry firing alert", "fingerprint", fp, "alertname", labels["alertname"])
		return
	}
	if !active {
		s.monitorEvalMu.Unlock()
		return
	}
	delete(s.cloudExpiryState, fp)
	s.monitorEvalMu.Unlock()
	item := NewCanonicalAlert(
		IngressSourceCloudExpiry,
		"cloud-expiry",
		"resolved",
		map[string]string{"alertname": labels["alertname"]},
		labels,
		IngressAlertDetail{
			Status:      "resolved",
			Labels:       labels,
			Annotations:  annotations,
			StartsAt:     now.Add(-time.Minute),
			EndsAt:       now,
			Fingerprint:  fp,
		},
	)
	item.Cloud = cloud
	_ = s.receiveCanonicalSync(ctx, item)
}

func parseRegionSet(scope string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, it := range strings.Split(scope, ",") {
		v := strings.TrimSpace(it)
		if v != "" {
			out[v] = struct{}{}
		}
	}
	return out
}

// 无 Redis 时按进程内时间戳记录各规则上次「按 Cron 触发评估」时间。
func (s *AlertService) cloudExpiryLocalLastEval(syntheticID uint) (time.Time, bool) {
	s.cloudExpiryEvalMu.Lock()
	defer s.cloudExpiryEvalMu.Unlock()
	if s.cloudExpiryNoRedisLastEval == nil {
		return time.Time{}, false
	}
	last, ok := s.cloudExpiryNoRedisLastEval[syntheticID]
	if !ok || last.IsZero() {
		return time.Time{}, false
	}
	return last, true
}

func (s *AlertService) touchCloudExpiryNoRedisLastEval(syntheticID uint, now time.Time) {
	s.cloudExpiryEvalMu.Lock()
	defer s.cloudExpiryEvalMu.Unlock()
	if s.cloudExpiryNoRedisLastEval == nil {
		s.cloudExpiryNoRedisLastEval = make(map[uint]time.Time)
	}
	s.cloudExpiryNoRedisLastEval[syntheticID] = now
}
