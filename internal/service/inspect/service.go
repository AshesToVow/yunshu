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

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	redis     *redis.Client
	dsSvc     *alert.AlertDatasourceService
	projects  interfaces.ProjectRepository
	mailer    mailer.Sender
	appName   string
	reportDir string

	workerOnce sync.Once
	workerCtx  context.Context
	jobCh      chan uint
}

func NewService(
	db *gorm.DB,
	redisClient *redis.Client,
	dsSvc *alert.AlertDatasourceService,
	projects interfaces.ProjectRepository,
	sender mailer.Sender,
	appName string,
) *Service {
	dir := filepath.Join("logs", "inspect-reports")
	_ = os.MkdirAll(dir, 0o755)
	return &Service{
		db:        db,
		redis:     redisClient,
		dsSvc:     dsSvc,
		projects:  projects,
		mailer:    sender,
		appName:   strings.TrimSpace(appName),
		reportDir: dir,
	}
}

func (s *Service) store(ctx context.Context) ReportStore {
	return resolveReportStore(ctx, s.db, s.reportDir)
}

// ReportStorageInfo 返回当前巡检报告存储后端与 MinIO 就绪状态。
func (s *Service) ReportStorageInfo(ctx context.Context) ReportStorageInfo {
	if s == nil {
		return ReportStorageInfo{Backend: StorageLocal}
	}
	return resolveReportStorageInfo(ctx, s.db, s.reportDir)
}

func (s *Service) appNameOrDefault() string {
	if s.appName != "" {
		return s.appName
	}
	return "yunshu"
}
