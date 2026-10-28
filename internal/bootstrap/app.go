package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/dictconfig"
	"yunshu/internal/middleware"
	"yunshu/internal/pkg/casbinadapter"
	"yunshu/internal/pkg/database"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/providers"
	"yunshu/internal/service/system"
	"yunshu/internal/store"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// App 运行时全局依赖：配置、日志、DB、Redis、Casbin、邮件与 Gin 引擎。
type App struct {
	Config   *config.Config
	Logger   *logx.Logger
	DB       *gorm.DB
	Redis    *redis.Client
	Enforcer *casbin.SyncedEnforcer
	Mailer   mailer.Sender
	Engine   *gin.Engine
	// YamlK8sEventForwardBase config.yaml 底稿，供 Event 转发运行期从字典重新解析。
	YamlK8sEventForwardBase config.K8sEventForwardConfig
	// casbinWatchStop 停止策略变更订阅。
	casbinWatchStop func()
}

type Builder struct {
	app                     *App
	err                     error
	yamlMailBase            config.MailConfig            // config.yaml 中的 mail 底稿（字典覆盖前）
	yamlK8sEventForwardBase config.K8sEventForwardConfig // config.yaml 中的 k8s_event_forward 底稿
	yamlAuthBase            config.AuthConfig            // config.yaml 中的 auth 底稿（字典覆盖前）
}

func NewBuilder() *Builder {
	return &Builder{app: &App{}}
}

func (b *Builder) WithInfra(infra *providers.Infra) *Builder {
	if b.err != nil {
		return b
	}
	if infra == nil {
		b.err = errors.New("infra is required")
		return b
	}
	b.app.Config = infra.Config
	b.app.Logger = infra.Logger
	b.app.DB = infra.DB
	b.app.Redis = infra.Redis
	if infra.Config != nil {
		b.yamlMailBase = infra.Config.Mail
		b.yamlK8sEventForwardBase = infra.Config.K8sEventForward
		b.yamlAuthBase = infra.Config.Auth
		b.app.YamlK8sEventForwardBase = infra.Config.K8sEventForward
	}
	if infra.Logger != nil {
		logx.Init(infra.Logger)
	}
	return b
}

func (b *Builder) WithConfig(path string) *Builder {
	if b.err != nil {
		return b
	}

	cfg, err := config.Load(path)
	if err != nil {
		b.err = err
		return b
	}
	b.app.Config = cfg
	b.yamlMailBase = cfg.Mail
	b.yamlK8sEventForwardBase = cfg.K8sEventForward
	b.yamlAuthBase = cfg.Auth
	b.app.YamlK8sEventForwardBase = cfg.K8sEventForward
	return b
}

func (b *Builder) WithLogger() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil {
		b.err = errors.New("config is required before logger")
		return b
	}

	b.app.Logger = logx.New(b.app.Config.Log)
	logx.Init(b.app.Logger)
	return b
}

func (b *Builder) WithDatabase() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil {
		b.err = errors.New("config is required before database")
		return b
	}
	if b.app.Logger == nil {
		b.err = errors.New("logger is required before database")
		return b
	}

	db, err := database.Open(b.app.Config.Database, b.app.Logger, b.app.Config.Log.Level)
	if err != nil {
		b.err = err
		return b
	}
	b.app.DB = db
	return b
}

// WithMySQL 保留旧链式调用名；实际按 database.driver 连接（默认 mysql）。
func (b *Builder) WithMySQL() *Builder {
	return b.WithDatabase()
}

// WithDictOverrides 在 MySQL 已就绪后，从数据字典覆盖“运行期可变”的配置项（告警域 + 邮件 + K8s Event 转发）。
// 注意：mysql/redis/app.env 等启动期项仍以 env/yaml 为准，避免启动鸡生蛋。
func (b *Builder) WithDictOverrides() *Builder {
	if b.err != nil {
		return b
	}
	if b.app == nil || b.app.Config == nil || b.app.DB == nil {
		// MySQL 未就绪则跳过；不作为错误
		return b
	}
	b.applyDictConfigOverrides(context.Background(), defaultDictConfigOverrides())
	return b
}

func (b *Builder) WithRedis() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil {
		b.err = errors.New("config is required before redis")
		return b
	}

	cfg := b.app.Config.Redis
	// 显式收紧超时：go-redis 默认拨号重试在黑洞场景下单操作可挂 ~20s，
	// 而认证中间件每个请求都会同步访问 Redis，必须快速失败。
	// PubSub Receive(0) 与阻塞命令走各自的超时路径，不受此影响。
	b.app.Redis = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		MaxRetries:   1,
	})
	if err := b.app.Redis.Ping(context.Background()).Err(); err != nil {
		b.err = fmt.Errorf("redis ping: %w", err)
		return b
	}
	return b
}

func (b *Builder) WithCasbin() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil || b.app.DB == nil {
		b.err = errors.New("config and mysql are required before casbin")
		return b
	}

	// Defensive cleanup: if casbin_rule contains malformed rows (e.g. empty ptype),
	// casbin's LoadPolicyLine may panic. Keep startup resilient by pruning obviously
	// invalid rows before adapter loads policies.
	// Keep Casbin startup resilient: prune malformed rules that can make casbin panic
	// when parsing policy lines (e.g. invalid/garbage ptype).
	//
	// Valid Casbin ptype is typically: p, g, p2, g2, ...
	_ = database.PruneInvalidCasbinRules(b.app.DB)

	adapter := casbinadapter.NewSafeGormAdapter(b.app.DB, "casbin_rule")

	enforcer, err := casbin.NewSyncedEnforcer(b.app.Config.Casbin.ModelPath, adapter)
	if err != nil {
		b.err = err
		return b
	}
	if err = enforcer.LoadPolicy(); err != nil {
		b.err = fmt.Errorf("casbin load policy: %w", err)
		return b
	}
	policyCount := len(enforcer.GetPolicy())
	groupingCount := len(enforcer.GetGroupingPolicy())
	if policyCount == 0 && groupingCount == 0 {
		slog.Default().With("component", "casbin").Warn("Loaded zero Casbin rules; authorize may deny all until policies are seeded")
	} else {
		slog.Default().With("component", "casbin").Info("Loaded Casbin policy", "p_rules", policyCount, "g_rules", groupingCount)
	}
	// 冒烟：确认 model 可执行 Enforce（adapter/模型损坏时此处会报错）
	if _, err = enforcer.Enforce("__casbin_smoke__", "/__smoke__", "GET"); err != nil {
		b.err = fmt.Errorf("casbin enforce smoke test: %w", err)
		return b
	}

	// 多副本部署：变更后通过 Redis 广播即时重载（推送），定时全量重载仅作兜底。
	// 推送可用时默认 300s 兜底（防消息丢失/订阅重连期间漏更新）；无 Redis 时退回 30s 轮询。
	// 显式配置为负数表示关闭轮询（推送仍生效）。
	interval := b.app.Config.Casbin.AutoLoadIntervalSeconds
	if interval == 0 {
		interval = 30
		if b.app.Redis != nil {
			interval = 300
		}
	}
	if interval > 0 {
		enforcer.StartAutoLoadPolicy(time.Duration(interval) * time.Second)
		slog.Default().With("component", "casbin").Info("Started Casbin auto policy reload", "interval_seconds", interval)
	} else {
		slog.Default().With("component", "casbin").Info("Casbin auto policy reload disabled")
	}

	// Redis 可用时启用推送：本副本写入策略后广播，其他副本收到即重载，
	// 无需等待下一轮全量轮询；轮询保留为兜底（Redis 抖动/消息丢失）。
	if b.app.Redis != nil {
		source := casbinInstanceID()
		system.SetPolicyChangeHook(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := store.PublishCasbinPolicyChange(ctx, b.app.Redis, source); err != nil {
				slog.Default().With("component", "casbin").Warn("publish policy change failed", "error", err)
			}
		})
		b.app.casbinWatchStop = store.WatchCasbinPolicyChanges(
			context.Background(), b.app.Redis, slog.Default().With("component", "casbin"), source, enforcer.LoadPolicy)
		slog.Default().With("component", "casbin").Info("Casbin policy push enabled", "source", source)
	}

	b.app.Enforcer = enforcer
	return b
}

// casbinInstanceID 生成副本标识，用于订阅端忽略自己发布的消息。
func casbinInstanceID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		host = "unknown-host"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

func (b *Builder) WithMailer() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil {
		b.err = errors.New("config is required before mailer")
		return b
	}

	resolved := b.app.Config.Mail
	if b.app.DB != nil {
		resolved = dictconfig.ResolveMailConfig(context.Background(), b.app.DB, b.yamlMailBase, dictconfig.DefaultMailDictTypes())
		b.app.Config.Mail = resolved
		b.app.Mailer = mailer.NewDynamicSender(&mailer.DictMailResolver{
			DB:       b.app.DB,
			YAMLBase: b.yamlMailBase,
		})
		enabled := b.app.Mailer.Enabled()
		slog.Default().With("component", "mail").Info("Initialized mail sender (dict-first, reload on send)",
			"enabled", enabled,
			"host", resolved.Host,
			"port", resolved.Port,
			"from", resolved.FromEmail,
		)
	} else {
		b.app.Mailer = mailer.NewSMTPSender(resolved)
	}
	return b
}

func (b *Builder) WithGin() *Builder {
	if b.err != nil {
		return b
	}
	if b.app.Config == nil || b.app.Logger == nil {
		b.err = errors.New("config and logger are required before gin")
		return b
	}

	if b.app.Config.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	if err := engine.SetTrustedProxies(b.app.Config.HTTP.TrustedProxies); err != nil {
		b.err = fmt.Errorf("invalid http.trusted_proxies: %w", err)
		return b
	}
	engine.Use(middleware.Recovery(b.app.Logger))
	cspOn := true
	if b.app.Config.Auth.CSPEnabled != nil {
		cspOn = *b.app.Config.Auth.CSPEnabled
	}
	engine.Use(middleware.SecurityHeaders(cspOn))
	engine.Use(middleware.RequestLogger(b.app.Logger))
	engine.Use(middleware.RequestBudget(time.Duration(b.app.Config.HTTP.RequestBudgetSeconds) * time.Second))
	engine.Use(middleware.ErrorHandler())
	middleware.RegisterOpsEndpoints(engine, b.app.DB, b.app.Redis, time.Now())
	b.app.Engine = engine
	return b
}

func (b *Builder) Build() (*App, error) {
	if b.err != nil {
		return nil, b.err
	}

	// 启动期安全闸门：不仅校验密钥存在，还要校验其强度。
	// 生产环境下弱密钥/占位值直接终止启动；非生产仅告警，避免阻断本地开发。
	warnings, err := b.app.Config.Validate()
	for _, w := range warnings {
		if b.app.Logger != nil && b.app.Logger.Info != nil {
			b.app.Logger.Info.Warn("insecure configuration detected", "detail", w)
		} else {
			slog.Warn("insecure configuration detected", "detail", w)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return b.app, b.err
}

func (a *App) Close() error {
	var errs []error

	if a.casbinWatchStop != nil {
		a.casbinWatchStop()
		a.casbinWatchStop = nil
	}

	if a.Redis != nil {
		if err := a.Redis.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if a.DB != nil {
		sqlDB, err := a.DB.DB()
		if err == nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				errs = append(errs, closeErr)
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}
