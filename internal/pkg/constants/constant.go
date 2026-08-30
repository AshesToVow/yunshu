// Package constants：统一业务错误码（数字 error_code）、产品话术、以及脚本生成的 ErrMsg* 文案与模板。
// 业务码分段：10xxx 通用；11xxx 请求校验；20xxx–26xxx 按功能域手写；11020/10901 等为可变文案固定码（Err*WithMsg，可传 ErrMsg* / fmt 拼接）。
// 请使用 BizError、域内 Err*；长尾或脚本 ErrMsg* 用 Err*WithMsg(constants.ErrMsg…)。response.Error(c, err) 将业务码写入 JSON error_code。
// 「固定错误/提示文案」「fmt.Sprintf 模板」两个脚本生成区已拆至同包 messages_generated.go，勿手改其常量值。
// 注意：手写 BizError（10xxx–26xxx）必须留在本文件，biz_reason_test.go 按文件名读取本文件校验覆盖率。
package constants

import (
	"fmt"
	"net/http"
	"strings"

	bizerrors "yunshu/internal/pkg/errors"
)

// BizError 构造业务错误：HTTP 状态、数字业务码、reason（OneX）、产品话术（message）。
// JSON 响应含 code/reason/message/error_code（兼容）/metadata，见 ErrorHandler 中间件。
func BizError(httpStatus, bizCode int, message string) error {
	return bizerrors.NewBiz(httpStatus, bizCode, ReasonForBizCode(bizCode), message)
}

// ErrBadRequestWithMsg 固定业务码 11020，文案由调用方传入（绑定失败、fmt 拼接等）。
func ErrBadRequestWithMsg(msg string) error {
	return BizError(http.StatusBadRequest, 11020, msg)
}

// ErrNotFoundWithMsg 固定业务码 11021。
func ErrNotFoundWithMsg(msg string) error {
	return BizError(http.StatusNotFound, 11021, msg)
}

// ErrForbiddenWithMsg 固定业务码 11022。
func ErrForbiddenWithMsg(msg string) error {
	return BizError(http.StatusForbidden, 11022, msg)
}

// ErrUnauthorizedWithMsg 固定业务码 11023。
func ErrUnauthorizedWithMsg(msg string) error {
	return BizError(http.StatusUnauthorized, 11023, msg)
}

// ErrConflictWithMsg 固定业务码 11024。
func ErrConflictWithMsg(msg string) error {
	return BizError(http.StatusConflict, 11024, msg)
}

// ErrInternalWithMsg 固定业务码 10901（与 ErrInternal 固定话术区分）。
func ErrInternalWithMsg(msg string) error {
	return BizError(http.StatusInternalServerError, 10901, msg)
}

// ErrTooManyRequestsWithMsg 固定业务码 10902。
func ErrTooManyRequestsWithMsg(msg string) error {
	return BizError(http.StatusTooManyRequests, 10902, msg)
}

// —— 通用 10xxx ——
var (
	ErrBadRequest               = BizError(http.StatusBadRequest, 10001, "请求参数无效，请检查后重试")
	ErrUnauthorized             = BizError(http.StatusUnauthorized, 10002, "登录已失效或凭证无效，请重新登录")
	ErrForbidden                = BizError(http.StatusForbidden, 10003, "当前账号无权执行该操作")
	ErrNotFound                 = BizError(http.StatusNotFound, 10004, "所请求的资源不存在或已删除")
	ErrConflict                 = BizError(http.StatusConflict, 10005, "资源状态冲突，请刷新后重试")
	ErrInternal                 = BizError(http.StatusInternalServerError, 10006, "平台服务异常，请稍后重试或联系管理员")
	ErrTooManyRequests          = BizError(http.StatusTooManyRequests, 10007, "操作过于频繁，请稍后再试")
	ErrMissingAuthHeader        = BizError(http.StatusUnauthorized, 10008, "缺少登录凭证，请先登录")
	ErrAccessTokenInvalid       = BizError(http.StatusUnauthorized, 10009, "访问令牌无效，请重新登录")
	ErrLoginSessionExpired      = BizError(http.StatusUnauthorized, 10010, "登录会话已过期，请重新登录")
	ErrAccountPrincipalNotFound = BizError(http.StatusUnauthorized, 10011, "账号不存在或已删除，请检查登录信息")
	ErrAccountDisabled          = BizError(http.StatusForbidden, 10012, "账号已被禁用，如需协助请联系管理员")
	ErrWSMissingTicketParam     = BizError(http.StatusUnauthorized, 10013, "缺少 WebSocket 握手票据，请先调用 POST /api/v1/auth/ws-ticket 获取 ticket 并在连接 URL 中携带")
	ErrWSTicketInvalid          = BizError(http.StatusUnauthorized, 10015, "WebSocket 握手票据无效或已过期，请重新获取 ticket 后再连接")
	ErrNotLoggedIn              = BizError(http.StatusUnauthorized, 10014, "未完成登录，请先登录后再访问该资源")
)

// —— 请求校验 11xxx ——
var (
	ErrIncludeRegexInvalid = BizError(http.StatusBadRequest, 11001, "「包含」筛选条件格式不正确，请检查正则表达式")
	ErrExcludeRegexInvalid = BizError(http.StatusBadRequest, 11002, "「排除」筛选条件格式不正确，请检查正则表达式")
	ErrInvalidRequestParam = BizError(http.StatusBadRequest, 11003, "请求参数不合法，请检查后重试")
)

// —— 认证与账号 20xxx ——
var (
	ErrUserNotFound            = BizError(http.StatusNotFound, 20001, "用户信息不存在或已停用")
	ErrPasswordIncorrect       = BizError(http.StatusUnauthorized, 20002, "账号或密码不正确，请检查后重试")
	ErrEmailAlreadyRegistered  = BizError(http.StatusConflict, 20003, "该邮箱已被占用，请更换后重试")
	ErrTokenGenerateFailed     = BizError(http.StatusInternalServerError, 20004, "登录凭证签发失败，请稍后重试")
	ErrEmailNotBound           = BizError(http.StatusBadRequest, 20005, "当前账号未绑定邮箱，请先在安全设置中完成绑定")
	ErrNicknameRequired        = BizError(http.StatusBadRequest, 20006, "昵称不能为空，请填写后保存")
	ErrCaptchaIPRateLimited    = BizError(http.StatusTooManyRequests, 20007, "验证码请求过于频繁，请稍后再试")
	ErrCaptchaExpired          = BizError(http.StatusBadRequest, 20008, "验证码已失效，请重新获取")
	ErrCaptchaIncorrect        = BizError(http.StatusBadRequest, 20009, "验证码不正确，请检查后重试")
	ErrCaptchaRequired         = BizError(http.StatusBadRequest, 20010, "请先填写验证码")
	ErrCaptchaInvalidOrExpired = BizError(http.StatusUnauthorized, 20011, "验证码无效或已过期，请重新获取")
	ErrUsernameTaken           = BizError(http.StatusConflict, 20012, "用户名已被占用，请更换后重试")
	ErrCaptchaCoolingDown      = BizError(http.StatusConflict, 20013, "图形验证码刷新过于频繁，请稍后再试")
	ErrLoginTooManyAttempts    = BizError(http.StatusTooManyRequests, 20014, "密码错误次数过多，账号已被临时锁定，请稍后再试")
)

// —— 日志采集 Agent 21xxx ——
var (
	ErrAgentTokenInvalid          = BizError(http.StatusUnauthorized, 21001, "Agent 访问凭证无效或已轮换，请重新下发 Token")
	ErrAgentRegisterClosed        = BizError(http.StatusForbidden, 21002, "公共 Agent 自助注册已关闭，请联系管理员")
	ErrServerDisabledForAgent     = BizError(http.StatusForbidden, 21003, "目标服务器已停用，无法完成 Agent 相关操作")
	ErrAgentRegisterSecretInvalid = BizError(http.StatusUnauthorized, 21004, "Agent 注册密钥无效，请核对平台侧配置")
	ErrAgentTokenMissing          = BizError(http.StatusUnauthorized, 21005, "缺少 Agent 访问令牌，请携带 Token 后重试")
)

// —— 告警 22xxx ——
var (
	ErrAlertSilenceNotFound     = BizError(http.StatusNotFound, 22001, "告警静默规则不存在或已失效")
	ErrAlertWebhookTokenInvalid = BizError(http.StatusForbidden, 22002, "告警回调令牌无效，请核对告警集成配置")
)

// —— 项目与服务器 23xxx ——
var (
	ErrProjectNotFound               = BizError(http.StatusNotFound, 23001, "项目不存在或已被删除")
	ErrServerNotFound                = BizError(http.StatusNotFound, 23002, "服务器不存在或已移除")
	ErrServerNotInCurrentProject     = BizError(http.StatusBadRequest, 23003, "该服务器不属于当前项目，请切换项目后再操作")
	ErrServerProjectMismatch         = BizError(http.StatusBadRequest, 23004, "项目与服务器归属不一致，请刷新后重试")
	ErrProjectIDRequired             = BizError(http.StatusBadRequest, 23005, "缺少项目标识（project_id），请补充后重试")
	ErrNameRequired                  = BizError(http.StatusBadRequest, 23006, "名称不能为空，请填写后提交")
	ErrUploadFailed                  = BizError(http.StatusBadRequest, 23007, "文件上传失败，请检查网络或文件大小后重试")
	ErrServerNotInProject            = BizError(http.StatusBadRequest, 23008, "该服务器不在当前项目中，请刷新或切换范围后重试")
	ErrServerNotInProjectForbidden   = BizError(http.StatusForbidden, 23009, "该服务器不在当前项目中，当前账号无权访问")
	ErrProjectMemberRequired         = BizError(http.StatusForbidden, 23010, "您不是该项目的成员，无权访问或操作该项目资源")
	ErrProjectAdminRequired          = BizError(http.StatusForbidden, 23011, "该操作需要项目管理员或负责人权限")
	ErrProjectReadonlyMember         = BizError(http.StatusForbidden, 23012, "项目只读成员不能执行此修改类操作")
	ErrK8sClusterProjectAccessDenied = BizError(http.StatusForbidden, 23013, "该集群已绑定到其他业务项目，当前账号不在允许范围内")
	ErrProjectArchived               = BizError(http.StatusForbidden, 23014, "项目已归档，仅允许只读访问")
)

// ErrLogSourceServerNotFound 历史别名，与 ErrServerNotFound 同码（23002）；新代码请使用 ErrServerNotFound。
var ErrLogSourceServerNotFound = ErrServerNotFound

// —— RBAC / 组织 24xxx ——
var (
	ErrRoleNotFound         = BizError(http.StatusNotFound, 24001, "角色不存在或已删除")
	ErrUserGroupNotFound    = BizError(http.StatusNotFound, 24002, "用户组不存在或已删除")
	ErrPermissionNotFound   = BizError(http.StatusNotFound, 24005, "权限项不存在或已变更")
	ErrMenuNotFound         = BizError(http.StatusNotFound, 24003, "菜单不存在或已下线")
	ErrDepartmentNotFound   = BizError(http.StatusNotFound, 24004, "部门不存在或已撤销")
	ErrPolicyAlreadyGranted = BizError(http.StatusConflict, 24006, "该角色已拥有此权限，请勿重复授权")
	ErrPolicyNotGranted     = BizError(http.StatusNotFound, 24007, "该角色未拥有此权限，无需回收")
)

// —— 自助注册 25xxx ——
var (
	ErrRegistrationRequestNotFound  = BizError(http.StatusNotFound, 25001, "注册申请不存在或已处理")
	ErrRegistrationAlreadyProcessed = BizError(http.StatusConflict, 25002, "该注册申请已审核，请勿重复操作")
	ErrRegistrationDuplicatePending = BizError(http.StatusConflict, 25003, "该用户名或邮箱已有待审核申请，请勿重复提交")
)

// —— Kubernetes 集群 26xxx ——
var (
	// ErrK8sNamespaceAlreadyExists 表单创建命名空间时名称已在集群中存在（HTTP 409 / error_code 26001）。
	ErrK8sNamespaceAlreadyExists = BizError(http.StatusConflict, 26001, "该命名空间已存在，请勿重复创建")
	// ErrK8sClusterAPIUnauthorized 集群 Token/证书无效（HTTP 403，避免与平台登录 401 混淆导致前端误登出）。
	ErrK8sClusterAPIUnauthorized = BizError(http.StatusForbidden, 26002, ErrMsgK8sAPIUnauthorized)
)

// ErrK8sNamespaceAlreadyExistsMsg 返回业务码 26001，文案包含冲突的名称。
func ErrK8sNamespaceAlreadyExistsMsg(name string) error {
	n := strings.TrimSpace(name)
	if n == "" {
		return ErrK8sNamespaceAlreadyExists
	}
	return BizError(http.StatusConflict, 26001, fmt.Sprintf("命名空间「%s」已存在，请勿重复创建", n))
}

// 展示用语（非 error）：Agent 离线归因话术
const (
	LogAgentOfflineNeverConnected = "从未连接成功"
	LogAgentOfflineHeartbeatLost  = "心跳超时（失联或进程僵死）"
	LogAgentOfflineAgentStopped   = "Agent 已停止"
	LogAgentOfflineAgentError     = "Agent 异常（进程上报 error）"
)

// 文案模板与拼接前缀
const (
	ErrFmtJSONFieldMustBeObject   = "%s 须为 JSON 对象字符串，请检查配置或请求体"
	ErrFmtAlertSilenceBatchEndsAt = "批量静默项「%s」结束时间须晚于开始时间"
	ErrMsgSSHConnectFailedPrefix  = "远程连接失败（SSH）："
	ErrMsgSSHExecFailedPrefix     = "远程命令执行失败："
	ErrMsgCloudSDKPrefix          = "云平台返回异常："
)
