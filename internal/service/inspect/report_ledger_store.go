package inspect

import (
	"context"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
)

// PeriodDiff 期次对比结论，用于报告「与上期对比」章节。
type PeriodDiff struct {
	HasBaseline    bool
	BaselineRunID  uint
	BaselineAt     time.Time
	BaselineText   string
	NewCount       int
	PersistCount   int
	RecoveredCount int
	// Recovered 上期存在、本期未再出现的风险条目（用于展示整改成效）。
	Recovered []LedgerEntry
	Summary   string
}

// applyPeriodDiff 用上一期台账回填本期条目的期次状态与整改字段，并给出对比结论。
//
// 规则：
//   - 上期存在且本期仍在 → persisting，沿用 Owner/DueDate/FirstSeen（不覆盖人工填写）
//   - 上期不存在 → new，FirstSeen 记为本期
//   - 上期存在但本期消失 → recovered，仅落库与展示，不进入本期风险清单
//
// 无上一期成功巡检时（首次巡检），全部记为 new 且不输出对比章节。
func (s *Service) applyPeriodDiff(ctx context.Context, projectID, runID uint, now time.Time, entries []LedgerEntry) ([]LedgerEntry, PeriodDiff) {
	diff := PeriodDiff{}
	prev, prevRun := s.loadPreviousFindings(ctx, projectID, runID)

	for i := range entries {
		entries[i].FirstSeenAt = now
		entries[i].State = "new"
		p, ok := prev[entries[i].Fingerprint]
		if !ok {
			entries[i].StateLabel = ledgerStateLabel(entries[i].State)
			diff.NewCount++
			continue
		}
		entries[i].State = "persisting"
		entries[i].StateLabel = ledgerStateLabel(entries[i].State)
		entries[i].Owner = p.Owner
		entries[i].DueDate = p.DueDate
		entries[i].DueDateText = formatDueDate(p.DueDate)
		if !p.FirstSeenAt.IsZero() {
			entries[i].FirstSeenAt = p.FirstSeenAt
			entries[i].DurationDays = int(now.Sub(p.FirstSeenAt).Hours()/24 + 0.5)
		}
		diff.PersistCount++
	}

	if prevRun == nil {
		diff.Summary = "本次为该项目首次生成结构化风险台账，暂无上期数据可比对，本期结论将作为后续对比基线。"
		return entries, diff
	}

	diff.HasBaseline = true
	diff.BaselineRunID = prevRun.ID
	if prevRun.FinishedAt != nil {
		diff.BaselineAt = *prevRun.FinishedAt
	} else {
		diff.BaselineAt = prevRun.CreatedAt
	}
	diff.BaselineText = diff.BaselineAt.Format("2006-01-02 15:04")

	current := make(map[string]bool, len(entries))
	for _, e := range entries {
		current[e.Fingerprint] = true
	}
	for fp, p := range prev {
		if current[fp] {
			continue
		}
		diff.RecoveredCount++
		diff.Recovered = append(diff.Recovered, LedgerEntry{
			Seq:             len(diff.Recovered) + 1,
			Fingerprint:     fp,
			Type:            p.Type,
			Name:            p.Name,
			Severity:        p.Severity,
			Priority:        findingPriorityLabel(p.Severity),
			AffectedService: p.AffectedService,
			State:           "recovered",
			StateLabel:      ledgerStateLabel("recovered"),
			Owner:           p.Owner,
		})
	}
	diff.Summary = buildDiffSummary(diff)
	return entries, diff
}

// loadPreviousFindings 取该项目上一次有台账记录的巡检，返回其非 recovered 条目。
func (s *Service) loadPreviousFindings(ctx context.Context, projectID, runID uint) (map[string]model.InspectFinding, *model.InspectRun) {
	if s.db == nil || projectID == 0 {
		return map[string]model.InspectFinding{}, nil
	}
	var prevRunID uint
	err := s.db.WithContext(ctx).Model(&model.InspectFinding{}).
		Where("project_id = ? AND run_id <> ?", projectID, runID).
		Order("run_id DESC").Limit(1).Pluck("run_id", &prevRunID).Error
	if err != nil || prevRunID == 0 {
		return map[string]model.InspectFinding{}, nil
	}

	var rows []model.InspectFinding
	if err := s.db.WithContext(ctx).
		Where("project_id = ? AND run_id = ? AND state <> ?", projectID, prevRunID, "recovered").
		Find(&rows).Error; err != nil {
		return map[string]model.InspectFinding{}, nil
	}
	out := make(map[string]model.InspectFinding, len(rows))
	for _, r := range rows {
		out[r.Fingerprint] = r
	}

	var prevRun model.InspectRun
	if err := s.db.WithContext(ctx).Where("id = ?", prevRunID).First(&prevRun).Error; err != nil {
		return out, nil
	}
	return out, &prevRun
}

// saveFindings 落库本期台账（含已恢复条目）。失败只告警不阻断报告生成。
func (s *Service) saveFindings(ctx context.Context, projectID, runID uint, now time.Time, entries []LedgerEntry, diff PeriodDiff) error {
	if s.db == nil || projectID == 0 || runID == 0 {
		return nil
	}
	rows := make([]model.InspectFinding, 0, len(entries)+len(diff.Recovered))
	for _, e := range entries {
		firstRun := runID
		if e.State == "persisting" && diff.BaselineRunID != 0 {
			firstRun = diff.BaselineRunID
		}
		firstSeen := e.FirstSeenAt
		if firstSeen.IsZero() {
			firstSeen = now
		}
		rows = append(rows, model.InspectFinding{
			ProjectID:       projectID,
			RunID:           runID,
			Fingerprint:     e.Fingerprint,
			Seq:             e.Seq,
			Type:            e.Type,
			Name:            e.Name,
			Severity:        e.Severity,
			Count:           e.Count,
			AffectedService: e.AffectedService,
			Instances:       e.Instances,
			Phenomenon:      e.Phenomenon,
			Impact:          e.Impact,
			Suggestion:      e.Suggestion,
			State:           e.State,
			Owner:           e.Owner,
			DueDate:         e.DueDate,
			FirstSeenRunID:  firstRun,
			FirstSeenAt:     firstSeen,
		})
	}
	for _, e := range diff.Recovered {
		rows = append(rows, model.InspectFinding{
			ProjectID:       projectID,
			RunID:           runID,
			Fingerprint:     e.Fingerprint,
			Type:            e.Type,
			Name:            e.Name,
			Severity:        e.Severity,
			AffectedService: e.AffectedService,
			State:           "recovered",
			Owner:           e.Owner,
			FirstSeenRunID:  diff.BaselineRunID,
			FirstSeenAt:     now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).CreateInBatches(rows, 100).Error
}

func buildDiffSummary(d PeriodDiff) string {
	var b strings.Builder
	b.WriteString("对比上期巡检（")
	b.WriteString(d.BaselineText)
	b.WriteString("）：")
	switch {
	case d.NewCount == 0 && d.PersistCount == 0 && d.RecoveredCount > 0:
		b.WriteString("上期风险已全部消除，本期未发现新增风险，整改成效达标。")
	case d.NewCount == 0 && d.PersistCount == 0:
		b.WriteString("两期均未发现风险项，运行状态保持稳定。")
	default:
		b.WriteString("新增 ")
		b.WriteString(strconv.Itoa(d.NewCount))
		b.WriteString(" 项、持续 ")
		b.WriteString(strconv.Itoa(d.PersistCount))
		b.WriteString(" 项、已恢复 ")
		b.WriteString(strconv.Itoa(d.RecoveredCount))
		b.WriteString(" 项。")
		if d.PersistCount > 0 {
			b.WriteString("持续项说明上期整改尚未闭环，建议优先复核责任人与整改期限。")
		} else {
			b.WriteString("上期风险均已闭环，需重点关注本期新增项。")
		}
	}
	return b.String()
}

func formatDueDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
