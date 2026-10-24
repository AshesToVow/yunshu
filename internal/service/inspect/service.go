package inspect

// 巡检服务装配：依赖注入、报告存储后端解析。
// 业务实现按关注点拆分在同包内：
//   - plan.go          巡检计划（inspect_plan）
//   - items.go         巡检项（inspect_item）与全局模板同步
//   - run.go           巡检执行（inspect_run）与报告生成
//   - notify.go        邮件通知
//   - report_access.go 报告读取、渲染入口与过期清理

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"yunshu/internal/interfaces"
	"yunshu/internal/pkg/mailer"
	"yunshu/internal/service/alert"
	"yunshu/internal/service/platformtpl"

	"github.com/redis/go-redis/v9"
)

// ReportStoreFactory 由装配层注入，避免 Service 直持 *gorm.DB 解析 MinIO。
type ReportStoreFactory func(ctx context.Context) ReportStore

// ReportStorageInfoFactory 由装配层注入，解析报告存储状态展示。
type ReportStorageInfoFactory func(ctx context.Context) ReportStorageInfo

type Service struct {
	repo            interfaces.InspectRepository
	platformTplRepo interfaces.PlatformTemplateRepository
	redis           *redis.Client
	dsSvc           *alert.AlertDatasourceService
	projects        interfaces.ProjectRepository
	mailer          mailer.Sender
	appName         string
	reportDir       string
	newReportStore  ReportStoreFactory
	storageInfo     ReportStorageInfoFactory

	workerOnce sync.Once
	workerCtx  context.Context
	jobCh      chan uint
}

func NewService(
	repo interfaces.InspectRepository,
	platformTplRepo interfaces.PlatformTemplateRepository,
	redisClient *redis.Client,
	dsSvc *alert.AlertDatasourceService,
	projects interfaces.ProjectRepository,
	sender mailer.Sender,
	appName string,
	newReportStore ReportStoreFactory,
	storageInfo ReportStorageInfoFactory,
) *Service {
	dir := filepath.Join("logs", "inspect-reports")
	_ = os.MkdirAll(dir, 0o755)
	return &Service{
		repo:            repo,
		platformTplRepo: platformTplRepo,
		redis:           redisClient,
		dsSvc:           dsSvc,
		projects:        projects,
		mailer:          sender,
		appName:         strings.TrimSpace(appName),
		reportDir:       dir,
		newReportStore:  newReportStore,
		storageInfo:     storageInfo,
	}
}

func (s *Service) reportStore(ctx context.Context) ReportStore {
	if s != nil && s.newReportStore != nil {
		return s.newReportStore(ctx)
	}
	return newLocalReportStore(s.reportDir)
}

// store 与 reportStore 同义，保留短名供报告读写路径使用。
func (s *Service) store(ctx context.Context) ReportStore {
	return s.reportStore(ctx)
}

func (s *Service) ReportStorageInfo(ctx context.Context) ReportStorageInfo {
	if s != nil && s.storageInfo != nil {
		return s.storageInfo(ctx)
	}
	return ReportStorageInfo{Backend: StorageLocal, LocalRoot: s.reportDir, MinioReason: "未配置"}
}

func (s *Service) platformTpl() *platformtpl.Service {
	return platformtpl.NewService(s.platformTplRepo, nil)
}

func (s *Service) appNameOrDefault() string {
	if s != nil && s.appName != "" {
		return s.appName
	}
	return "yunshu"
}
