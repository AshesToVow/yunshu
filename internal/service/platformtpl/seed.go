package platformtpl

import (
	"context"
	"embed"
	"io/fs"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

//go:embed seeds/*
var seedFS embed.FS

type seedDef struct {
	Key, Category, Name, Format, Description, File string
}

var seedCatalog = []seedDef{
	{
		Key: "cicd.apollo.backend-launch", Category: model.PlatformTemplateCategoryCicdSnippet,
		Name: "SSH 启动 Apollo 片段", Format: model.PlatformTemplateFormatShell,
		Description: "backend-launch-template 中 Apollo 启动参数；占位符 {{JAVA_BIN}} {{APOLLO_*}}",
		File: "seeds/cicd-apollo-backend-launch.sh",
	},
	{
		Key: "cicd.apollo.k8s-env", Category: model.PlatformTemplateCategoryCicdSnippet,
		Name: "K8s Deployment Apollo env", Format: model.PlatformTemplateFormatYAML,
		Description: "容器 env 中 APOLLO_OPTS / JAVA_OPTS 片段",
		File: "seeds/cicd-apollo-k8s-env.yaml",
	},
	{
		Key: "cicd.consul.register", Category: model.PlatformTemplateCategoryCicdSnippet,
		Name: "Consul 注册标签/注解", Format: model.PlatformTemplateFormatYAML,
		Description: "Pod template metadata：consul.register/* 与 yunshu-metrics",
		File: "seeds/cicd-consul-register.yaml",
	},
	{
		Key: "alert.channel.dingtalk.card", Category: model.PlatformTemplateCategoryAlert,
		Name: "钉钉告警卡片模板", Format: model.PlatformTemplateFormatGoTemplate,
		Description: "告警通道 Go template 预设（钉钉）；变量见通道预览",
		File: "seeds/alert-dingtalk-card.tmpl",
	},
	{
		Key: "alert.channel.wecom.markdown", Category: model.PlatformTemplateCategoryAlert,
		Name: "企微 Markdown 模板", Format: model.PlatformTemplateFormatGoTemplate,
		Description: "告警通道 Go template 预设（企业微信）",
		File: "seeds/alert-wecom-markdown.tmpl",
	},
	{
		Key: "inspect.report.default", Category: model.PlatformTemplateCategoryInspect,
		Name: "巡检报告标准版（引用）", Format: model.PlatformTemplateFormatHTML,
		Description: "平台级引用键；正文可覆盖 embed templates/report.html；项目级仍用 inspect_report_templates",
		File: "seeds/inspect-report-default.html",
	},
	{
		Key: "inspect.report.executive", Category: model.PlatformTemplateCategoryInspect,
		Name: "巡检报告高管摘要（引用）", Format: model.PlatformTemplateFormatHTML,
		Description: "覆盖 embed templates/executive.html 的平台级引用",
		File: "seeds/inspect-report-executive.html",
	},
	{
		Key: "loggie.pipeline.default", Category: model.PlatformTemplateCategoryLoggie,
		Name: "Loggie 默认 pipeline", Format: model.PlatformTemplateFormatYAML,
		Description: "Agent 侧 pipeline 参考配置",
		File: "seeds/loggie-pipeline-default.yml",
	},
}

// BuiltinContent 返回内置种子正文（未发布或未入库时的兜底）。
func BuiltinContent(templateKey string) (content, format string, ok bool) {
	for _, d := range seedCatalog {
		if d.Key != templateKey {
			continue
		}
		b, err := seedFS.ReadFile(d.File)
		if err != nil {
			return "", "", false
		}
		return string(b), d.Format, true
	}
	return "", "", false
}

// EnsureSeeded 启动/迁移后写入内置目录；已存在则跳过（不覆盖用户发布内容）。
func EnsureSeeded(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	svc := NewService(db)
	for _, d := range seedCatalog {
		var n int64
		if err := db.WithContext(ctx).Model(&model.PlatformTemplate{}).
			Where("template_key = ?", d.Key).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		body, err := fs.ReadFile(seedFS, d.File)
		if err != nil {
			continue
		}
		row := model.PlatformTemplate{
			TemplateKey: d.Key, Category: d.Category, Name: d.Name,
			Format: d.Format, Description: d.Description,
			Status: model.PlatformTemplateStatusEnabled, IsBuiltin: true,
		}
		if err := db.WithContext(ctx).Create(&row).Error; err != nil {
			return err
		}
		ver := model.PlatformTemplateVersion{
			TemplateID: row.ID, Version: 1,
			ContentInline: string(body), Checksum: checksum(string(body)),
			Remark: "builtin seed",
		}
		ver.StorageKey = svc.tryMirrorMinIO(ctx, d.Key, 1, ver.ContentInline, d.Format)
		if err := db.WithContext(ctx).Create(&ver).Error; err != nil {
			return err
		}
		if err := db.WithContext(ctx).Model(&row).Update("published_version", 1).Error; err != nil {
			return err
		}
	}
	return nil
}
