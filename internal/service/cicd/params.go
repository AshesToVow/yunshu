package cicd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
)

// BuildParamsInput 触发 Jenkins 构建所需的上下文。
type BuildParamsInput struct {
	Service          *model.CicdService
	CiConfig         *model.CicdCiConfig
	DeployConfig     *model.CicdDeployConfig
	Cfg              config.CicdConfig
	BranchName       string
	PublishMode      string
	Tenv             string
	DestIPs          string
	EmailUser        string
	ImageAddress     string
	SelectedVersion  string
	ReleaseOperation string
	ForceCleanDeploy bool
	UsesK8sPipeline  bool
	// Harbor 项目级覆盖（空字段表示该键仍用全局配置）。
	HarborURL     string
	HarborProject string
	// Apollo 项目级覆盖（空字段表示不传该键，沿用 Jenkins Job 默认；供 launch/K8s 模板占位符替换）。
	ApolloMeta       string
	ApolloEnv        string
	ApolloNamespaces string
	// YunshuBuildRunID 注入 Jenkins，便于 CI 回调回写阶段/门禁。
	YunshuBuildRunID uint
	// YunshuReleaseRunID 注入 Jenkins，便于 CD 回调定位发布工单。
	YunshuReleaseRunID uint
	// EnableSonarOverride 非 nil 时覆盖字典开关（CD 发布通常传 false）。
	EnableSonarOverride *bool
}

// BuildJenkinsParams 将服务配置映射为 Jenkins buildWithParameters 参数（与 jenkinsfile 文档一致）。
func BuildJenkinsParams(in BuildParamsInput) map[string]string {
	params := map[string]string{
		"Tenv":        strings.TrimSpace(in.Tenv),
		"publishMode": strings.TrimSpace(in.PublishMode),
		"emailUser":   strings.TrimSpace(in.EmailUser),
		"waitMins":    dictconfig.FormatWaitMins(in.Cfg),
	}
	if params["Tenv"] == "" {
		params["Tenv"] = "dev"
	}
	if params["publishMode"] == "" {
		params["publishMode"] = "仅构建"
	}
	applyCredentialParams(params, in.Cfg, in.Service, in)

	svc := in.Service
	ci := in.CiConfig
	if svc == nil {
		return params
	}
	if ci != nil {
		params["SrcURL"] = strings.TrimSpace(ci.GitURL)
		branch := strings.TrimSpace(in.BranchName)
		if branch == "" {
			branch = strings.TrimSpace(ci.RefName)
		}
		params["branchName"] = branch
		params["buildType"] = strings.TrimSpace(ci.BuildType)
		params["buildshell"] = strings.TrimSpace(ci.BuildShell)
		if ci.Version != "" {
			params["imageTag"] = ci.Version
		}
	}

	switch strings.ToLower(strings.TrimSpace(svc.ServiceType)) {
	case model.CicdServiceTypeFrontend:
		if in.UsesK8sPipeline {
			buildK8sCiParams(params, ci, in)
		} else {
			buildFrontendParams(params, ci, in)
		}
	default:
		if in.UsesK8sPipeline {
			buildK8sCiParams(params, ci, in)
		} else {
			buildBackendParams(params, ci, in)
		}
	}

	if dc := in.DeployConfig; dc != nil {
		applyDeployParams(params, dc, in)
	}
	if v := strings.TrimSpace(in.SelectedVersion); v != "" {
		params["selectedVersion"] = v
	}
	if v := strings.TrimSpace(in.ReleaseOperation); v != "" {
		params["deployAction"] = releaseDeployAction(v)
	}
	if v := strings.TrimSpace(in.ImageAddress); v != "" {
		params["FULL_IMAGE_NAME"] = v
	}
	applySonarAndCallbackParams(params, in)
	// Apollo 需在 deploy 参数（含 Tenv）确定后再注入，SSH 与容器模板共用。
	applyApolloParams(params, in)
	return params
}

func applyCredentialParams(params map[string]string, cfg config.CicdConfig, svc *model.CicdService, in BuildParamsInput) {
	gitCred := strings.TrimSpace(cfg.Credentials.Git)
	if gitCred == "" {
		gitCred = "gitee_registry_ssh"
	}
	sshCred := strings.TrimSpace(cfg.Credentials.SSHDeploy)
	if sshCred == "" {
		sshCred = "target-server-credential"
	}
	params["GIT_CREDENTIAL_ID"] = gitCred
	params["SSH_KEY_CREDENTIAL_ID"] = sshCred

	minioCred := strings.TrimSpace(cfg.Credentials.MinIO)
	if minioCred == "" {
		minioCred = "minio-credentials"
	}
	params["MINIO_CREDENTIAL_ID"] = minioCred

	minioBucket := strings.TrimSpace(cfg.MinIO.BucketBackend)
	if svc != nil && strings.EqualFold(svc.ServiceType, model.CicdServiceTypeFrontend) {
		minioBucket = strings.TrimSpace(cfg.MinIO.BucketFrontend)
	}
	if minioBucket == "" {
		minioBucket = "backend-artifacts"
	}
	params["MINIO_BUCKET"] = minioBucket

	minioEndpoint := strings.TrimSpace(cfg.MinIO.Endpoint)
	if minioEndpoint == "" {
		minioEndpoint = "http://192.168.56.102:8021"
	}
	params["MINIO_ENDPOINT"] = minioEndpoint
	applyHarborParams(params, cfg, in.HarborURL, in.HarborProject)
}

func applyHarborParams(params map[string]string, cfg config.CicdConfig, projectURL, projectGroup string) {
	harborURL := strings.TrimSpace(cfg.Harbor.URL)
	if v := strings.TrimSpace(projectURL); v != "" {
		harborURL = stripHarborHost(v)
	}
	if harborURL == "" {
		harborURL = "harbor.deploy.local"
	}
	harborCred := strings.TrimSpace(cfg.Credentials.Harbor)
	if harborCred == "" {
		harborCred = "HARBOR_ID"
	}
	harborProject := strings.TrimSpace(cfg.Harbor.ProjectGroup)
	if v := strings.TrimSpace(projectGroup); v != "" {
		harborProject = v
	}
	if harborProject == "" {
		harborProject = "registry"
	}
	harborHostIP := strings.TrimSpace(cfg.Harbor.HostIP)
	if harborHostIP == "" {
		harborHostIP = "10.10.10.103"
	}
	params["HARBOR_URL"] = harborURL
	params["HARBOR_HOST_IP"] = harborHostIP
	params["HARBOR_CREDENTIAL_ID"] = harborCred
	params["PROJECT_GROUP"] = harborProject
}

func stripHarborHost(raw string) string {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(v, "https://")
	v = strings.TrimPrefix(v, "http://")
	return strings.TrimRight(v, "/")
}

// applyApolloParams 将项目 Apollo 配置写入 Jenkins 参数，供 shared-lib launch/K8s 模板替换
// {{APOLLO_META}} / {{APOLLO_ENV}} / {{APOLLO_NAMESPACES}}。
// APOLLO_META 支持逗号分隔多个 Meta 地址。
func applyApolloParams(params map[string]string, in BuildParamsInput) {
	meta := normalizeApolloMetaList(in.ApolloMeta)
	env := strings.TrimSpace(in.ApolloEnv)
	ns := strings.TrimSpace(in.ApolloNamespaces)
	if meta == "" && env == "" && ns == "" {
		return
	}
	if env == "" {
		env = apolloEnvFromTenv(params["Tenv"])
	}
	if meta != "" {
		params["APOLLO_META"] = meta
	}
	if env != "" {
		params["APOLLO_ENV"] = env
	}
	if ns != "" {
		params["APOLLO_NAMESPACES"] = ns
	}
}

// normalizeApolloMetaList 规范化逗号分隔的多个 Meta 地址（去空白、去空段）。
func normalizeApolloMetaList(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, ",")
}

func apolloEnvFromTenv(tenv string) string {
	switch strings.ToLower(strings.TrimSpace(tenv)) {
	case "prod", "production", "pro":
		return "PRO"
	case "uat":
		return "UAT"
	case "test", "fat", "qa":
		return "FAT"
	case "dev", "development", "local":
		return "DEV"
	default:
		if v := strings.TrimSpace(tenv); v != "" {
			return strings.ToUpper(v)
		}
		return "PRO"
	}
}

func buildK8sCiParams(params map[string]string, ci *model.CicdCiConfig, in BuildParamsInput) {
	if params["buildType"] == "" {
		params["buildType"] = "mvn"
	}
	if params["buildshell"] == "" {
		params["buildshell"] = "clean package -DskipTests"
	}
	imageName := ""
	if ci != nil {
		imageName = strings.TrimSpace(ci.ProjectName)
	}
	if imageName == "" && in.Service != nil {
		imageName = strings.TrimSpace(in.Service.Identifier)
	}
	imageName = strings.ToLower(imageName)
	params["projectName"] = imageName
	params["imageName"] = imageName
	if ci != nil && strings.TrimSpace(ci.Version) != "" {
		params["imageTag"] = strings.TrimSpace(ci.Version)
	}
	retain := in.Cfg.DefaultArtifactRetain
	if retain <= 0 {
		retain = 10
	}
	params["artifactRetainCount"] = strconv.Itoa(retain)
}

func buildFrontendParams(params map[string]string, ci *model.CicdCiConfig, in BuildParamsInput) {
	if ci == nil {
		return
	}
	if params["buildType"] == "" {
		params["buildType"] = "npm"
	}
	if params["buildshell"] == "" {
		params["buildshell"] = "run build"
	}
	if ci.BuildPath != "" {
		params["destPath"] = strings.TrimSpace(ci.BuildPath)
	}
	params["npmInstallMode"] = strings.TrimSpace(ci.NpmInstallMode)
	if params["npmInstallMode"] == "" {
		params["npmInstallMode"] = "install"
	}
	params["cleanNpmCache"] = boolStr(ci.CleanNpmCache)
	params["cleanNodeModules"] = boolStr(ci.CleanNodeModules)
	params["nodeToolName"] = model.NodeToolNameFromConfig(ci)
	retain := in.Cfg.DefaultArtifactRetain
	if retain <= 0 {
		retain = 10
	}
	params["artifactRetainCount"] = strconv.Itoa(retain)
}

func buildBackendParams(params map[string]string, ci *model.CicdCiConfig, in BuildParamsInput) {
	if ci == nil {
		return
	}
	if params["buildType"] == "" {
		params["buildType"] = "mvn"
	}
	if params["buildshell"] == "" {
		params["buildshell"] = "clean package -DskipTests"
	}
	projectName := strings.TrimSpace(ci.ProjectName)
	if projectName == "" && in.Service != nil {
		projectName = strings.TrimSpace(in.Service.Identifier)
	}
	params["projectName"] = projectName
	if ci.BuildPath != "" {
		params["buildPath"] = strings.TrimSpace(ci.BuildPath)
	} else {
		params["buildPath"] = "target"
	}
	if ci.JavaToolName != "" {
		params["javaToolName"] = ci.JavaToolName
	}
	if ci.ServerPort != "" {
		params["serverPort"] = ci.ServerPort
	}
	if ci.PackConfigPaths != "" {
		params["packConfigPaths"] = ci.PackConfigPaths
	}
	retain := in.Cfg.DefaultArtifactRetain
	if retain <= 0 {
		retain = 10
	}
	params["artifactRetainCount"] = strconv.Itoa(retain)
}

func applyDeployParams(params map[string]string, dc *model.CicdDeployConfig, in BuildParamsInput) {
	if dc.Tenv != "" {
		params["Tenv"] = dc.Tenv
	}
	if strings.EqualFold(dc.DeployKind, model.CicdDeployKindContainer) {
		params["deployMethod"] = dc.DeployMethod
		params["deployAction"] = dc.DeployAction
		params["deployConfigType"] = dc.DeployConfigType
		params["deployConfigTemplate"] = dc.DeployConfigTemplate
		params["k8s_ns"] = dc.K8sNamespace
		params["replicas"] = strconv.Itoa(maxInt(dc.Replicas, 1))
		params["ContainerPort"] = strconv.Itoa(maxInt(dc.ContainerPort, 8080))
		strategy := normalizeDeployStrategy(dc.DeployStrategy)
		params["deployStrategy"] = strategy
		params["canaryReplicas"] = strconv.Itoa(maxInt(dc.CanaryReplicas, 1))
		params["canaryPercent"] = strconv.Itoa(maxInt(dc.CanaryPercent, 10))
		steps := strings.TrimSpace(dc.CanaryStepsJSON)
		if steps == "" {
			steps = "10,50,100"
		}
		params["canarySteps"] = steps
		if v := strings.TrimSpace(dc.BlueGreenService); v != "" {
			params["blueGreenService"] = v
		}
		if dc.ImageName != "" {
			params["imageName"] = dc.ImageName
		}
		if dc.ImageTag != "" {
			params["imageTag"] = dc.ImageTag
		}
		if in.ImageAddress != "" {
			params["FULL_IMAGE_NAME"] = in.ImageAddress
		}
		return
	}
	if dc.DestPath != "" {
		params["destPath"] = dc.DestPath
	}
	if in.DestIPs != "" {
		params["destIp"] = in.DestIPs
	}
	if dc.DeployUser != "" {
		params["deployUser"] = dc.DeployUser
	}
	if dc.DeployGroup != "" {
		params["deployGroup"] = dc.DeployGroup
	}
	runUser := strings.TrimSpace(dc.RunUser)
	if runUser == "" {
		runUser = "app"
	}
	params["runUser"] = runUser
	if dc.StartScriptType != "" {
		params["startScriptType"] = dc.StartScriptType
	} else {
		params["startScriptType"] = "脚本模板"
	}
	if dc.CustomScriptContent != "" {
		params["customScriptContent"] = dc.CustomScriptContent
	}
	params["cleanDeployDir"] = boolStr(dc.CleanDeployDir)
	if in.ForceCleanDeploy {
		params["cleanDeployDir"] = "true"
	}
	if dc.JVMOpts != "" {
		params["JVM_OPTS"] = dc.JVMOpts
	}
	if dc.ServerPort > 0 {
		params["serverPort"] = strconv.Itoa(dc.ServerPort)
	}
	if dc.ArtifactRetainCount > 0 {
		params["artifactRetainCount"] = strconv.Itoa(dc.ArtifactRetainCount)
	}
}

func boolStr(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func applySonarAndCallbackParams(params map[string]string, in BuildParamsInput) {
	enable := in.Cfg.Sonar.Enabled
	if in.EnableSonarOverride != nil {
		enable = *in.EnableSonarOverride
	}
	// CD「制品发布」默认不跑 Sonar，避免重复扫描。
	if strings.TrimSpace(in.PublishMode) == model.CicdPublishModeArtifactDeploy {
		enable = false
	}
	if enable {
		params["enableSonar"] = "true"
	} else {
		params["enableSonar"] = "false"
	}
	if v := strings.TrimSpace(in.Cfg.Sonar.URL); v != "" {
		params["SONAR_HOST_URL"] = v
	}
	if v := strings.TrimSpace(in.Cfg.Sonar.Token); v != "" {
		params["SONAR_TOKEN"] = v
	}
	if v := strings.TrimSpace(in.Cfg.Callback.CallbackURL); v != "" {
		params["YUNSHU_CALLBACK_URL"] = v
	}
	if v := strings.TrimSpace(in.Cfg.Callback.HMACSecret); v != "" {
		params["YUNSHU_CALLBACK_HMAC_SECRET"] = v
	}
	if in.YunshuReleaseRunID > 0 {
		// Job 参数名沿用 YUNSHU_BUILD_RUN_ID；靠 YUNSHU_RUN_KIND=release 区分工单表。
		params["YUNSHU_BUILD_RUN_ID"] = strconv.FormatUint(uint64(in.YunshuReleaseRunID), 10)
		params["YUNSHU_RUN_KIND"] = model.CicdRunKindRelease
	} else if in.YunshuBuildRunID > 0 {
		params["YUNSHU_BUILD_RUN_ID"] = strconv.FormatUint(uint64(in.YunshuBuildRunID), 10)
		params["YUNSHU_RUN_KIND"] = model.CicdRunKindBuild
	}
}

// ParseServerIDs 解析 deploy config 中的服务器 ID 列表。
func ParseServerIDs(raw string) ([]uint, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("invalid server_ids_json: %w", err)
	}
	return ids, nil
}

// maskedParamValue 敏感参数落库时的占位值。
const maskedParamValue = "***"

// sensitiveParamKeys 精确匹配的敏感参数名（Jenkins Job 参数）。
var sensitiveParamKeys = map[string]struct{}{
	"SONAR_TOKEN":                 {},
	"YUNSHU_CALLBACK_HMAC_SECRET": {},
	"harborPassword":              {},
	"HARBOR_PASSWORD":             {},
	"gitPassword":                 {},
	"GIT_PASSWORD":                {},
}

// sensitiveParamSubstrings 子串匹配（大写后比较），覆盖后续新增的凭据类参数。
var sensitiveParamSubstrings = []string{
	"PASSWORD", "SECRET", "TOKEN", "PRIVATE_KEY", "PRIVATEKEY", "CREDENTIAL", "PASSPHRASE", "APIKEY", "API_KEY",
}

// IsSensitiveParamKey 判断 Jenkins 参数名是否属于凭据类，需在落库/回显前脱敏。
func IsSensitiveParamKey(key string) bool {
	k := strings.TrimSpace(key)
	if k == "" {
		return false
	}
	if _, ok := sensitiveParamKeys[k]; ok {
		return true
	}
	upper := strings.ToUpper(k)
	for _, frag := range sensitiveParamSubstrings {
		if strings.Contains(upper, frag) {
			return true
		}
	}
	return false
}

// ParamsJSON 序列化 Jenkins 参数用于落库。
// 凭据类参数（SONAR_TOKEN / YUNSHU_CALLBACK_HMAC_SECRET / *PASSWORD* 等）一律替换为 "***"：
// params_json 会随工单详情接口回传前端，明文落库等于把回调签名密钥和扫描令牌长期暴露。
// 注意：脱敏只影响持久化副本，传给 Jenkins 的 params map 不变。
func ParamsJSON(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	safe := make(map[string]string, len(params))
	for k, v := range params {
		if IsSensitiveParamKey(k) && strings.TrimSpace(v) != "" {
			safe[k] = maskedParamValue
			continue
		}
		safe[k] = v
	}
	b, _ := json.Marshal(safe)
	return string(b)
}
