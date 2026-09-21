package config

import "strings"

// ApplyDefaults 填充日志轮转等零值；compressSet 表示 yaml/env 是否显式配置了 log.compress。
func (c *LogConfig) ApplyDefaults(compressSet bool) {
	if strings.TrimSpace(c.FilePath) == "" {
		c.FilePath = "./logs"
	}
	if c.MaxSizeMB <= 0 {
		c.MaxSizeMB = 100
	}
	if c.MaxAgeDays <= 0 {
		c.MaxAgeDays = 30
	}
	if c.MaxBackups <= 0 {
		c.MaxBackups = 10
	}
	if !compressSet {
		c.Compress = true
	}
}
