package config

import "strings"

// LoggieConfig Agent 安装与运维相关配置。
type LoggieConfig struct {
	// OfflineBinaryPath Yunshu 主机上的离线二进制路径（相对工作目录或绝对路径）。
	// 默认 deploy/loggie/binary/loggie；安装时从该文件 SFTP 上传，不再在线下载。
	OfflineBinaryPath string `mapstructure:"offline_binary_path"`
	UnitName          string `mapstructure:"unit_name"`
	DeployDir         string `mapstructure:"deploy_dir"`
	// DaemonSetImage K8s 集群采集 DaemonSet 使用的 Loggie 镜像。
	DaemonSetImage string `mapstructure:"daemonset_image"`
}

func (c LoggieConfig) Normalized() LoggieConfig {
	out := c
	if strings.TrimSpace(out.UnitName) == "" {
		out.UnitName = "loggie.service"
	}
	if strings.TrimSpace(out.DeployDir) == "" {
		out.DeployDir = "/export/loggie"
	}
	if strings.TrimSpace(out.OfflineBinaryPath) == "" {
		out.OfflineBinaryPath = "deploy/loggie/binary/loggie"
	}
	if strings.TrimSpace(out.DaemonSetImage) == "" {
		out.DaemonSetImage = "ghcr.io/loggie-io/loggie:v1.7.1"
	}
	return out
}
