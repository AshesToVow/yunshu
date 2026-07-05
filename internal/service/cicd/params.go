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
	applyCredentialParams(params, in.Cfg, in.Service)

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
	return params
}

func applyCredentialParams(params map[string]string, cfg config.CicdConfig, svc *model.CicdService) {
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
	applyHarborParams(params, cfg)
}

func applyHarborParams(params map[string]string, cfg config.CicdConfig) {
	harborURL := strings.TrimSpace(cfg.Harbor.URL)
	if harborURL == "" {
		harborURL = "harbor.deploy.local"
	}
	harborCred := strings.TrimSpace(cfg.Credentials.Harbor)
	if harborCred == "" {
		harborCred = "HARBOR_ID"
	}
	harborProject := strings.TrimSpace(cfg.Harbor.ProjectGroup)
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

func ParamsJSON(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	b, _ := json.Marshal(params)
	return string(b)
}
