package cicd

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
	"yunshu/internal/pkg/jenkins"
)

var (
	packagePathFullRe = regexp.MustCompile(`(?i)([\w-]+-artifacts)/([\w./-]+\.(?:tar\.gz|jar|bin))`)
	artifactFileRe    = regexp.MustCompile(`(?i)([\w.-]+-\d{8}_\d{6}-[\w]+\.(?:tar\.gz|jar|bin))`)
	deployInfoJarRe   = regexp.MustCompile(`(?i)JAR:\s*([\w.-]+-\d{8}_\d{6}-[\w]+\.(?:tar\.gz|jar|bin))`)
	helmPushOCIRe     = regexp.MustCompile(`(?i)helm\s+push\s+\S+\s+(oci://[^\s'"]+)`)
	helmChartRepoRe   = regexp.MustCompile(`(?i)https?://[^\s'"]+/chartrepo/[^\s'"]+`)
	helmChartTgzRe    = regexp.MustCompile(`(?i)([\w.-]+)-(\d+(?:\.\d+)*)\.tgz`)
	imageAddressRe    = regexp.MustCompile(`(?i)((?:[\w.-]+\.)*[\w.-]+/[\w.-]+/[\w.-]+:[\w._-]+)`)
	imageTaggedRe     = regexp.MustCompile(`(?i)Successfully tagged (\S+)`)
	imagePushRepoRe   = regexp.MustCompile(`(?i)The push refers to repository \[([^\]]+)\]`)
)

// JenkinsSyncResult Jenkins Job 创建/更新结果。
type JenkinsSyncResult struct {
	JobName    string `json:"job_name"`
	ScriptPath string `json:"script_path"`
	Created    bool   `json:"created"`
	Updated    bool   `json:"updated"`
}

func (s *Service) syncJenkinsJob(ctx context.Context, svc *model.CicdService, ci *model.CicdCiConfig) (*JenkinsSyncResult, error) {
	client, cfg, err := s.jenkinsClient(ctx)
	if err != nil {
		return nil, err
	}
	jobName := strings.TrimSpace(svc.JenkinsJob)
	if jobName == "" {
		jobName = strings.TrimSpace(svc.Identifier)
	}
	usesK8s := s.serviceUsesK8sPipeline(ctx, svc)
	scriptPath := dictconfig.JenkinsScriptPath(cfg, svc.ServiceType, usesK8s)
	existsBefore, err := client.JobExists(ctx, jobName)
	if err != nil {
		return nil, err
	}
	var k8sDefaults *model.CicdDeployConfig
	if usesK8s {
		k8sDefaults = s.primaryContainerDeployConfig(ctx, svc.ID)
	}
	xml := jenkins.BuildPipelineJobConfigXML(jenkins.JobTemplateInput{
		Service:     svc,
		CiConfig:    ci,
		Cfg:         cfg,
		ScriptPath:  scriptPath,
		K8sPipeline: usesK8s,
		K8sDefaults: k8sDefaults,
	})
	if err := client.EnsurePipelineJob(ctx, jobName, xml); err != nil {
		return nil, err
	}
	result := &JenkinsSyncResult{
		JobName:    jobName,
		ScriptPath: scriptPath,
		Updated:    existsBefore,
		Created:    !existsBefore,
	}
	if strings.TrimSpace(svc.JenkinsJob) == "" {
		_ = s.db.WithContext(ctx).Model(svc).Update("jenkins_job", jobName).Error
	}
	return result, nil
}

// resolveJenkinsJobName Jenkins Job 名：优先 jenkins_job，否则用服务标识符。
func resolveJenkinsJobName(svc *model.CicdService) string {
	if svc == nil {
		return ""
	}
	if v := strings.TrimSpace(svc.JenkinsJob); v != "" {
		return v
	}
	return strings.TrimSpace(svc.Identifier)
}

func minioFolderHints(svc *model.CicdService, ci *model.CicdCiConfig) []string {
	seen := make(map[string]struct{})
	var hints []string
	add := func(v string) {
		v = strings.Trim(strings.TrimSpace(v), "/")
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		hints = append(hints, v)
	}
	add(resolveJenkinsJobName(svc))
	if ci != nil {
		add(ci.ProjectName)
	}
	if svc != nil {
		add(svc.Identifier)
	}
	return hints
}

func helmChartNameHints(svc *model.CicdService, ci *model.CicdCiConfig, dc *model.CicdDeployConfig) []string {
	seen := make(map[string]struct{})
	var hints []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		hints = append(hints, v)
	}
	if dc != nil {
		add(dc.ImageName)
	}
	if ci != nil {
		add(ci.ProjectName)
	}
	if svc != nil {
		add(svc.Identifier)
		add(svc.Name)
	}
	return hints
}

func harborOCIChartRef(harbor config.HarborConfig, chartName string) string {
	host := strings.TrimSpace(harbor.URL)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(host, "/")
	project := strings.TrimSpace(harbor.ProjectGroup)
	chartName = strings.TrimSpace(chartName)
	if host == "" || project == "" || chartName == "" {
		return ""
	}
	return fmt.Sprintf("oci://%s/%s/%s", host, project, chartName)
}

// extractHelmChartRefFromConsole 从 Jenkins Console 解析 Harbor Helm Chart（OCI 或 chartrepo）。
func extractHelmChartRefFromConsole(log string, harbor config.HarborConfig, chartHints ...string) string {
	log = strings.TrimSpace(log)
	if log == "" {
		return ""
	}
	if m := helmPushOCIRe.FindStringSubmatch(log); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	if m := helmChartRepoRe.FindStringSubmatch(log); len(m) > 0 {
		return strings.TrimSpace(m[0])
	}
	for _, hint := range chartHints {
		hint = strings.TrimSpace(hint)
		if hint == "" {
			continue
		}
		for _, m := range helmChartTgzRe.FindAllStringSubmatch(log, -1) {
			if len(m) < 3 {
				continue
			}
			if !strings.EqualFold(m[1], hint) {
				continue
			}
			base := harborOCIChartRef(harbor, hint)
			if base == "" {
				continue
			}
			return base + ":" + m[2]
		}
	}
	return ""
}

type resolvedBuildArtifacts struct {
	PackagePath  string
	ImageAddress string
}

func (s *Service) resolveBuildArtifactsFromLog(ctx context.Context, svc model.CicdService, ci model.CicdCiConfig, logText string) resolvedBuildArtifacts {
	cfg := s.resolvedConfig(ctx)
	usesK8s := s.serviceUsesK8sPipeline(ctx, &svc)
	dc := s.primaryContainerDeployConfig(ctx, svc.ID)
	folderHints := minioFolderHints(&svc, &ci)
	bucket := dictconfig.MinIOBucketForService(cfg, svc.ServiceType)

	out := resolvedBuildArtifacts{}
	if bucket != "" {
		out.PackagePath = extractPackagePathFromConsole(logText, bucket, folderHints...)
	}
	if usesK8s {
		hint := s.buildImageNameHint(ctx, svc)
		out.ImageAddress = extractImageAddressFromConsole(logText, hint)
		if out.PackagePath == "" {
			out.PackagePath = extractHelmChartRefFromConsole(logText, cfg.Harbor, helmChartNameHints(&svc, &ci, dc)...)
		}
	}
	return out
}

// extractPackagePathFromConsole 从 Jenkins Console 解析 MinIO 制品路径（bucket/folder/file）。
func extractPackagePathFromConsole(log, bucket string, folderHints ...string) string {
	log = strings.TrimSpace(log)
	if log == "" {
		return ""
	}
	if m := packagePathFullRe.FindStringSubmatch(log); len(m) >= 3 {
		return m[1] + "/" + strings.TrimPrefix(m[2], "/")
	}
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return ""
	}
	for _, folder := range folderHints {
		folder = strings.Trim(strings.TrimSpace(folder), "/")
		if folder == "" {
			continue
		}
		prefix := bucket + "/" + folder + "/"
		if idx := strings.LastIndex(log, prefix); idx >= 0 {
			rest := log[idx+len(prefix):]
			if end := strings.IndexAny(rest, " \t\r\n\"'"); end > 0 {
				rest = rest[:end]
			}
			rest = strings.Trim(rest, ".,;)")
			if rest != "" && isDeployArtifactName(rest) {
				return prefix + rest
			}
		}
	}
	artifactName := ""
	if m := deployInfoJarRe.FindStringSubmatch(log); len(m) >= 2 {
		artifactName = strings.TrimSpace(m[1])
	}
	if artifactName == "" {
		if m := artifactFileRe.FindAllString(log, -1); len(m) > 0 {
			artifactName = m[len(m)-1]
		}
	}
	if artifactName == "" {
		return ""
	}
	for _, folder := range folderHints {
		if path := buildArtifactPackagePath(folder, bucket, artifactName); path != "" {
			return path
		}
	}
	return ""
}

func buildArtifactPackagePath(jobName, bucket, artifactName string) string {
	jobName = strings.Trim(strings.TrimSpace(jobName), "/")
	bucket = strings.TrimSpace(bucket)
	artifactName = strings.TrimSpace(artifactName)
	if jobName == "" || bucket == "" || artifactName == "" {
		return ""
	}
	return bucket + "/" + jobName + "/" + artifactName
}

func (s *Service) buildImageNameHint(ctx context.Context, svc model.CicdService) string {
	if dc := s.primaryContainerDeployConfig(ctx, svc.ID); dc != nil {
		if v := strings.TrimSpace(dc.ImageName); v != "" {
			return strings.ToLower(v)
		}
	}
	var ci model.CicdCiConfig
	if err := s.db.WithContext(ctx).Where("service_id = ?", svc.ID).First(&ci).Error; err == nil {
		if v := strings.TrimSpace(ci.ProjectName); v != "" {
			return strings.ToLower(v)
		}
	}
	return strings.ToLower(strings.TrimSpace(svc.Identifier))
}

// extractImageAddressFromConsole 从 Jenkins Console 解析 Harbor 完整镜像地址。
func extractImageAddressFromConsole(log string, hints ...string) string {
	log = strings.TrimSpace(log)
	if log == "" {
		return ""
	}
	hint := strings.ToLower(strings.TrimSpace(firstNonEmpty(hints...)))

	if tagged := pickBusinessImage(collectSubmatch1(imageTaggedRe, log), hint); tagged != "" {
		return tagged
	}
	if pushed := pickBusinessImage(collectSubmatch1(imagePushRepoRe, log), hint); pushed != "" {
		return pushed
	}

	const marker = "镜像Tag:"
	if idx := strings.LastIndex(log, marker); idx >= 0 {
		tagLine := log[idx:]
		if tag := strings.TrimSpace(strings.TrimPrefix(tagLine, marker)); tag != "" {
			for _, img := range imageAddressRe.FindAllString(log, -1) {
				img = strings.TrimSpace(img)
				if strings.HasSuffix(strings.ToLower(img), ":"+strings.ToLower(tag)) && !isIgnoredCIImage(img) {
					if hint == "" || strings.Contains(strings.ToLower(img), "/"+hint+":") {
						return img
					}
				}
			}
		}
	}

	matches := imageAddressRe.FindAllString(log, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		img := strings.TrimSpace(matches[i])
		if isIgnoredCIImage(img) {
			continue
		}
		if hint != "" && !strings.Contains(strings.ToLower(img), "/"+hint+":") {
			continue
		}
		return img
	}
	for i := len(matches) - 1; i >= 0; i-- {
		img := strings.TrimSpace(matches[i])
		if !isIgnoredCIImage(img) {
			return img
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func collectSubmatch1(re *regexp.Regexp, log string) []string {
	matches := re.FindAllStringSubmatch(log, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			out = append(out, strings.TrimSpace(m[1]))
		}
	}
	return out
}

func pickBusinessImage(candidates []string, hint string) string {
	for i := len(candidates) - 1; i >= 0; i-- {
		img := candidates[i]
		if isIgnoredCIImage(img) {
			continue
		}
		if hint != "" && !strings.Contains(strings.ToLower(img), "/"+hint) {
			continue
		}
		return img
	}
	for i := len(candidates) - 1; i >= 0; i-- {
		if !isIgnoredCIImage(candidates[i]) {
			return candidates[i]
		}
	}
	return ""
}

func isIgnoredCIImage(img string) bool {
	lower := strings.ToLower(strings.TrimSpace(img))
	if lower == "" {
		return true
	}
	ignored := []string{
		"/inbound-agent:",
		"/jenkins/inbound-agent:",
		"/maven:",
		"/docker:",
		"/nodejs:",
		"/kubectl:",
		"/helm_jq:",
		"/helm:",
		"/pause:",
	}
	for _, part := range ignored {
		if strings.Contains(lower, part) {
			return true
		}
	}
	return false
}
