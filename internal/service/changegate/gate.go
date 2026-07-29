package changegate

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

var (
	mu    sync.RWMutex
	bound *gorm.DB
)

func BindDB(db *gorm.DB) {
	mu.Lock()
	defer mu.Unlock()
	bound = db
}

func db() *gorm.DB {
	mu.RLock()
	defer mu.RUnlock()
	return bound
}

// CheckInput 写路径门禁入参。
type CheckInput struct {
	ProjectID uint
	Source    string // cicd|k8s|dbmgmt
	Env       string // prod/staging/...；空则仅匹配 env 为空的冻结窗
	ServiceID *uint  // catalog service id
	Namespace string // k8s ns，用于冲突
	Action    string
	BlockOnConflict bool // true=冲突时拒绝；false=仅返回警告
	ConflictWindow  time.Duration
}

// CheckResult 门禁结果。
type CheckResult struct {
	Allowed         bool                `json:"allowed"`
	BlockedByFreeze bool                `json:"blocked_by_freeze"`
	FreezeWindowID  uint                `json:"freeze_window_id,omitempty"`
	FreezeName      string              `json:"freeze_name,omitempty"`
	ConflictWarning bool                `json:"conflict_warning"`
	ConflictEvents  []model.ChangeEvent `json:"conflict_events,omitempty"`
	Message         string              `json:"message,omitempty"`
}

// Check 校验冻结窗与近期冲突；冻结命中时返回业务错误风格结果（调用方应拒绝）。
func Check(ctx context.Context, in CheckInput) CheckResult {
	d := db()
	out := CheckResult{Allowed: true}
	if d == nil || in.ProjectID == 0 {
		return out
	}
	now := time.Now()
	var freezes []model.ChangeFreezeWindow
	q := d.WithContext(ctx).Model(&model.ChangeFreezeWindow{}).
		Where("project_id = ? AND enabled = ? AND starts_at <= ? AND ends_at >= ?", in.ProjectID, true, now, now)
	_ = q.Find(&freezes).Error
	src := strings.ToLower(strings.TrimSpace(in.Source))
	env := strings.ToLower(strings.TrimSpace(in.Env))
	for _, f := range freezes {
		if !scopeMatch(f.Scope, src) {
			continue
		}
		fEnv := strings.ToLower(strings.TrimSpace(f.Env))
		if fEnv != "" && env != "" && fEnv != env {
			continue
		}
		if fEnv != "" && env == "" {
			// 调用方未声明环境时，仅「全环境」冻结窗生效；带 env 的冻结窗不拦截
			continue
		}
		out.Allowed = false
		out.BlockedByFreeze = true
		out.FreezeWindowID = f.ID
		out.FreezeName = f.Name
		out.Message = fmt.Sprintf("变更冻结窗口「%s」生效中（%s ~ %s）", f.Name, f.StartsAt.Format(time.RFC3339), f.EndsAt.Format(time.RFC3339))
		return out
	}

	win := in.ConflictWindow
	if win <= 0 {
		win = 15 * time.Minute
	}
	since := now.Add(-win)
	cq := d.WithContext(ctx).Model(&model.ChangeEvent{}).
		Where("project_id = ? AND started_at >= ? AND source = ?", in.ProjectID, since, src)
	if in.ServiceID != nil && *in.ServiceID > 0 {
		cq = cq.Where("service_id = ?", *in.ServiceID)
	}
	if ns := strings.TrimSpace(in.Namespace); ns != "" {
		like := `%"namespace":"` + ns + `"%`
		cq = cq.Where("payload_json LIKE ?", like)
	}
	var recent []model.ChangeEvent
	_ = cq.Order("id DESC").Limit(10).Find(&recent).Error
	if len(recent) > 0 {
		out.ConflictWarning = true
		out.ConflictEvents = recent
		out.Message = fmt.Sprintf("近 %s 内已有 %d 条同类变更", win.String(), len(recent))
		if in.BlockOnConflict {
			out.Allowed = false
			out.Message = "变更冲突：" + out.Message
		}
	}
	return out
}

func scopeMatch(scope, source string) bool {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" || scope == model.FreezeScopeAll {
		return true
	}
	return scope == source
}

// ErrBlocked 将门禁结果转为可返回给 API 的错误。
func ErrBlocked(res CheckResult) error {
	if res.Allowed {
		return nil
	}
	msg := strings.TrimSpace(res.Message)
	if msg == "" {
		msg = "变更被治理策略拦截"
	}
	return constants.ErrBadRequestWithMsg(msg)
}
