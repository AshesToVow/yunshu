package constants

import "strconv"

// bizReasonByCode 将数字业务码映射为 OneX 风格 reason（英文 PascalCase，稳定可枚举）。
// 须与 constant.go 中手写 BizError（10xxx–26xxx）保持一一对应；可变文案码 10901/10902、11020–11024 亦在此登记。
var bizReasonByCode = map[int]string{
	// 通用 10xxx
	10001: "BadRequest",
	10002: "Unauthorized",
	10003: "Forbidden",
	10004: "NotFound",
	10005: "Conflict",
	10006: "InternalError",
	10007: "TooManyRequests",
	10008: "MissingAuthHeader",
	10009: "AccessTokenInvalid",
	10010: "LoginSessionExpired",
	10011: "AccountPrincipalNotFound",
	10012: "AccountDisabled",
	10013: "WSMissingTicketParam",
	10014: "NotLoggedIn",
	10015: "WSTicketInvalid",
	// 可变文案 10xxx / 11xxx
	10901: "InternalErrorWithMessage",
	10902: "TooManyRequestsWithMessage",
	11001: "IncludeRegexInvalid",
	11002: "ExcludeRegexInvalid",
	11003: "InvalidRequestParam",
	11020: "BadRequestWithMessage",
	11021: "NotFoundWithMessage",
	11022: "ForbiddenWithMessage",
	11023: "UnauthorizedWithMessage",
	11024: "ConflictWithMessage",
	// 认证与账号 20xxx
	20001: "UserNotFound",
	20002: "PasswordIncorrect",
	20003: "EmailAlreadyRegistered",
	20004: "TokenGenerateFailed",
	20005: "EmailNotBound",
	20006: "NicknameRequired",
	20007: "CaptchaIPRateLimited",
	20008: "CaptchaExpired",
	20009: "CaptchaIncorrect",
	20010: "CaptchaRequired",
	20011: "CaptchaInvalidOrExpired",
	20012: "UsernameTaken",
	20013: "CaptchaCoolingDown",
	20014: "LoginAccountLocked",
	// 日志采集 Agent 21xxx
	21001: "AgentTokenInvalid",
	21002: "AgentRegisterClosed",
	21003: "ServerDisabledForAgent",
	21004: "AgentRegisterSecretInvalid",
	21005: "AgentTokenMissing",
	// 告警 22xxx
	22001: "AlertSilenceNotFound",
	22002: "AlertWebhookTokenInvalid",
	// 项目与服务器 23xxx
	23001: "ProjectNotFound",
	23002: "ServerNotFound",
	23003: "ServerNotInCurrentProject",
	23004: "ServerProjectMismatch",
	23005: "ProjectIDRequired",
	23006: "NameRequired",
	23007: "UploadFailed",
	23008: "ServerNotInProject",
	23009: "ServerNotInProjectForbidden",
	23010: "ProjectMemberRequired",
	23011: "ProjectAdminRequired",
	23012: "ProjectReadonlyMember",
	23013: "K8sClusterProjectAccessDenied",
	23014: "ProjectArchived",
	// RBAC / 组织 24xxx
	24001: "RoleNotFound",
	24002: "UserGroupNotFound",
	24003: "MenuNotFound",
	24004: "DepartmentNotFound",
	24005: "PermissionNotFound",
	24006: "PolicyAlreadyGranted",
	24007: "PolicyNotGranted",
	// 自助注册 25xxx
	25001: "RegistrationRequestNotFound",
	25002: "RegistrationAlreadyProcessed",
	25003: "RegistrationDuplicatePending",
	// Kubernetes 26xxx
	26001: "K8sNamespaceAlreadyExists",
	26002: "K8sClusterAPIUnauthorized",
}

// ReasonForBizCode 返回业务码对应的 reason，未知码时返回 BizError{code}。
func ReasonForBizCode(bizCode int) string {
	if r, ok := bizReasonByCode[bizCode]; ok {
		return r
	}
	return "BizError" + strconv.Itoa(bizCode)
}
