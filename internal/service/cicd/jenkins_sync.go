package cicd

import (
	"context"
	"regexp"
	"strings"

	"yunshu/internal/dictconfig"
	"yunshu/internal/model"
	"yunshu/internal/pkg/jenkins"
)

var (
	packagePathFullRe = regexp.MustCompile(`(?i)([\w-]+-artifacts)/([\w./-]+\.(?:tar\.gz|jar|bin))`)
	artifactFileRe    = regexp.MustCompile(`(?i)([\w-]+-\d{8}_\d{6}-[\w]+\.(?:tar\.gz|jar|bin))`)
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

// extractPackagePathFromConsole 从 Jenkins Console 解析 MinIO 制品路径（bucket/job/file）。
func extractPackagePathFromConsole(log, jobName, bucket string) string {
	log = strings.TrimSpace(log)
	if log == "" {
		return ""
	}
	if m := packagePathFullRe.FindStringSubmatch(log); len(m) >= 3 {
		return m[1] + "/" + strings.TrimPrefix(m[2], "/")
	}
	jobName = strings.Trim(strings.TrimSpace(jobName), "/")
	bucket = strings.TrimSpace(bucket)
	if jobName == "" || bucket == "" {
		return ""
	}
	prefix := bucket + "/" + jobName + "/"
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
	if m := artifactFileRe.FindAllString(log, -1); len(m) > 0 {
		name := m[len(m)-1]
		return bucket + "/" + jobName + "/" + name
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
