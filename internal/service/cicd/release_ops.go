package cicd

import (
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

// ReleaseOperationLabel 操作类型展示名。
func ReleaseOperationLabel(op string) string {
	switch strings.TrimSpace(op) {
	case model.CicdReleaseOpFrontendOnline:
		return "服务上线"
	case model.CicdReleaseOpFrontendRollback:
		return "服务回滚"
	case model.CicdReleaseOpBackendInitial:
		return "服务初次部署"
	case model.CicdReleaseOpBackendUpdate:
		return "服务更新"
	case model.CicdReleaseTypeServiceOnline:
		return "服务上线"
	case model.CicdReleaseTypePodUpdate:
		return "POD 更新"
	case model.CicdReleaseOpContainerRollback:
		return "回滚"
	default:
		return op
	}
}

func validateReleaseOperation(serviceType, op string) error {
	op = strings.TrimSpace(op)
	if op == "" {
		return constants.ErrBadRequestWithMsg("请选择发布操作类型")
	}
	switch strings.ToLower(strings.TrimSpace(serviceType)) {
	case model.CicdServiceTypeFrontend:
		if op != model.CicdReleaseOpFrontendOnline && op != model.CicdReleaseOpFrontendRollback {
			return constants.ErrBadRequestWithMsg("前端服务操作类型须为：服务上线、服务回滚")
		}
	default:
		if op != model.CicdReleaseOpBackendInitial && op != model.CicdReleaseOpBackendUpdate {
			return constants.ErrBadRequestWithMsg("后端服务操作类型须为：服务初次部署、服务更新")
		}
	}
	return nil
}

func validateContainerReleaseOperation(op string) error {
	op = strings.TrimSpace(op)
	if op == "" {
		return constants.ErrBadRequestWithMsg("请选择容器发布操作类型")
	}
	switch op {
	case model.CicdReleaseTypeServiceOnline, model.CicdReleaseTypePodUpdate, model.CicdReleaseOpContainerRollback:
		return nil
	default:
		return constants.ErrBadRequestWithMsg("容器发布操作类型须为：服务上线、POD 更新、回滚")
	}
}

func defaultReleaseOperation(serviceType string) string {
	if strings.EqualFold(serviceType, model.CicdServiceTypeFrontend) {
		return model.CicdReleaseOpFrontendOnline
	}
	return model.CicdReleaseOpBackendUpdate
}

// releaseDeployAction 映射为 Jenkins deployAction 参数（供共享库识别）。
func releaseDeployAction(op string) string {
	switch strings.TrimSpace(op) {
	case model.CicdReleaseTypeServiceOnline:
		return "初始化部署"
	case model.CicdReleaseTypePodUpdate:
		return "服务更新"
	case model.CicdReleaseOpContainerRollback:
		return "服务更新"
	default:
		if label := ReleaseOperationLabel(op); label != "" && label != op {
			return label
		}
		return op
	}
}

func releaseForceCleanDeploy(op string) bool {
	return op == model.CicdReleaseOpBackendInitial
}
