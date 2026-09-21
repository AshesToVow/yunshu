package kafkamgmt

import (
	"context"
	"errors"
	"strings"

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

// assertConnectionManage 连接 CRUD / 写操作：超管或 Owner。
func (s *Service) assertConnectionManage(ctx context.Context, connectionID uint, actor *auth.CurrentUser) error {
	if connectionID == 0 {
		return constants.ErrBadRequestWithMsg("请指定具体连接")
	}
	if isSuperAdmin(actor) {
		return nil
	}
	row, err := s.repo.GetConnectionOwner(ctx, connectionID)
	if err != nil {
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

// assertConnectionWrite 写 Topic/生产/删组：超管或连接 Owner；connectionID=0 时校验默认连接。
func (s *Service) assertConnectionWrite(ctx context.Context, connectionID uint, actor *auth.CurrentUser) error {
	if connectionID == 0 {
		if actorID(actor) == 0 {
			return constants.ErrForbidden
		}
		def, err := s.repo.GetDefaultConnection(ctx)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return constants.ErrForbiddenWithMsg("未配置默认 Kafka 连接，请指定 connection_id")
			}
			return err
		}
		return s.assertConnectionManage(ctx, def.ID, actor)
	}
	return s.assertConnectionManage(ctx, connectionID, actor)
}

func normalizeSASL(mech, username string) string {
	mech = strings.ToLower(strings.TrimSpace(mech))
	switch mech {
	case "plain", "scram-sha-256", "scram-sha-512", "none", "":
	default:
		mech = "plain"
	}
	if mech == "" && strings.TrimSpace(username) != "" {
		mech = "plain"
	}
	if mech == "none" {
		return ""
	}
	return mech
}
