package changeevent

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"

	"gorm.io/gorm"
)

var (
	mu     sync.RWMutex
	bound  *gorm.DB
	logger = slog.Default().With("component", "changeevent")
)

// BindDB 在应用启动时绑定全局 DB，供各域写路径 best-effort 埋点。
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

// Input 变更埋点入参。
type Input struct {
	ProjectID   uint
	ServiceID   *uint
	Source      string
	Action      string
	RiskLevel   string
	Status      string
	ActorUserID *uint
	Summary     string
	Payload     any
	StartedAt   *time.Time
	FinishedAt  *time.Time
	RollbackRef string
}

// Record 写入 change_events；失败仅打日志，不阻断主流程。
func Record(ctx context.Context, in Input) {
	d := db()
	if d == nil || in.ProjectID == 0 {
		return
	}
	source := strings.TrimSpace(in.Source)
	action := strings.TrimSpace(in.Action)
	summary := strings.TrimSpace(in.Summary)
	if source == "" || action == "" || summary == "" {
		return
	}
	risk := strings.TrimSpace(in.RiskLevel)
	if risk == "" {
		risk = model.ChangeRiskMedium
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = model.ChangeStatusSucceeded
	}
	actor := in.ActorUserID
	if actor == nil {
		if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil {
			id := u.ID
			actor = &id
		}
	}
	started := time.Now()
	if in.StartedAt != nil {
		started = *in.StartedAt
	}
	finished := in.FinishedAt
	if finished == nil && (status == model.ChangeStatusSucceeded ||
		status == model.ChangeStatusFailed ||
		status == model.ChangeStatusAborted) {
		now := time.Now()
		finished = &now
	}
	payload := ""
	if in.Payload != nil {
		if b, err := json.Marshal(in.Payload); err == nil {
			payload = string(b)
		}
	}
	row := model.ChangeEvent{
		ProjectID:   in.ProjectID,
		ServiceID:   in.ServiceID,
		Source:      source,
		Action:      action,
		RiskLevel:   risk,
		Status:      status,
		ActorUserID: actor,
		Summary:     summary,
		PayloadJSON: payload,
		StartedAt:   started,
		FinishedAt:  finished,
		RollbackRef: strings.TrimSpace(in.RollbackRef),
	}
	if err := d.WithContext(ctx).Create(&row).Error; err != nil {
		logger.Warn("record change event failed", "error", err, "source", source, "action", action)
	}
}
