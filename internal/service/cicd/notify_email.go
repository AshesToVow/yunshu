package cicd

import (
	"context"
	"strings"

	"yunshu/internal/model"
)

// resolveNotifyEmail 解析 Jenkins emailUser：请求显式指定 > 应用 Owner > 当前操作者。
func (s *Service) resolveNotifyEmail(ctx context.Context, explicit string, svc *model.CicdService, operatorUserID *uint) string {
	if e := strings.TrimSpace(explicit); e != "" {
		return e
	}
	if svc != nil {
		if e := s.lookupUserEmail(ctx, strings.TrimSpace(svc.Owner)); e != "" {
			return e
		}
	}
	if operatorUserID != nil && *operatorUserID > 0 {
		if e := s.lookupUserEmailByID(ctx, *operatorUserID); e != "" {
			return e
		}
	}
	return ""
}

func (s *Service) lookupUserEmail(ctx context.Context, username string) string {
	if s.userRepo == nil || username == "" {
		return ""
	}
	row, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return ""
	}
	return activeUserEmail(row)
}

func (s *Service) lookupUserEmailByID(ctx context.Context, id uint) string {
	if s.userRepo == nil || id == 0 {
		return ""
	}
	row, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return ""
	}
	return activeUserEmail(row)
}

func activeUserEmail(u *model.User) string {
	if u == nil || u.Status != model.StatusEnabled || u.Email == nil {
		return ""
	}
	return strings.TrimSpace(*u.Email)
}
