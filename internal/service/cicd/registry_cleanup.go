package cicd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/cronutil"
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
	q := s.db.WithContext(ctx).Model(&model.ImageCleanupPolicy{})
	if registryID > 0 {
		q = q.Where("registry_id = ?", registryID)
	}
	var rows []model.ImageCleanupPolicy
	if err := q.Order("id ASC").Find(&rows).Error; err != nil {
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
		if err := s.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
			return nil, constants.ErrNotFound
		}
	}
	row.RegistryID = req.RegistryID
	row.HarborProject = strings.TrimSpace(req.HarborProject)
	row.KeepLastN = req.KeepLastN
	row.RetainDays = req.RetainDays
	row.Enabled = enabled
	row.CronSpec = spec
	if id == 0 {
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	} else if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *Service) DeleteCleanupPolicy(ctx context.Context, id uint) error {
	res := s.db.WithContext(ctx).Delete(&model.ImageCleanupPolicy{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
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
	var policies []model.ImageCleanupPolicy
	_ = s.db.WithContext(ctx).Where("enabled = ?", true).Find(&policies).Error
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
		_ = s.db.WithContext(ctx).Model(&model.ImageCleanupPolicy{}).Where("id = ?", p.ID).Updates(map[string]any{
			"last_run_at":  now,
			"last_result":  truncate(msg, 1000),
			"updated_at":   now,
		}).Error
	}
}

func (s *Service) runOneCleanupPolicy(ctx context.Context, p *model.ImageCleanupPolicy) string {
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
			return "list projects: " + err.Error()
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
			continue
		}
		for _, repo := range repos {
			arts, err := s.ListHarborArtifacts(ctx, p.RegistryID, 0, proj, repo.Name)
			if err != nil {
				continue
			}
			type ranked struct {
				art   HarborTagItem
				t     time.Time
				ref   string
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
			keep := map[string]struct{}{}
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
					keep[it.ref] = struct{}{}
					skipped++
					continue
				}
				if err := s.DeleteHarborArtifact(ctx, p.RegistryID, 0, proj, repo.Name, it.ref); err != nil {
					continue
				}
				deleted++
			}
			_ = keep
		}
	}
	return fmt.Sprintf("deleted=%d skipped_or_kept=%d", deleted, skipped)
}

// RunCleanupPolicyNow 手动触发一条策略。
func (s *Service) RunCleanupPolicyNow(ctx context.Context, id uint) (string, error) {
	var p model.ImageCleanupPolicy
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&p).Error; err != nil {
		return "", constants.ErrNotFound
	}
	msg := s.runOneCleanupPolicy(ctx, &p)
	now := time.Now()
	_ = s.db.WithContext(ctx).Model(&p).Updates(map[string]any{
		"last_run_at": now,
		"last_result": truncate(msg, 1000),
	}).Error
	return msg, nil
}
