package cicd

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/cronutil"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

type CleanupPolicyUpsertRequest struct {
	RegistryID    uint   `json:"registry_id" binding:"required"`
	HarborProject string `json:"harbor_project" binding:"omitempty,max=128"`
	KeepLastN     int    `json:"keep_last_n"`
	RetainDays    int    `json:"retain_days"`
	Enabled       *bool  `json:"enabled"`
	CronSpec      string `json:"cron_spec" binding:"omitempty,max=64"`
}

func (s *Service) ListCleanupPolicies(ctx context.Context, registryID uint) ([]model.ImageCleanupPolicy, error) {
	p := repository.CicdCleanupPolicyListParams{}
	if registryID > 0 {
		p.RegistryID = &registryID
	}
	rows, _, err := s.repo.ListCleanupPolicies(ctx, p)
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.ImageCleanupPolicy{}
	}
	return rows, nil
}

func (s *Service) UpsertCleanupPolicy(ctx context.Context, id uint, req CleanupPolicyUpsertRequest) (*model.ImageCleanupPolicy, error) {
	if _, err := s.getRegistry(ctx, req.RegistryID); err != nil {
		return nil, err
	}
	if req.KeepLastN < 0 {
		req.KeepLastN = 0
	}
	if req.RetainDays < 0 {
		req.RetainDays = 0
	}
	spec := strings.TrimSpace(req.CronSpec)
	if spec == "" {
		spec = "0 3 * * *"
	}
	if err := cronutil.ValidateSpec(spec, "cron_spec"); err != nil {
		return nil, constants.ErrBadRequestWithMsg(err.Error())
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var row model.ImageCleanupPolicy
	if id > 0 {
		existing, err := s.repo.GetCleanupPolicy(ctx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, constants.ErrNotFound
			}
			return nil, err
		}
		row = *existing
	}
	row.RegistryID = req.RegistryID
	row.HarborProject = strings.TrimSpace(req.HarborProject)
	row.KeepLastN = req.KeepLastN
	row.RetainDays = req.RetainDays
	row.Enabled = enabled
	row.CronSpec = spec
	if id == 0 {
		if err := s.repo.CreateCleanupPolicy(ctx, &row); err != nil {
			return nil, err
		}
	} else if err := s.repo.SaveCleanupPolicy(ctx, &row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) DeleteCleanupPolicy(ctx context.Context, id uint) error {
	n, err := s.repo.DeleteCleanupPolicy(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return constants.ErrNotFound
	}
	return nil
}

// RunImageCleanupWorker 按各策略 Cron 清理镜像 Tag。
func (s *Service) RunImageCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	lastByPolicy := map[uint]time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tickImageCleanup(ctx, lastByPolicy)
		}
	}
}

func (s *Service) tickImageCleanup(ctx context.Context, lastByPolicy map[uint]time.Time) {
	policies, _ := s.repo.ListEnabledCleanupPolicies(ctx)
	now := time.Now()
	for _, p := range policies {
		spec := strings.TrimSpace(p.CronSpec)
		if spec == "" {
			spec = "0 3 * * *"
		}
		last := lastByPolicy[p.ID]
		hasLast := !last.IsZero()
		if p.LastRunAt != nil && p.LastRunAt.After(last) {
			last = *p.LastRunAt
			hasLast = true
		}
		if !cronutil.ShouldRunAfterLast(spec, last, hasLast, now) {
			continue
		}
		msg := s.runOneCleanupPolicy(ctx, &p)
		lastByPolicy[p.ID] = now
		_ = s.repo.UpdateCleanupPolicyFields(ctx, p.ID, map[string]any{
			"last_run_at": now,
			"last_result": truncate(msg, 1000),
			"updated_at":  now,
		})
	}
}

// cleanupMaxDeletePerRun 单次策略执行允许删除的最大 Tag 数（熔断）。
// 触及上限即停止并在 last_result 中标注，避免规则配置失误造成大批量误删。
const cleanupMaxDeletePerRun = 200

func (s *Service) runOneCleanupPolicy(ctx context.Context, p *model.ImageCleanupPolicy) string {
	return s.executeCleanupPolicy(ctx, p, false)
}

// executeCleanupPolicy 执行一条清理策略。dryRun=true 时只统计不删除。
// 任一层级的列举失败都会中止整条策略：镜像列表不完整会让 keep_last_n 的排序判定失真，
// 继续执行等于按残缺数据删除本应保留的镜像。
func (s *Service) executeCleanupPolicy(ctx context.Context, p *model.ImageCleanupPolicy, dryRun bool) string {
	reg, err := s.getRegistry(ctx, p.RegistryID)
	if err != nil {
		return "registry not found"
	}
	resolved := registryToResolved(reg)
	projects := []string{}
	if v := strings.TrimSpace(p.HarborProject); v != "" {
		projects = []string{v}
	} else if resolved.Type == model.ImageRegistryTypeHarbor {
		items, err := s.ListHarborProjects(ctx, p.RegistryID, 0)
		if err != nil {
			return "aborted: list projects: " + err.Error()
		}
		for _, it := range items {
			projects = append(projects, it.Name)
		}
	} else {
		projects = []string{""}
	}

	deleted := 0
	skipped := 0
	for _, proj := range projects {
		repos, err := s.ListHarborRepositories(ctx, p.RegistryID, 0, proj)
		if err != nil {
			return fmt.Sprintf("aborted: list repositories(%s): %s (deleted=%d skipped_or_kept=%d)", proj, err.Error(), deleted, skipped)
		}
		for _, repo := range repos {
			arts, err := s.ListHarborArtifacts(ctx, p.RegistryID, 0, proj, repo.Name)
			if err != nil {
				return fmt.Sprintf("aborted: list artifacts(%s/%s): %s (deleted=%d skipped_or_kept=%d)", proj, repo.Name, err.Error(), deleted, skipped)
			}
			type ranked struct {
				art HarborTagItem
				t   time.Time
				ref string
			}
			rankedList := make([]ranked, 0, len(arts))
			for _, a := range arts {
				ref := a.Digest
				if len(a.Tags) > 0 {
					ref = a.Tags[0]
				}
				if ref == "" {
					continue
				}
				ts, _ := time.Parse(time.RFC3339Nano, a.PushTime)
				if ts.IsZero() {
					ts, _ = time.Parse(time.RFC3339, a.PushTime)
				}
				rankedList = append(rankedList, ranked{art: a, t: ts, ref: ref})
			}
			sort.Slice(rankedList, func(i, j int) bool {
				return rankedList[i].t.After(rankedList[j].t)
			})
			for i, it := range rankedList {
				protect := false
				if len(it.art.Linked) > 0 {
					protect = true
				}
				if p.KeepLastN > 0 && i < p.KeepLastN {
					protect = true
				}
				if p.RetainDays > 0 && !it.t.IsZero() && time.Since(it.t) < time.Duration(p.RetainDays)*24*time.Hour {
					protect = true
				}
				if protect {
					skipped++
					continue
				}
				if deleted >= cleanupMaxDeletePerRun {
					return fmt.Sprintf("aborted: 单次删除上限 %d 已触发，请核对策略后再执行 (deleted=%d skipped_or_kept=%d)", cleanupMaxDeletePerRun, deleted, skipped)
				}
				if dryRun {
					deleted++
					continue
				}
				if err := s.DeleteHarborArtifact(ctx, p.RegistryID, 0, proj, repo.Name, it.ref); err != nil {
					continue
				}
				deleted++
			}
		}
	}
	if dryRun {
		return fmt.Sprintf("dry_run=true would_delete=%d skipped_or_kept=%d", deleted, skipped)
	}
	return fmt.Sprintf("deleted=%d skipped_or_kept=%d", deleted, skipped)
}

// RunCleanupPolicyNow 手动触发一条策略。dryRun=true 时只预演不删除，且不刷新 last_run_at。
// 注意：本接口的访问控制目前完全依赖路由层 Casbin（/api/v1/registries 组），service 层未再做平台管理员断言。
func (s *Service) RunCleanupPolicyNow(ctx context.Context, id uint, dryRun bool) (string, error) {
	p, err := s.repo.GetCleanupPolicy(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", constants.ErrNotFound
		}
		return "", err
	}
	msg := s.executeCleanupPolicy(ctx, p, dryRun)
	if dryRun {
		return msg, nil
	}
	now := time.Now()
	_ = s.repo.UpdateCleanupPolicyFields(ctx, id, map[string]any{
		"last_run_at": now,
		"last_result": truncate(msg, 1000),
	})
	return msg, nil
}
