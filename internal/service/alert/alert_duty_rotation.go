package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/lifecycle"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

const (
	dutyRotationHorizon          = 14 * 24 * time.Hour
	dutyRotationSyncInterval     = 6 * time.Hour
	dutyRotationMinPeriodMinutes = 60
	dutyRotationMaxPeriodMinutes = 30 * 24 * 60
)

type AlertDutyRotationListQuery struct {
	MonitorRuleID *uint `form:"monitor_rule_id"`
	ProjectID     *uint `form:"project_id"`
	Enabled       *bool `form:"enabled"`
	Page          int   `form:"page"`
	PageSize      int   `form:"page_size"`
}

type AlertDutyRotationUpsertRequest struct {
	MonitorRuleID     uint      `json:"monitor_rule_id" binding:"required"`
	Name              string    `json:"name" binding:"required,max=128"`
	ParticipantsJSON  string    `json:"participants_json"`
	DepartmentIDsJSON string    `json:"department_ids_json"`
	ExtraEmailsJSON   string    `json:"extra_emails_json"`
	AnchorAt          time.Time `json:"anchor_at" binding:"required"`
	PeriodMinutes     int       `json:"period_minutes" binding:"required"`
	Enabled           *bool     `json:"enabled"`
	Remark            string    `json:"remark" binding:"omitempty,max=512"`
}

type AlertDutyOnCallQuery struct {
	MonitorRuleID uint   `form:"monitor_rule_id" binding:"required"`
	At            string `form:"at"`
}

type AlertDutyRotationSlotInfo struct {
	RotationID    uint      `json:"rotation_id"`
	Name          string    `json:"name"`
	PeriodMinutes int       `json:"period_minutes"`
	StartsAt      time.Time `json:"starts_at"`
	EndsAt        time.Time `json:"ends_at"`
	Participants  []uint    `json:"participants"`
}

type AlertDutyOnCallUser struct {
	ID         uint   `json:"id"`
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Source     string `json:"source"` // manual | rotation
	RotationID uint   `json:"rotation_id,omitempty"`
}

type AlertDutyOnCallResult struct {
	At          time.Time                  `json:"at"`
	Users       []AlertDutyOnCallUser      `json:"users"`
	ExtraEmails []string                   `json:"extra_emails"`
	Blocks      []model.AlertDutyBlock     `json:"blocks"`
	Rotations   []AlertDutyRotationSlotInfo `json:"rotations"`
}

func (s *AlertDutyService) ListRotations(ctx context.Context, q AlertDutyRotationListQuery) ([]model.AlertDutyRotation, int64, int, int, error) {
	page, pageSize := pagination.Normalize(q.Page, q.PageSize)
	list, total, err := s.rotationRepo.List(ctx, repository.AlertDutyRotationListFilter{
		MonitorRuleID: q.MonitorRuleID,
		ProjectID:     q.ProjectID,
		Enabled:       q.Enabled,
	}, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, page, pageSize, bizerrors.Pass(ctx, "alert.duty", "ListRotations", err)
	}
	return list, total, page, pageSize, nil
}

func (s *AlertDutyService) CreateRotation(ctx context.Context, req AlertDutyRotationUpsertRequest) (*model.AlertDutyRotation, error) {
	rule, err := s.loadRotationRule(ctx, req.MonitorRuleID, "CreateRotation")
	if err != nil {
		return nil, err
	}
	if err := assertAlertProjectWrite(ctx, s.memberRepo, rule.ProjectID); err != nil {
		return nil, err
	}
	row, err := normalizeDutyRotation(req)
	if err != nil {
		return nil, err
	}
	if row.Enabled {
		if err := s.ensureSingleEnabledRotation(ctx, row.MonitorRuleID, 0); err != nil {
			return nil, err
		}
	}
	if err := s.rotationRepo.Create(ctx, &row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "CreateRotation", err)
	}
	if err := s.materializeRotation(ctx, &row, time.Now()); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *AlertDutyService) UpdateRotation(ctx context.Context, id uint, req AlertDutyRotationUpsertRequest) (*model.AlertDutyRotation, error) {
	row, err := s.rotationRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg("轮换配置不存在")
		}
		return nil, bizerrors.Pass(ctx, "alert.duty", "UpdateRotation", err)
	}
	rule, err := s.loadRotationRule(ctx, row.MonitorRuleID, "UpdateRotation")
	if err != nil {
		return nil, err
	}
	if err := assertAlertProjectWrite(ctx, s.memberRepo, rule.ProjectID); err != nil {
		return nil, err
	}
	if req.MonitorRuleID > 0 && req.MonitorRuleID != row.MonitorRuleID {
		nextRule, err := s.loadRotationRule(ctx, req.MonitorRuleID, "UpdateRotation")
		if err != nil {
			return nil, err
		}
		if err := assertAlertProjectWrite(ctx, s.memberRepo, nextRule.ProjectID); err != nil {
			return nil, err
		}
		row.MonitorRuleID = req.MonitorRuleID
	}
	next, err := normalizeDutyRotation(req)
	if err != nil {
		return nil, err
	}
	if next.Enabled {
		if err := s.ensureSingleEnabledRotation(ctx, row.MonitorRuleID, row.ID); err != nil {
			return nil, err
		}
	}
	row.Name = next.Name
	row.ParticipantsJSON = next.ParticipantsJSON
	row.DepartmentIDsJSON = next.DepartmentIDsJSON
	row.ExtraEmailsJSON = next.ExtraEmailsJSON
	row.AnchorAt = next.AnchorAt
	row.PeriodMinutes = next.PeriodMinutes
	row.Enabled = next.Enabled
	row.Remark = next.Remark
	if err := s.rotationRepo.Save(ctx, row); err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "UpdateRotation", err)
	}
	if err := s.materializeRotation(ctx, row, time.Now()); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *AlertDutyService) DeleteRotation(ctx context.Context, id uint) error {
	row, err := s.rotationRepo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return constants.ErrNotFoundWithMsg("轮换配置不存在")
		}
		return bizerrors.Pass(ctx, "alert.duty", "DeleteRotation", err)
	}
	rule, err := s.loadRotationRule(ctx, row.MonitorRuleID, "DeleteRotation")
	if err != nil {
		return err
	}
	if err := assertAlertProjectWrite(ctx, s.memberRepo, rule.ProjectID); err != nil {
		return err
	}
	if _, err := s.repo.DeleteFutureByRotation(ctx, row.ID, time.Now()); err != nil {
		return bizerrors.Pass(ctx, "alert.duty", "DeleteRotation", err)
	}
	n, err := s.rotationRepo.Delete(ctx, id)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.duty", "DeleteRotation", err)
	}
	if n == 0 {
		return constants.ErrNotFoundWithMsg("轮换配置不存在")
	}
	return nil
}

// OnCallAtRule 解析某条规则在指定时刻的值班人：手工班次 + 启用的轮换班次（轮换实时计算，不依赖物化是否及时）。
func (s *AlertDutyService) OnCallAtRule(ctx context.Context, monitorRuleID uint, at time.Time) (*AlertDutyOnCallResult, error) {
	if at.IsZero() {
		at = time.Now()
	}
	rule, err := s.ruleRepo.GetByID(ctx, monitorRuleID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrNotFoundWithMsg(constants.ErrMsgdfcd891c9a94)
		}
		return nil, bizerrors.Pass(ctx, "alert.duty", "OnCallAtRule", err)
	}
	if err := assertAlertProjectVisible(ctx, s.memberRepo, rule.ProjectID); err != nil {
		return nil, err
	}
	blocks, err := s.repo.ListActiveAtRule(ctx, monitorRuleID, at)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "OnCallAtRule", err)
	}
	rotations, err := s.rotationRepo.ListEnabledByRule(ctx, monitorRuleID)
	if err != nil {
		return nil, bizerrors.Pass(ctx, "alert.duty", "OnCallAtRule", err)
	}

	uidSet := map[uint]struct{}{}
	sourceByUID := map[uint]string{}
	rotationByUID := map[uint]uint{}
	var extraEmails []string
	for _, b := range blocks {
		if b.RotationID > 0 {
			continue
		}
		for _, id := range parseUintSliceJSON(b.UserIDsJSON) {
			uidSet[id] = struct{}{}
			sourceByUID[id] = "manual"
		}
		more, err := s.userRepo.ListActiveIDsByDepartmentSubtree(ctx, parseUintSliceJSON(b.DepartmentIDsJSON))
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.duty", "OnCallAtRule", err)
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
			sourceByUID[id] = "manual"
		}
		extras, _ := assigneeParseStringSliceJSON(b.ExtraEmailsJSON)
		extraEmails = append(extraEmails, extras...)
	}

	slots := make([]AlertDutyRotationSlotInfo, 0, len(rotations))
	for i := range rotations {
		rot := rotations[i]
		slot, ok := CurrentRotationSlot(rot.AnchorAt, time.Duration(rot.PeriodMinutes)*time.Minute, parseUintSliceJSON(rot.ParticipantsJSON), at)
		if !ok {
			continue
		}
		slots = append(slots, AlertDutyRotationSlotInfo{
			RotationID:    rot.ID,
			Name:          rot.Name,
			PeriodMinutes: rot.PeriodMinutes,
			StartsAt:      slot.StartsAt,
			EndsAt:        slot.EndsAt,
			Participants:  slot.UserIDs,
		})
		for _, id := range slot.UserIDs {
			uidSet[id] = struct{}{}
			if _, exists := sourceByUID[id]; !exists {
				sourceByUID[id] = "rotation"
				rotationByUID[id] = rot.ID
			}
		}
		more, err := s.userRepo.ListActiveIDsByDepartmentSubtree(ctx, parseUintSliceJSON(rot.DepartmentIDsJSON))
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.duty", "OnCallAtRule", err)
		}
		for _, id := range more {
			uidSet[id] = struct{}{}
			if _, exists := sourceByUID[id]; !exists {
				sourceByUID[id] = "rotation"
				rotationByUID[id] = rot.ID
			}
		}
		extras, _ := assigneeParseStringSliceJSON(rot.ExtraEmailsJSON)
		extraEmails = append(extraEmails, extras...)
	}

	ids := make([]uint, 0, len(uidSet))
	for id := range uidSet {
		ids = append(ids, id)
	}
	users := make([]AlertDutyOnCallUser, 0, len(ids))
	if len(ids) > 0 {
		rows, err := s.userRepo.ListByIDs(ctx, ids)
		if err != nil {
			return nil, bizerrors.Pass(ctx, "alert.duty", "OnCallAtRule", err)
		}
		for i := range rows {
			u := rows[i]
			if u.Status != 1 {
				continue
			}
			email := ""
			if u.Email != nil {
				email = *u.Email
			}
			users = append(users, AlertDutyOnCallUser{
				ID:         u.ID,
				Username:   u.Username,
				Nickname:   u.Nickname,
				Email:      email,
				Phone:      u.Phone,
				Source:     sourceByUID[u.ID],
				RotationID: rotationByUID[u.ID],
			})
		}
	}
	sort.Slice(users, func(i, j int) bool {
		a, b := users[i], users[j]
		if a.Nickname != b.Nickname {
			return a.Nickname < b.Nickname
		}
		return a.Username < b.Username
	})
	if blocks == nil {
		blocks = []model.AlertDutyBlock{}
	}
	return &AlertDutyOnCallResult{
		At:          at,
		Users:       users,
		ExtraEmails: dedupeEmailsLower(extraEmails),
		Blocks:      blocks,
		Rotations:   slots,
	}, nil
}

// SyncRotations 为全部启用轮换补齐未来窗口内的物化班次；后台任务与保存后都会调用。
func (s *AlertDutyService) SyncRotations(ctx context.Context) {
	rows, err := s.rotationRepo.ListEnabled(ctx)
	if err != nil {
		alertLog().Warn("duty rotation sync list failed", "error", err)
		return
	}
	now := time.Now()
	for i := range rows {
		if err := s.materializeRotation(ctx, &rows[i], now); err != nil {
			alertLog().Warn("duty rotation materialize failed", "rotation_id", rows[i].ID, "error", err)
		}
	}
}

// RunRotationSyncWorker 周期性物化轮换班次，保证 ListActiveAtRule / 值班日历始终有数据。
func (s *AlertDutyService) RunRotationSyncWorker(ctx context.Context) {
	if s == nil || s.rotationRepo == nil {
		return
	}
	s.SyncRotations(ctx)
	ticker := time.NewTicker(dutyRotationSyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SyncRotations(ctx)
		}
	}
}

func (s *AlertDutyService) startRotationSync(ctx context.Context) {
	lifecycle.Go("alert.duty-rotation-sync", func() { s.RunRotationSyncWorker(ctx) })
}

func (s *AlertDutyService) loadRotationRule(ctx context.Context, ruleID uint, op string) (*model.AlertMonitorRule, error) {
	if ruleID == 0 {
		return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgdfcd891c9a94)
	}
	rule, err := s.ruleRepo.GetByID(ctx, ruleID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsgdfcd891c9a94)
		}
		return nil, bizerrors.Pass(ctx, "alert.duty", op, err)
	}
	return rule, nil
}

func (s *AlertDutyService) ensureSingleEnabledRotation(ctx context.Context, ruleID, exceptID uint) error {
	n, err := s.rotationRepo.CountEnabledByRuleExcept(ctx, ruleID, exceptID)
	if err != nil {
		return bizerrors.Pass(ctx, "alert.duty", "ensureSingleEnabledRotation", err)
	}
	if n > 0 {
		return constants.ErrBadRequestWithMsg("该规则已有启用的轮换配置，请先停用后再启用新的轮换")
	}
	return nil
}

// materializeRotation 重建未来窗口内的班次：保留历史班次，仅清理 ends_at > now 的旧物化行。
func (s *AlertDutyService) materializeRotation(ctx context.Context, rot *model.AlertDutyRotation, now time.Time) error {
	if rot == nil || rot.ID == 0 {
		return nil
	}
	if _, err := s.repo.DeleteFutureByRotation(ctx, rot.ID, now); err != nil {
		return bizerrors.Pass(ctx, "alert.duty", "materializeRotation", err)
	}
	if !rot.Enabled {
		return nil
	}
	participants := parseUintSliceJSON(rot.ParticipantsJSON)
	if len(participants) == 0 {
		return nil
	}
	slots := ComputeRotationSlots(rot.AnchorAt, time.Duration(rot.PeriodMinutes)*time.Minute, participants, now, now.Add(dutyRotationHorizon))
	for _, slot := range slots {
		rawUsers, _ := json.Marshal(slot.UserIDs)
		row := model.AlertDutyBlock{
			MonitorRuleID:     rot.MonitorRuleID,
			RotationID:        rot.ID,
			StartsAt:          slot.StartsAt,
			EndsAt:            slot.EndsAt,
			Title:             rot.Name,
			UserIDsJSON:       string(rawUsers),
			DepartmentIDsJSON: rot.DepartmentIDsJSON,
			ExtraEmailsJSON:   rot.ExtraEmailsJSON,
			Remark:            rot.Remark,
		}
		if err := s.repo.Create(ctx, &row); err != nil {
			return bizerrors.Pass(ctx, "alert.duty", "materializeRotation", err)
		}
	}
	return nil
}

func normalizeDutyRotation(req AlertDutyRotationUpsertRequest) (model.AlertDutyRotation, error) {
	var row model.AlertDutyRotation
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return row, constants.ErrBadRequestWithMsg("轮换名称必填")
	}
	participants := dedupeUintIDs(parseUintSliceJSON(req.ParticipantsJSON))
	if len(participants) == 0 {
		return row, constants.ErrBadRequestWithMsg("轮换至少需要一名参与人")
	}
	if req.AnchorAt.IsZero() {
		return row, constants.ErrBadRequestWithMsg("轮换起算时间必填")
	}
	if req.PeriodMinutes < dutyRotationMinPeriodMinutes || req.PeriodMinutes > dutyRotationMaxPeriodMinutes {
		return row, constants.ErrBadRequestWithMsg(fmt.Sprintf("每班时长需在 %d ~ %d 分钟之间", dutyRotationMinPeriodMinutes, dutyRotationMaxPeriodMinutes))
	}
	extras, err := assigneeParseStringSliceJSON(req.ExtraEmailsJSON)
	if err != nil {
		return row, constants.ErrBadRequestWithMsg("额外邮箱须为字符串数组 JSON")
	}
	var rawDepts []uint
	if raw := strings.TrimSpace(req.DepartmentIDsJSON); raw != "" && raw != "[]" {
		if err := json.Unmarshal([]byte(raw), &rawDepts); err != nil {
			return row, constants.ErrBadRequestWithMsg("部门 ID 须为数字数组 JSON")
		}
	}
	depts := dedupeUintIDs(rawDepts)
	rawParticipants, _ := json.Marshal(participants)
	rawDeptJSON, _ := json.Marshal(depts)
	rawExtras, _ := json.Marshal(dedupeEmailsLower(extras))
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	row = model.AlertDutyRotation{
		MonitorRuleID:     req.MonitorRuleID,
		Name:              name,
		ParticipantsJSON:  string(rawParticipants),
		DepartmentIDsJSON: string(rawDeptJSON),
		ExtraEmailsJSON:   string(rawExtras),
		AnchorAt:          req.AnchorAt,
		PeriodMinutes:     req.PeriodMinutes,
		Enabled:           enabled,
		Remark:            strings.TrimSpace(req.Remark),
	}
	return row, nil
}

func dedupeUintIDs(ids []uint) []uint {
	seen := map[uint]struct{}{}
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
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
