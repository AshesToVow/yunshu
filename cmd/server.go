package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"log/slog"
	"yunshu/internal/bootstrap"
	"yunshu/internal/handler"
	"yunshu/internal/model"

	"yunshu/internal/pkg/lifecycle"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/password"
	"yunshu/internal/repository"
	"yunshu/internal/router"
	"yunshu/internal/service"

	"github.com/casbin/casbin/v2"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start yunshu platform server",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := bootstrap.BuildServerApp(configPath)
		if err != nil {
			return err
		}
		defer app.Close()

		logx.Init(app.Logger)
		handler.SetLogger(app.Logger)

		bootLog := slog.Default().With("component", "bootstrap")

		// 表结构变更闸门：生产环境默认不在服务进程内改表。
		// 滚动发布期多副本会并发执行 DDL，且 AutoMigrate 无法回滚、
		// 不会删除废弃列/索引，线上应走独立的 `yunshu migrate` 命令。
		if app.Config.AutoMigrateEnabled() {
			if err := bootstrap.AutoMigrateModels(app.DB, &app.Config.Plugins); err != nil {
				return fmt.Errorf("auto migrate: %w", err)
			}
			bootLog.Info("Database schema migrated")
		} else {
			bootLog.Info("AutoMigrate disabled; skipping schema migration",
				"env", app.Config.App.Env,
				"hint", "run `yunshu migrate` explicitly, or set database.auto_migrate=true to override")
		}
		if err := app.Enforcer.LoadPolicy(); err != nil {
			return fmt.Errorf("reload casbin policy: %w", err)
		}

		ctx := context.Background()
		if err := initReadonlyDemoUser(ctx, app.DB, app.Enforcer, bootLog); err != nil {
			bootLog.Error("Failed to init readonly demo user", "error", err)
		}

		bgWorkersCtx, bgWorkersCancel := context.WithCancel(context.Background())
		defer bgWorkersCancel()

		k8sEventForwardMgr, err := router.Register(app, bgWorkersCtx)
		if err != nil {
			return fmt.Errorf("register http routes: %w", err)
		}
		if k8sEventForwardMgr != nil {
			defer k8sEventForwardMgr.Stop()
		}

		server := &http.Server{
			Addr:              fmt.Sprintf(":%d", app.Config.App.Port),
			Handler:           app.Engine,
			ReadHeaderTimeout: time.Duration(app.Config.HTTP.ReadHeaderTimeoutSeconds) * time.Second,
			ReadTimeout:       time.Duration(app.Config.HTTP.ReadTimeoutSeconds) * time.Second,
			WriteTimeout:      time.Duration(app.Config.HTTP.WriteTimeoutSeconds) * time.Second,
			IdleTimeout:       time.Duration(app.Config.HTTP.IdleTimeoutSeconds) * time.Second,
		}

		errCh := make(chan error, 1)
		go func() {
			slog.Default().With("component", "server").Info("HTTP server started", "addr", server.Addr)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- err
			}
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-stop:
			slog.Default().With("component", "server").Info("Received shutdown signal", "signal", sig.String())
		case err := <-errCh:
			return err
		}

		httpShutdown := time.Duration(app.Config.HTTP.ShutdownTimeoutSeconds) * time.Second
		if httpShutdown <= 0 {
			httpShutdown = 10 * time.Second
		}
		defer logx.Sync()

		shutdownLog := slog.Default().With("component", "server")

		// 关闭顺序：先停止接收新请求（HTTP），再通知后台 worker 退出并等待收敛。
		// 反过来会让仍在处理中的请求访问到已停止的后台组件。
		ctxHTTP, cancelHTTP := context.WithTimeout(context.Background(), httpShutdown)
		defer cancelHTTP()
		httpErr := server.Shutdown(ctxHTTP)
		if httpErr != nil {
			shutdownLog.Error("HTTP server shutdown error", "error", httpErr)
		} else {
			shutdownLog.Info("HTTP server stopped")
		}

		// 显式停止 K8s 事件转发管理器（原先挂在 defer，无法保证在 worker 等待之前完成）。
		if k8sEventForwardMgr != nil {
			k8sEventForwardMgr.Stop()
		}

		// 通知所有后台 worker 退出，并带超时等待其收敛，
		// 避免 Kafka 消费位点、告警投递、备份任务在写入中途被强杀。
		bgWorkersCancel()
		lifecycle.WaitAndLog(shutdownLog, httpShutdown)

		return httpErr
	},
}

// initReadonlyDemoUser 初始化只读演示账号 viewer/viewer123。
func initReadonlyDemoUser(ctx context.Context, db *gorm.DB, enforcer *casbin.SyncedEnforcer, logger *slog.Logger) error {
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	permRepo := repository.NewPermissionRepository(db)

	roleCode := "viewer"
	roleName := "只读用户"
	var role *model.Role
	allRoles, err := roleRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list roles: %w", err)
	}
	for i := range allRoles {
		if allRoles[i].Code == roleCode {
			role = &allRoles[i]
			break
		}
	}

	if role == nil {
		role = &model.Role{
			Code:        roleCode,
			Name:        roleName,
			Description: "只读演示账号，仅可查看",
		}
		if err := db.Create(role).Error; err != nil {
			return fmt.Errorf("create role: %w", err)
		}
		logger.Info("Created readonly role", "code", roleCode)
	}

	if _, err := enforcer.RemoveFilteredPolicy(0, roleCode); err != nil {
		logger.Warn("Failed to remove old Casbin policies", "error", err)
	}

	perms, err := permRepo.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list permissions: %w", err)
	}

	added := 0
	for _, p := range perms {
		if p.Action != "GET" {
			continue
		}
		obj := p.Resource
		if obj == "" {
			continue
		}
		if _, err := enforcer.AddPolicy(roleCode, obj, "GET"); err != nil {
			logger.Warn("Failed to add Casbin policy", "resource", obj, "error", err)
			continue
		}
		added++
	}

	accessRepo := repository.NewK8sClusterAccessRepository(db)
	if err := accessRepo.Upsert(ctx, &model.K8sClusterAccessGrant{
		PrincipalKind: model.K8sPrincipalRole,
		PrincipalRef:  roleCode,
		ClusterID:     0,
		Preset:        "readonly",
	}); err != nil {
		return fmt.Errorf("upsert k8s cluster access grant: %w", err)
	}

	logger.Info("Configured readonly role permissions", "role", roleCode, "policies_added", added)

	username := "viewer"
	email := "viewer@yunshu.demo"
	plainPassword := "viewer123"

	user, err := userRepo.GetByUsername(ctx, username)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("get user: %w", err)
		}
		hashedPassword, err := password.Hash(plainPassword)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}

		user = &model.User{
			Username: username,
			Email:    &email,
			Password: hashedPassword,
			Status:   1,
		}
		if err := db.Create(user).Error; err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		logger.Info("Created demo user", "username", username)
	} else {
		logger.Info("Demo user already exists", "username", username)
	}

	if err := userRepo.ReplaceRoles(ctx, user, []model.Role{*role}); err != nil {
		return fmt.Errorf("bind role to user: %w", err)
	}

	if err := service.SyncUserRoles(enforcer, user.ID, []model.Role{*role}); err != nil {
		return fmt.Errorf("sync user roles: %w", err)
	}

	logger.Info("Initialized demo user", "username", username, "password", plainPassword, "role", roleCode)
	return nil
}
