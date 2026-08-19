package esmgmt

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
)

func actorID(actor *auth.CurrentUser) uint {
	if actor == nil {
		return 0
	}
	return actor.ID
}

func isSuperAdmin(actor *auth.CurrentUser) bool {
	return actor != nil && auth.IsSuperAdminRole(actor.RoleCodes)
}

// assertConnectionManage 连接 CRUD：超管或 Owner。日志平台虚拟连接不可改。
func (s *Service) assertConnectionManage(ctx context.Context, connectionID uint, actor *auth.CurrentUser) error {
	if connectionID == 0 {
		return constants.ErrBadRequestWithMsg("「日志平台 ES」为只读虚拟连接，不可变更")
	}
	if isSuperAdmin(actor) {
		return nil
	}
	var row model.EsmgmtConnection
	if err := s.db.WithContext(ctx).Select("id", "owner_user_id").First(&row, connectionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrNotFound
		}
		return err
	}
	if actorID(actor) == 0 {
		return constants.ErrForbidden
	}
	if row.OwnerUserID == 0 {
		return constants.ErrForbiddenWithMsg("该连接未绑定负责人，仅超级管理员可变更")
	}
	if actor.ID == row.OwnerUserID {
		return nil
	}
	return constants.ErrForbidden
}

// assertConnectionWrite 备份/恢复/写代理：超管、Owner，或日志平台虚拟连接（id=0，由 Casbin 约束）。
func (s *Service) assertConnectionWrite(ctx context.Context, connectionID uint, actor *auth.CurrentUser) error {
	if connectionID == 0 {
		if actorID(actor) == 0 {
			return constants.ErrForbidden
		}
		return nil
	}
	return s.assertConnectionManage(ctx, connectionID, actor)
}

func (s *Service) assertDestructiveRestore(ctx context.Context, req RestoreIndexRequest, actor *auth.CurrentUser) error {
	if !req.DeleteExisting {
		return nil
	}
	if isSuperAdmin(actor) {
		return nil
	}
	target := strings.TrimSpace(req.TargetIndex)
	if target == "" && req.BackupJobID > 0 {
		if backup, err := s.GetBackupJob(ctx, req.BackupJobID); err == nil {
			target = backup.IndexName
		}
	}
	confirm := strings.TrimSpace(req.ConfirmTargetIndex)
	if confirm == "" {
		return constants.ErrBadRequestWithMsg("覆盖恢复须填写 confirm_target_index 与目标索引名一致")
	}
	if confirm != target {
		return constants.ErrBadRequestWithMsg("confirm_target_index 与目标索引不一致")
	}
	connID := req.ConnectionID
	if connID == 0 && req.BackupJobID > 0 {
		if backup, err := s.GetBackupJob(ctx, req.BackupJobID); err == nil {
			connID = backup.ConnectionID
		}
	}
	return s.assertConnectionWrite(ctx, connID, actor)
}
