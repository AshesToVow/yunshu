package jenkins

import (
	"fmt"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/model"
)

// JobTemplateInput 生成 Pipeline Job config.xml 所需上下文。
type JobTemplateInput struct {
	Service     *model.CicdService
	CiConfig    *model.CicdCiConfig
	Cfg         config.CicdConfig
	ScriptPath  string
	Description string
	K8sPipeline bool
	K8sDefaults *model.CicdDeployConfig
}

// BuildPipelineJobConfigXML 生成 Pipeline script from SCM 的 config.xml（与 jenkinsfile 实践文档对齐）。
func BuildPipelineJobConfigXML(in JobTemplateInput) string {
	svc := in.Service
	ci := in.CiConfig
	cfg := in.Cfg

	gitURL := ""
	refName := "main"
	buildType := "mvn"
	buildShell := ""
	projectName := ""
	buildPath := ""
	if ci != nil {
		gitURL = strings.TrimSpace(ci.GitURL)
		refName = strings.TrimSpace(ci.RefName)
		if refName == "" {
			refName = "main"
		}
		buildType = strings.TrimSpace(ci.BuildType)
		buildShell = strings.TrimSpace(ci.BuildShell)
		projectName = strings.TrimSpace(ci.ProjectName)
		buildPath = strings.TrimSpace(ci.BuildPath)
	}
	if svc != nil && projectName == "" {
		projectName = strings.TrimSpace(svc.Identifier)
	}

	serviceType := ""
	if svc != nil {
		serviceType = strings.ToLower(strings.TrimSpace(svc.ServiceType))
	}

	switch serviceType {
	case model.CicdServiceTypeFrontend:
		if buildType == "" {
			buildType = "npm"
		}
		if buildShell == "" {
			buildShell = "run build"
		}
	default:
		if buildType == "" {
			buildType = "mvn"
		}
		if buildShell == "" {
			buildShell = "clean package -DskipTests"
		}
		if buildPath == "" {
			buildPath = "target"
		}
	}

	jenkinsfileRepo := strings.TrimSpace(cfg.Jenkinsfile.Repo)
	if jenkinsfileRepo == "" {
		jenkinsfileRepo = "git@gitee.com:wxd_ops/jenkinsfile-new.git"
	}
	jenkinsfileBranch := strings.TrimSpace(cfg.Jenkinsfile.Branch)
	if jenkinsfileBranch == "" {
		jenkinsfileBranch = "main"
	}
	gitCred := strings.TrimSpace(cfg.Credentials.Git)
	if gitCred == "" {
		gitCred = "gitee_registry_ssh"
	}
	sshCred := strings.TrimSpace(cfg.Credentials.SSHDeploy)
	if sshCred == "" {
		sshCred = "target-server-credential"
	}
	minioBucket := strings.TrimSpace(cfg.MinIO.BucketBackend)
	if serviceType == model.CicdServiceTypeFrontend {
		minioBucket = strings.TrimSpace(cfg.MinIO.BucketFrontend)
	}
	if minioBucket == "" {
		minioBucket = "backend-artifacts"
	}
	artifactRetain := cfg.DefaultArtifactRetain
	if artifactRetain <= 0 {
		artifactRetain = 10
	}
	waitMins := cfg.DefaultWaitMins
	if waitMins <= 0 {
		waitMins = 60
	}

	scriptPath := strings.TrimSpace(in.ScriptPath)
	if scriptPath == "" {
		scriptPath = "backend.jenkinsfile"
	}

	desc := strings.TrimSpace(in.Description)
	if desc == "" && svc != nil {
		desc = fmt.Sprintf("Yunshu 托管 — %s (%s)", svc.Name, svc.Identifier)
	}

	minioEndpoint := strings.TrimSpace(cfg.MinIO.Endpoint)
	if minioEndpoint == "" {
		minioEndpoint = "http://192.168.56.102:8021"
	}

	minioCred := strings.TrimSpace(cfg.Credentials.MinIO)
	if minioCred == "" {
		minioCred = "minio-credentials"
	}
	envLines := []string{
		"SSH_KEY_CREDENTIAL_ID=" + sshCred,
		"GIT_CREDENTIAL_ID=" + gitCred,
		"MINIO_CREDENTIAL_ID=" + minioCred,
		"MINIO_BUCKET=" + minioBucket,
		"MINIO_ENDPOINT=" + minioEndpoint,
	}
	if mc := strings.TrimSpace(cfg.MinIO.MCBin); mc != "" {
		envLines = append(envLines, "MC_BIN="+mc)
	}
	if alias := strings.TrimSpace(cfg.MinIO.MCAlias); alias != "" {
		envLines = append(envLines, "MC_ALIAS="+alias)
	}

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

	var params strings.Builder
	// MINIO_* / 凭据 ID 必须作为 Job 参数进入 env；EnvInject 在 Pipeline from SCM 下常不生效。
	params.WriteString(stringParam("MINIO_BUCKET", minioBucket, "MinIO 制品桶"))
	params.WriteString(stringParam("MINIO_ENDPOINT", minioEndpoint, "MinIO S3 地址（含 http://）"))
	params.WriteString(stringParam("MINIO_CREDENTIAL_ID", minioCred, "MinIO 凭据 ID（Jenkins Username with password）"))
	params.WriteString(stringParam("SSH_KEY_CREDENTIAL_ID", sshCred, "SSH 部署私钥凭据 ID（须为 Username with private key）"))
	params.WriteString(stringParam("GIT_CREDENTIAL_ID", gitCred, "Git 拉代码 SSH 凭据 ID"))
	params.WriteString(choiceParam("Tenv", []string{"dev", "test", "prod"}, "dev"))
	params.WriteString(choiceParam("publishMode", []string{"自动发布", "手动发布", "仅构建", "制品发布", "回滚"}, "仅构建"))
	params.WriteString(stringParam("SrcURL", gitURL, "业务仓库 SSH 地址"))
	params.WriteString(stringParam("branchName", refName, "构建分支或 TAG"))
	params.WriteString(choiceParam("buildType", buildTypeChoices(serviceType), buildType))
	params.WriteString(stringParam("buildshell", buildShell, "构建命令（不含 npm/yarn/mvn 前缀）"))

	k8sPipeline := in.K8sPipeline || serviceType == model.CicdServiceTypeMicro
	if k8sPipeline {
		k8s := in.K8sDefaults
		imageName := projectName
		imageTag := ""
		k8sNS := ""
		replicas := "1"
		containerPort := "8080"
		deployMethod := "kubectl"
		deployAction := "服务更新"
		deployConfigType := "使用deployment模板"
		deployConfigTemplate := "基础模板"
		if k8s != nil {
			if v := strings.TrimSpace(k8s.ImageName); v != "" {
				imageName = v
			}
			imageTag = strings.TrimSpace(k8s.ImageTag)
			k8sNS = strings.TrimSpace(k8s.K8sNamespace)
			if k8s.Replicas > 0 {
				replicas = fmt.Sprintf("%d", k8s.Replicas)
			}
			if k8s.ContainerPort > 0 {
				containerPort = fmt.Sprintf("%d", k8s.ContainerPort)
			}
			if v := strings.TrimSpace(k8s.DeployMethod); v != "" {
				deployMethod = v
			}
			if v := strings.TrimSpace(k8s.DeployAction); v != "" {
				deployAction = v
			}
			if v := strings.TrimSpace(k8s.DeployConfigType); v != "" {
				deployConfigType = v
			}
			if v := strings.TrimSpace(k8s.DeployConfigTemplate); v != "" {
				deployConfigTemplate = v
			}
		}
		if imageName == "" && svc != nil {
			imageName = strings.TrimSpace(svc.Identifier)
		}
		params.WriteString(stringParam("projectName", imageName, "镜像/服务名"))
		params.WriteString(stringParam("imageName", imageName, "Harbor 镜像名"))
		params.WriteString(stringParam("imageTag", imageTag, "镜像 Tag"))
		params.WriteString(stringParam("k8s_ns", k8sNS, "K8s 命名空间"))
		params.WriteString(stringParam("replicas", replicas, "副本数"))
		params.WriteString(stringParam("ContainerPort", containerPort, "容器端口"))
		params.WriteString(stringParam("deployMethod", deployMethod, "kubectl|helm"))
		params.WriteString(stringParam("deployConfigType", deployConfigType, "工作负载类型"))
		params.WriteString(stringParam("deployConfigTemplate", deployConfigTemplate, "部署模板"))
		params.WriteString(stringParam("FULL_IMAGE_NAME", "", "CD 发布时指定完整镜像地址（跳过构建）"))
		params.WriteString(stringParam("deployAction", deployAction, "部署动作"))
		params.WriteString(stringParam("HARBOR_URL", harborURL, "Harbor 地址（不含协议）"))
		params.WriteString(stringParam("HARBOR_HOST_IP", harborHostIP, "Harbor 解析 IP（Jenkins Pod hostAliases）"))
		params.WriteString(stringParam("HARBOR_CREDENTIAL_ID", harborCred, "Harbor 凭据 ID（Jenkins Username/Password）"))
		params.WriteString(stringParam("PROJECT_GROUP", harborProject, "Harbor 镜像项目名"))
		envLines = append(envLines,
			"HARBOR_URL="+harborURL,
			"HARBOR_HOST_IP="+harborHostIP,
			"HARBOR_CREDENTIAL_ID="+harborCred,
			"PROJECT_GROUP="+harborProject,
		)
	} else if serviceType == model.CicdServiceTypeFrontend {
		params.WriteString(stringParam("destPath", buildPath, "目标机部署目录"))
		params.WriteString(stringParam("nodeToolName", model.NodeToolNameFromConfig(ci), "Node.js Global Tool 名称（如 node18/node20，与 Jenkins 全局工具一致）"))
		params.WriteString(choiceParam("npmInstallMode", []string{"install", "ci", "skip"}, "install"))
		params.WriteString(choiceParam("cleanNpmCache", []string{"false", "true"}, "false"))
		params.WriteString(choiceParam("cleanNodeModules", []string{"false", "true"}, "false"))
	} else if !k8sPipeline {
		params.WriteString(stringParam("projectName", projectName, "JAR 命名"))
		params.WriteString(stringParam("buildPath", buildPath, "制品目录"))
		params.WriteString(stringParam("javaToolName", "jdk8", "JDK 工具名"))
		params.WriteString(stringParam("serverPort", "8080", "服务端口"))
		params.WriteString(stringParam("packConfigPaths", "", "随包配置目录"))
		params.WriteString(stringParam("destPath", "", "部署目录"))
		params.WriteString(choiceParam("cleanDeployDir", []string{"false", "true"}, "true"))
	}

	params.WriteString(stringParam("destIp", "", "部署服务器 IP，逗号分隔"))
	params.WriteString(stringParam("deployUser", "root", "部署目录属主"))
	params.WriteString(stringParam("deployGroup", "root", "部署目录属组"))
	if serviceType != model.CicdServiceTypeFrontend && !k8sPipeline {
		params.WriteString(stringParam("runUser", "app", "后端 JAR/进程运行用户"))
		params.WriteString(choiceParam("startScriptType", []string{"脚本模板", "自定义脚本"}, "脚本模板"))
		params.WriteString(stringParam("customScriptContent", "", "自定义 launch.sh 内容"))
		params.WriteString(stringParam("JVM_OPTS", "", "JVM 启动参数"))
	}
	if !k8sPipeline {
		params.WriteString(stringParam("deployAction", "服务更新", "CD 发布操作类型"))
	}
	params.WriteString(stringParam("artifactRetainCount", fmt.Sprintf("%d", artifactRetain), "MinIO 保留制品数"))
	params.WriteString(stringParam("waitMins", fmt.Sprintf("%d", waitMins), "手动发布等待分钟"))
	params.WriteString(stringParam("selectedVersion", "", "制品发布/回滚时指定 MinIO 制品文件名（Yunshu CD 发布必传，跳过 Jenkins input）"))
	params.WriteString(stringParam("emailUser", "", "构建通知邮箱"))
	params.WriteString(choiceParam("enableSonar", []string{"false", "true"}, "false"))
	params.WriteString(stringParam("SONAR_HOST_URL", strings.TrimSpace(cfg.Sonar.URL), "SonarQube 地址（字典 cicd_sonar_url）"))
	params.WriteString(stringParam("SONAR_TOKEN", "", "SonarQube Token（运行时由 Yunshu 注入，勿在 Job 默认值存明文）"))
	params.WriteString(stringParam("YUNSHU_CALLBACK_URL", strings.TrimSpace(cfg.Callback.CallbackURL), "阶段/门禁/制品回调 URL"))
	params.WriteString(stringParam("YUNSHU_CALLBACK_HMAC_SECRET", "", "回调 HMAC 密钥（运行时由 Yunshu 注入）"))
	params.WriteString(stringParam("YUNSHU_BUILD_RUN_ID", "", "Yunshu 构建/发布记录 ID（由 YUNSHU_RUN_KIND 区分）"))
	params.WriteString(stringParam("YUNSHU_RUN_KIND", "build", "build|release"))

	return fmt.Sprintf(`<?xml version='1.1' encoding='UTF-8'?>
<flow-definition plugin="workflow-job">
  <actions/>
  <description>%s</description>
  <keepDependencies>false</keepDependencies>
  <properties>
    <hudson.model.ParametersDefinitionProperty>
      <parameterDefinitions>
%s
      </parameterDefinitions>
    </hudson.model.ParametersDefinitionProperty>
    <hudson.plugins.envinject.EnvInjectJobProperty plugin="envinject">
      <info>
        <propertiesContent>%s</propertiesContent>
      </info>
    </hudson.plugins.envinject.EnvInjectJobProperty>
    <org.jenkinsci.plugins.workflow.job.properties.DisableConcurrentBuildsJobProperty>
      <abortPrevious>false</abortPrevious>
    </org.jenkinsci.plugins.workflow.job.properties.DisableConcurrentBuildsJobProperty>
  </properties>
  <definition class="org.jenkinsci.plugins.workflow.cps.CpsScmFlowDefinition" plugin="workflow-cps">
    <scm class="hudson.plugins.git.GitSCM" plugin="git">
      <configVersion>2</configVersion>
      <userRemoteConfigs>
        <hudson.plugins.git.UserRemoteConfig>
          <url>%s</url>
          <credentialsId>%s</credentialsId>
        </hudson.plugins.git.UserRemoteConfig>
      </userRemoteConfigs>
      <branches>
        <hudson.plugins.git.BranchSpec>
          <name>*/%s</name>
        </hudson.plugins.git.BranchSpec>
      </branches>
      <doGenerateSubmoduleConfigurations>false</doGenerateSubmoduleConfigurations>
      <submoduleCfg class="empty-list"/>
      <extensions/>
    </scm>
    <scriptPath>%s</scriptPath>
    <lightweight>true</lightweight>
  </definition>
  <triggers/>
  <disabled>false</disabled>
</flow-definition>`,
		xmlEscape(desc),
		params.String(),
		xmlEscape(strings.Join(envLines, "\n")),
		xmlEscape(jenkinsfileRepo),
		xmlEscape(gitCred),
		xmlEscape(jenkinsfileBranch),
		xmlEscape(scriptPath),
	)
}

func buildTypeChoices(serviceType string) []string {
	switch strings.ToLower(strings.TrimSpace(serviceType)) {
	case model.CicdServiceTypeFrontend:
		return []string{"npm", "yarn"}
	case model.CicdServiceTypeMicro:
		return []string{"mvn", "gradle", "docker"}
	default:
		return []string{"mvn", "gradle", "python", "golang"}
	}
}

func stringParam(name, defaultValue, description string) string {
	return fmt.Sprintf(`        <hudson.model.StringParameterDefinition>
          <name>%s</name>
          <description>%s</description>
          <defaultValue>%s</defaultValue>
          <trim>true</trim>
        </hudson.model.StringParameterDefinition>
`, xmlEscape(name), xmlEscape(description), xmlEscape(defaultValue))
}

func choiceParam(name string, choices []string, defaultValue string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`        <hudson.model.ChoiceParameterDefinition>
          <name>%s</name>
          <choices class="java.util.Arrays$ArrayList">
            <a class="string-array">
`, xmlEscape(name)))
	for _, c := range choices {
		b.WriteString("              <string>" + xmlEscape(c) + "</string>\n")
	}
	b.WriteString(`            </a>
          </choices>
        </hudson.model.ChoiceParameterDefinition>
`)
	_ = defaultValue // Jenkins Choice 无 defaultValue 节点，靠排序第一项；Yunshu 触发时显式传参
	return b.String()
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// FolderConfigXML 创建 Jenkins Folder 的最小 config.xml。
func FolderConfigXML() string {
	return `<?xml version='1.1' encoding='UTF-8'?>
<com.cloudbees.hudson.plugins.folder.Folder plugin="cloudbees-folder">
  <description>Yunshu CI/CD</description>
  <properties/>
  <folderViews class="com.cloudbees.hudson.plugins.folder.views.DefaultFolderViewHolder">
    <views>
      <hudson.model.AllView>
        <owner class="com.cloudbees.hudson.plugins.folder.Folder" reference="../.."/>
        <name>All</name>
        <filterExecutors>false</filterExecutors>
        <filterQueue>false</filterQueue>
        <properties class="hudson.model.View$PropertyList"/>
      </hudson.model.AllView>
    </views>
    <tabBar class="hudson.views.DefaultViewsTabBar"/>
  </folderViews>
  <healthMetrics/>
  <icon class="com.cloudbees.hudson.plugins.folder.icons.StockFolderIcon"/>
</com.cloudbees.hudson.plugins.folder.Folder>`
}
