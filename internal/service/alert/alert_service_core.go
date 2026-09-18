package alert

import (
	"context"
	"crypto/cipher"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"yunshu/internal/config"
	"yunshu/internal/interfaces"
	cryptox "yunshu/internal/pkg/crypto"
	"yunshu/internal/pkg/lifecycle"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/service/logplatform"

	"github.com/redis/go-redis/v9"
)

// 历史 monitor_pipeline 取值 prometheus/platform 仍可能存在于旧数据；新写入以数据源为主，见 resolveAlertDatasourceMeta。

type AlertChannelListQuery struct {
	Keyword string `form:"keyword"`
}

type AlertEventListQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
	Cluster  string `form:"cluster"`
	AlertIP  string `form:"alertIP"`
	Status   string `form:"status"`
	// Severity 单值或逗号分隔；支持 critical/warning/info 以及别名 p1/p2/p3
	Severity        string `form:"severity"`
	MonitorPipeline string `form:"monitorPipeline"`
	DatasourceID    uint   `form:"datasourceId"`
	GroupKey        string `form:"groupKey"`
	// Fingerprint 按告警指纹筛选投递/跳过留痕（兼容旧数据：group_key 或 payload 内 fingerprint）
	Fingerprint string `form:"fingerprint"`
	// Category 策略分类：delivery|routing|silence|inhibition|timing|resolved|failure|other
	Category  string `form:"category"`
	ProjectID uint   `form:"project_id"`
	// ProjectIDAlias 兼容历史 query projectId
	ProjectIDAlias uint `form:"projectId"`
}

type AlertChannelUpsertRequest struct {
	Name        string `json:"name" binding:"required,max=64"`
	Type        string `json:"type"`
	URL         string `json:"url" binding:"omitempty,url,max=1024"`
	Secret      string `json:"secret"`
	HeadersJSON string `json:"headers_json"`
	Enabled     *bool  `json:"enabled"`
	TimeoutMS   int    `json:"timeout_ms"`
}

type AlertTestRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	Severity string `json:"severity"`
}

type AlertChannelTestResult struct {
	Success        bool   `json:"success"`
	HTTPStatusCode int    `json:"http_status_code,omitempty"`
	ResponseBody   string `json:"response_body,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

type AlertManagerPayload struct {
	Status            string              `json:"status"`
	Version           string              `json:"version"`
	Receiver          string              `json:"receiver"`
	GroupLabels       map[string]string   `json:"groupLabels"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	ExternalURL       string              `json:"externalURL"`
	TruncatedAlerts   int                 `json:"truncatedAlerts"`
	Alerts            []AlertManagerAlert `json:"alerts"`
}

type AlertManagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
	// SkipGroupTiming 仅服务端使用（不入 JSON）：云到期「立即评估」等路径为 true 时，跳过 Redis group_wait/repeat 节流，保证立刻投递。
	SkipGroupTiming bool `json:"-"`
}

type AlertService struct {
	redis       *redis.Client
	mailer      mailer.Sender
	cfg         config.AlertConfig
	enrichQueue chan promEnrichTask

	silenceSvc     *AlertSilenceService
	maintenanceSvc *AlertMaintenanceService
	assigneeSvc    *AlertRuleAssigneeService
	dutySvc        *AlertDutyService

	monitorEvalCancel context.CancelFunc
	monitorEvalMu     sync.Mutex
	aead              cipher.AEAD
	cloudExpiryState  map[string]bool
	cloudExpiryEvalMu sync.Mutex
	// 无 Redis 时云到期规则按 synthetic rule id 记录上次「按 Cron 触发评估」时间
	cloudExpiryNoRedisLastEval map[uint]time.Time
	// 无 Redis 时内置监控规则 firing 状态（key=monitor_rule_{id}）
	monitorNoRedisActive map[string]bool

	// 可选依赖：告警抑制、订阅树路由
	inhibitionSvc      *AlertInhibitionService   // 告警抑制服务
	subscriptionSvc    *AlertSubscriptionService // 订阅树服务
	receiverGroupCache *ReceiverGroupCache       // 接收组缓存

	metrics           *AlertMetrics // Prometheus自监控指标
	metricsUpdater    *AlertMetricsUpdater
	timingLeaderToken string

	eventRepo            interfaces.AlertEventRepository
	channelRepo          interfaces.AlertChannelRepository
	monitorRuleRepo      interfaces.AlertMonitorRuleRepository
	datasourceRepo       interfaces.AlertDatasourceRepository
	projectRepo          interfaces.ProjectRepository
	firingDeliveryRepo   interfaces.AlertFiringDeliveryRepository
	cloudExpiryRepo      interfaces.CloudExpiryRuleRepository
	cloudAccountRepo     interfaces.CloudAccountRepository
	ackRepo              interfaces.AlertAckRepository
	progressNoteRepo     interfaces.AlertProgressNoteRepository
	curHisRepo           interfaces.AlertCurHisRepository
	promqlSavedQueryRepo interfaces.PromqlSavedQueryRepository
	changeEventRepo      interfaces.ChangeEventRepository
	dictEntryRepo        interfaces.DictEntryRepository
	receiverGroupRepo    interfaces.AlertReceiverGroupRepository

	alertStateSvc AlertStateService

	// 可选：证据包拉取项目日志采样
	logSearch *logplatform.LogSearchService

	// Redis 不可用时的进程内分组节流与升级队列
	localMu sync.Mutex
	local   *processLocalAlertState
}

// SetLogSearch 注入日志检索（证据包）；可在 DI 后置绑定。
func (s *AlertService) SetLogSearch(ls *logplatform.LogSearchService) {
	if s == nil {
		return
	}
	s.logSearch = ls
}

// AlertServiceOptions 可选依赖：静默、处理人、内置规则评估。
type AlertServiceOptions struct {
	SilenceSvc     *AlertSilenceService
	MaintenanceSvc *AlertMaintenanceService
	AssigneeSvc    *AlertRuleAssigneeService
	DutySvc        *AlertDutyService
	// ReceiverGroupCache 与 AlertReceiverGroupService 共用，避免 CRUD 失效与投递缓存不一致。
	ReceiverGroupCache *ReceiverGroupCache
	// EncryptionKey 与项目/云账号凭据加密一致；非空时用于云到期规则解密云账号 AK/SK。
	EncryptionKey        string
	EventRepo            interfaces.AlertEventRepository
	ChannelRepo          interfaces.AlertChannelRepository
	MonitorRuleRepo      interfaces.AlertMonitorRuleRepository
	DatasourceRepo       interfaces.AlertDatasourceRepository
	ProjectRepo          interfaces.ProjectRepository
	FiringDeliveryRepo   interfaces.AlertFiringDeliveryRepository
	CloudExpiryRepo      interfaces.CloudExpiryRuleRepository
	CloudAccountRepo     interfaces.CloudAccountRepository
	StateSvc             AlertStateService
	SubscriptionRepo     interfaces.AlertSubscriptionRepository
	InhibitionRuleRepo   interfaces.AlertInhibitionRuleRepository
	AckRepo              interfaces.AlertAckRepository
	ProgressNoteRepo     interfaces.AlertProgressNoteRepository
	CurHisRepo           interfaces.AlertCurHisRepository
	PromqlSavedQueryRepo interfaces.PromqlSavedQueryRepository
	ChangeEventRepo      interfaces.ChangeEventRepository
	DictEntryRepo        interfaces.DictEntryRepository
	ReceiverGroupRepo    interfaces.AlertReceiverGroupRepository
	LogSearch            *logplatform.LogSearchService
	MemberRepo           interfaces.ProjectMemberRepository
}

type promEnrichTask struct {
	Fingerprint  string
	GeneratorURL string
}

// NewAlertService 创建相关逻辑。依赖经 opts 注入仓库，不再持有 *gorm.DB。
func NewAlertService(redisClient *redis.Client, sender mailer.Sender, cfg config.AlertConfig, opts *AlertServiceOptions) *AlertService {
	if opts == nil {
		panic("alert: AlertServiceOptions is required")
	}
	cfg.ApplyDefaults()

	receiverCache := opts.ReceiverGroupCache
	if receiverCache == nil && opts.ReceiverGroupRepo != nil {
		receiverCache = NewReceiverGroupCache(opts.ReceiverGroupRepo)
	}

	svc := &AlertService{
		redis:                redisClient,
		mailer:               sender,
		cfg:                  cfg,
		cloudExpiryState:     make(map[string]bool),
		inhibitionSvc:        NewAlertInhibitionServiceWithRepo(opts.InhibitionRuleRepo, redisClient),
		subscriptionSvc:      NewAlertSubscriptionService(opts.SubscriptionRepo),
		receiverGroupCache:   receiverCache,
		metrics:              NewAlertMetrics(),
		timingLeaderToken:    fmt.Sprintf("%d-%d", os.Getpid(), time.Now().UnixNano()),
		silenceSvc:           opts.SilenceSvc,
		maintenanceSvc:       opts.MaintenanceSvc,
		assigneeSvc:          opts.AssigneeSvc,
		dutySvc:              opts.DutySvc,
		eventRepo:            opts.EventRepo,
		channelRepo:          opts.ChannelRepo,
		monitorRuleRepo:      opts.MonitorRuleRepo,
		datasourceRepo:       opts.DatasourceRepo,
		projectRepo:          opts.ProjectRepo,
		firingDeliveryRepo:   opts.FiringDeliveryRepo,
		cloudExpiryRepo:      opts.CloudExpiryRepo,
		cloudAccountRepo:     opts.CloudAccountRepo,
		ackRepo:              opts.AckRepo,
		progressNoteRepo:     opts.ProgressNoteRepo,
		curHisRepo:           opts.CurHisRepo,
		promqlSavedQueryRepo: opts.PromqlSavedQueryRepo,
		changeEventRepo:      opts.ChangeEventRepo,
		dictEntryRepo:        opts.DictEntryRepo,
		receiverGroupRepo:    opts.ReceiverGroupRepo,
		alertStateSvc:        opts.StateSvc,
		logSearch:            opts.LogSearch,
	}
	if svc.subscriptionSvc != nil {
		svc.subscriptionSvc.SetMemberRepo(opts.MemberRepo)
	}
	if opts.ReceiverGroupRepo != nil {
		svc.subscriptionSvc.AttachReceiverGroups(NewAlertReceiverGroupService(
			opts.ReceiverGroupRepo,
			receiverCache,
		))
	}

	// 初始化指标更新器并启动
	svc.metricsUpdater = NewAlertMetricsUpdater(svc.metrics, svc.inhibitionSvc)
	svc.metricsUpdater.Start()

	if key := strings.TrimSpace(opts.EncryptionKey); key != "" {
		if aead, err := cryptox.NewAESGCMFromKeyString(key); err == nil {
			svc.aead = aead
		}
	}
	return svc
}

// RunBackgroundWorkers 启动告警后台任务（由 alert 插件 StartWorkers 调用）。
func (s *AlertService) RunBackgroundWorkers(ctx context.Context) {
	if s == nil {
		return
	}
	s.startPrometheusEnrichWorkers(ctx)
	s.startInhibitionPruner(ctx)
	evalCtx, cancel := context.WithCancel(ctx)
	s.monitorEvalCancel = cancel
	lifecycle.Go("alert.monitor-rule-evaluator", func() { s.runMonitorRuleEvaluator(evalCtx) })
	lifecycle.Go("alert.cloud-expiry-evaluator", func() { s.runCloudExpiryEvaluator(evalCtx) })
	lifecycle.Go("alert.timing-worker", func() { s.runAlertTimingWorker(evalCtx) })
	s.runAlertWebhookIngestWorker(evalCtx)
}

func (s *AlertService) GetSubscriptionService() *AlertSubscriptionService {
	return s.subscriptionSvc
}

func (s *AlertService) GetInhibitionService() *AlertInhibitionService {
	return s.inhibitionSvc
}
