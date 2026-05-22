package router

import (
	"context"
	"fmt"

	"yunshu/internal/bootstrap"
	grpcclient "yunshu/internal/grpc/client"
	"yunshu/internal/handler"
	"yunshu/internal/pkg/logutil"
	"yunshu/internal/service/k8seventforward"
)

// Register wires dependencies via Wire and registers all HTTP routes.
// Returns the K8s event forward manager (may be nil). bgCtx is used for background workers.
func Register(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient, bgCtx context.Context) (*k8seventforward.Manager, error) {
	handler.SetLogger(app.Logger)
	registerSwagger(app)

	d, err := InitializeRouteDeps(app, runtimeClient)
	if err != nil {
		return nil, fmt.Errorf("initialize route deps: %w", err)
	}

	api := app.Engine.Group("/api/v1")
	registerPlatformRoutes(api, d)
	registerK8sRoutes(api, d)
	registerProjectRoutes(api, d)

	if d.mysqlBackupSvc != nil && bgCtx != nil {
		go d.mysqlBackupSvc.RunMysqlBackupScheduler(bgCtx)
	}

	mgr, err := k8seventforward.NewManager(
		app.DB,
		d.k8sRuntimeService,
		app.YamlK8sEventForwardBase,
		app.Config.Alert,
		app.Config.App.Port,
	)
	if err != nil {
		logutil.Worker("k8s.event_forward").Errorw(err, "Failed to init K8s event forward manager")
		return nil, nil
	}
	mgr.Start()
	return mgr, nil
}
