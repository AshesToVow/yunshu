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
)

// Recorder persists change events (best-effort). Implemented by ChangeEventRepository.
type Recorder interface {
	Create(ctx context.Context, row *model.ChangeEvent) error
}

var (
	mu     sync.RWMutex
	bound  Recorder
	logger = slog.Default().With("component", "changeevent")
)

// BindRepo binds a ChangeEvent repository for package-level Record helpers.
func BindRepo(repo Recorder) {
	mu.Lock()
	defer mu.Unlock()
	bound = repo
}

func recorder() Recorder {
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
	rec := recorder()
	if rec == nil || in.ProjectID == 0 {
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
	if err := rec.Create(ctx, &row); err != nil {
		logger.Warn("record change event failed", "error", err, "source", source, "action", action)
	}
}
