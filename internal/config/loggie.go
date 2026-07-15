package config

import "strings"

// LoggieConfig Agent 安装与运维相关配置。
type LoggieConfig struct {
	BinaryURL string `mapstructure:"binary_url"` // 可含 {arch}，如 https://example/loggie-linux-{arch}
	UnitName  string `mapstructure:"unit_name"`
	DeployDir string `mapstructure:"deploy_dir"`
}

func (c LoggieConfig) Normalized() LoggieConfig {
	out := c
	if strings.TrimSpace(out.UnitName) == "" {
		out.UnitName = "loggie.service"
	}
	if strings.TrimSpace(out.DeployDir) == "" {
		out.DeployDir = "/export/loggie"
	}
	return out
}
