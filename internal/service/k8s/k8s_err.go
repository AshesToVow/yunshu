package k8s

import (
	"context"
	"errors"
	"strings"

	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/constants"

	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// k8sFail 统一处理 K8s Service 层错误：保留 AppError；映射 apiserver 状态码；其余走 bizerrors.Pass。
func k8sFail(ctx context.Context, component, operation string, err error, attrs ...any) error {
	if err == nil {
		return nil
	}
	if _, ok := bizerrors.As(err); ok {
		return err
	}
	if mapped := k8sMapAPIError(err); mapped != err {
		if _, ok := bizerrors.As(mapped); ok {
			return mapped
		}
	}
	return bizerrors.Pass(ctx, component, operation, err, attrs...)
}

// k8sRepoErr 将仓储层错误转为业务错误（未找到 → 10004）。
func k8sRepoErr(ctx context.Context, component, operation string, err error, attrs ...any) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return constants.ErrNotFound
	}
	return bizerrors.Pass(ctx, component, operation, err, attrs...)
}

func isK8sUnauthorizedErr(err error) bool {
	if err == nil {
		return false
	}
	if biz, ok := bizerrors.As(err); ok && biz.ErrorCode == "26002" {
		return true
	}
	if apierrors.IsUnauthorized(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unauthorized")
}

func k8sMapAPIError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case apierrors.IsNotFound(err):
		return constants.ErrNotFound
	case apierrors.IsForbidden(err):
		// 区分平台 Casbin 与集群 RBAC：保留 K8s 原文，避免一律落成「当前账号无权执行该操作」。
		msg := strings.TrimSpace(err.Error())
		if msg == "" {
			return constants.ErrForbidden
		}
		if len(msg) > 280 {
			msg = msg[:280] + "…"
		}
		return constants.ErrForbiddenWithMsg("Kubernetes 集群拒绝访问（非平台超管权限）：" + msg)
	case apierrors.IsConflict(err):
		return constants.ErrConflict
	case apierrors.IsUnauthorized(err):
		return constants.ErrK8sClusterAPIUnauthorized
	case apierrors.IsInvalid(err), apierrors.IsBadRequest(err):
		msg := string(apierrors.ReasonForError(err))
		if msg == "" {
			msg = err.Error()
		}
		return constants.ErrBadRequestWithMsg(msg)
	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "unauthorized") || strings.Contains(msg, "401 unauthorized") {
			return constants.ErrK8sClusterAPIUnauthorized
		}
		return err
	}
}

// k8sFailOrInternal 优先映射 apiserver 业务错误；否则返回带上下文的 10901。
func k8sFailOrInternal(ctx context.Context, component, operation string, err error, msgFmt string, attrs ...any) error {
	if err == nil {
		return nil
	}
	if mapped := k8sMapAPIError(err); mapped != err {
		if _, ok := bizerrors.As(mapped); ok {
			return mapped
		}
	}
	return bizerrors.Internalf(ctx, component, operation, err, msgFmt, attrs...)
}
