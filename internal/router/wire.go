//go:build wireinject
// +build wireinject

package router

import (
	"yunshu/internal/bootstrap"
	grpcclient "yunshu/internal/grpc/client"

	"github.com/google/wire"
	"gorm.io/gorm"
)

//go:generate go run -mod=mod github.com/google/wire/cmd/wire

func provideDB(app *bootstrap.App) *gorm.DB {
	return app.DB
}

func provideRouteRepositories(db *gorm.DB) *routeRepositories {
	return newRouteRepositories(db)
}

func provideRouteDeps(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient, repos *routeRepositories, svcs *routeServices) (*routeDeps, error) {
	return assembleRouteDeps(app, runtimeClient, repos, svcs)
}

// InitializeRouteDeps is the Wire entry for HTTP route dependencies.
func InitializeRouteDeps(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient) (*routeDeps, error) {
	wire.Build(
		provideDB,
		provideRouteRepositories,
		provideRouteServices,
		provideRouteDeps,
	)
	return nil, nil
}
