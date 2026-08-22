package logger

import (
	"io"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"

	"yunshu/internal/config"
)

// openRotatingLogFile 返回带大小/天数轮转的文件写入器（lumberjack）。
func openRotatingLogFile(cfg config.LogConfig, logDir, channel string) io.Writer {
	fileName := filepath.Join(logDir, channel+".log")
	return &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    cfg.MaxSizeMB,
		MaxAge:     cfg.MaxAgeDays,
		MaxBackups: cfg.MaxBackups,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
}
