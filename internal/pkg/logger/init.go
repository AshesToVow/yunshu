package logger

import (
	"log/slog"
)

var defaultLogger *Logger

// SetDefault 保存进程级 Logger 实例。
func SetDefault(l *Logger) {
	defaultLogger = l
}

// Default 返回进程级 Logger。
func Default() *Logger {
	return defaultLogger
}

// Init 初始化全局 Logger 并注册 slog.SetDefault（Info/Warn→info.log，Error→error.log）。
func Init(l *Logger, extra ...ContextExtractors) {
	SetDefault(l)
	if l != nil && l.Default != nil {
		slog.SetDefault(l.Default)
	}
	for _, ext := range extra {
		RegisterContextExtractors(ext)
	}
}

// Sync 刷新日志缓冲；slog 标准输出一般无需 flush，保留接口便于进程退出时调用。
func Sync() {
	if defaultLogger == nil {
		return
	}
	defaultLogger.Sync()
}

// Sync 实例方法，便于进程退出时调用。
func (l *Logger) Sync() {}
