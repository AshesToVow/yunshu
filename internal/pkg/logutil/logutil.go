package logutil

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	global     *slog.Logger
	globalMu   sync.RWMutex
	ctxKey     = &struct{}{}
	loggerOnce sync.Once
)

type Config struct {
	Level      slog.Level
	Format     string
	Output     string
	LogDir     string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
	AddSource  bool
}

func DefaultConfig() Config {
	return Config{Level: slog.LevelDebug, Format: "text", Output: "console", AddSource: true}
}

func ProductionConfig() Config {
	return Config{
		Level: slog.LevelInfo, Format: "json", Output: "both", LogDir: "./logs",
		MaxSizeMB: 100, MaxBackups: 7, MaxAgeDays: 30, AddSource: false,
	}
}

func Init(cfg Config) error {
	loggerOnce.Do(func() {
		global = slog.New(buildHandler(cfg, buildWriters(cfg)))
	})
	return nil
}

func MustInit(cfg Config) {
	if err := Init(cfg); err != nil {
		panic("logutil.Init failed: " + err.Error())
	}
}

func SetDefaultLogger(l *slog.Logger) {
	if l == nil {
		return
	}
	globalMu.Lock()
	defer globalMu.Unlock()
	global = l
}

func Default() *slog.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if global == nil {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return global
}

func Ctx(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return Default()
	}
	if l, ok := ctx.Value(ctxKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return Default()
}

func WithContext(ctx context.Context, l *slog.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKey, l)
}

func WithFields(args ...any) *slog.Logger { return Default().With(args...) }

func CtxWithFields(ctx context.Context, args ...any) *slog.Logger { return Ctx(ctx).With(args...) }

func Info(msg string, args ...any)  { Default().Info(msg, args...) }
func Warn(msg string, args ...any)  { Default().Warn(msg, args...) }
func Error(msg string, args ...any) { Default().Error(msg, args...) }
func Debug(msg string, args ...any) { Default().Debug(msg, args...) }

func ComponentLogger(ctx context.Context, name string) *slog.Logger {
	return Ctx(ctx).With("component", name)
}

func Worker(name string) *Component {
	return &Component{log: Default().With("layer", "worker", "component", name)}
}

func WorkerCtx(ctx context.Context, name string) *slog.Logger {
	return Ctx(ctx).With("layer", "worker", "component", name)
}

func HTTP(name string) *Component {
	return &Component{log: Default().With("layer", "http", "component", name)}
}

func HTTPCtx(ctx context.Context, name string) *slog.Logger {
	return Ctx(ctx).With("layer", "http", "component", name)
}

func API(name string) *slog.Logger {
	return Default().With("layer", "api", "component", name)
}

func APICtx(ctx context.Context, name string) *slog.Logger {
	return Ctx(ctx).With("layer", "api", "component", name)
}

func GRPC(name string) *slog.Logger {
	return Default().With("layer", "grpc", "component", name)
}

func GRPCCtx(ctx context.Context, name string) *slog.Logger {
	return Ctx(ctx).With("layer", "grpc", "component", name)
}

func DAO(name string) *slog.Logger {
	return Default().With("layer", "dao", "component", name)
}

func DAOCtx(ctx context.Context, name string) *slog.Logger {
	return Ctx(ctx).With("layer", "dao", "component", name)
}

func buildWriters(cfg Config) []io.Writer {
	var writers []io.Writer
	if cfg.Output == "console" || cfg.Output == "both" {
		writers = append(writers, os.Stdout)
	}
	if (cfg.Output == "file" || cfg.Output == "both") && cfg.LogDir != "" {
		f, err := os.OpenFile(cfg.LogDir+"/app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			writers = append(writers, os.Stderr)
		} else {
			writers = append(writers, f)
		}
	}
	if len(writers) == 0 {
		writers = []io.Writer{os.Stdout}
	}
	return writers
}

func buildHandler(cfg Config, writers []io.Writer) slog.Handler {
	multi := io.MultiWriter(writers...)
	opts := &slog.HandlerOptions{
		Level:     cfg.Level,
		AddSource: cfg.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				return shortenSource(a)
			}
			return a
		},
	}
	if cfg.Format == "json" {
		return slog.NewJSONHandler(multi, opts)
	}
	return slog.NewTextHandler(multi, opts)
}

func shortenSource(a slog.Attr) slog.Attr {
	if a.Value.Kind() != slog.KindAny {
		return a
	}
	src, ok := a.Value.Any().(*slog.Source)
	if !ok || src == nil {
		return a
	}
	file := src.File
	if i := lastIndex(file, "yunshu/"); i >= 0 {
		file = file[i+len("yunshu/"):]
	}
	return slog.Any(slog.SourceKey, &slog.Source{Function: trimFunc(src.Function), File: file, Line: src.Line})
}

func lastIndex(s, sep string) int {
	for i := len(s) - len(sep); i >= 0; i-- {
		if s[i:i+len(sep)] == sep {
			return i
		}
	}
	return -1
}

func trimFunc(fn string) string {
	if i := lastIndexOf(fn, '/'); i >= 0 {
		fn = fn[i+1:]
	}
	if i := lastIndexOf(fn, '.'); i >= 0 {
		return fn[i+1:]
	}
	return fn
}

func lastIndexOf(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
