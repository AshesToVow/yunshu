package router

import (
	"context"
	"fmt"

	"yunshu/internal/bootstrap"
	grpcclient "yunshu/internal/grpc/client"
	"yunshu/internal/handler"
	"yunshu/internal/plugin"
	"yunshu/internal/service/k8s/eventforward"

	_ "yunshu/internal/plugins/all" // 注册内置业务插件
)

// Register 装配依赖并按配置加载各业务插件的路由与后台任务。
// 返回 K8s 事件转发管理器（k8s 插件未启用或未初始化时为 nil）。
func Register(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient, bgCtx context.Context) (*eventforward.Manager, error) {
	handler.SetLogger(app.Logger)
	registerSwagger(app)

	d, err := InitializeRouteDeps(app, runtimeClient)
	if err != nil {
		return nil, fmt.Errorf("initialize route deps: %w", err)
	}

	rt := &plugin.Runtime{
		DB:                      app.DB,
		Config:                  app.Config,
		YamlK8sEventForwardBase: app.YamlK8sEventForwardBase,
		GRPCClient:              runtimeClient,
		Deps:                    d,
		Enabled:                 plugin.ResolveEnabled(&app.Config.Plugins),
		K8sRuntime:              d.K8sRuntimeService(),
		MysqlBackup:             d.MysqlBackupService(),
		Cicd:                    d.CicdService(),
		Alert:                   d.AlertService(),
	}

	api := app.Engine.Group("/api/v1")
	if err := plugin.RegisterRoutes(api, rt, &app.Config.Plugins); err != nil {
		return nil, fmt.Errorf("register plugin routes: %w", err)
	}
	if err := plugin.StartWorkers(bgCtx, rt, &app.Config.Plugins); err != nil {
		return nil, fmt.Errorf("start plugin workers: %w", err)
	}
	return eventforward.Active(), nil
}
