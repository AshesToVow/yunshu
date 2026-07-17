//go:build wireinject
// +build wireinject

package router

import (
	"yunshu/internal/bootstrap"

	"github.com/google/wire"
)

//go:generate go run -mod=mod github.com/google/wire/cmd/wire

func provideRouteDeps(app *bootstrap.App, repos *routeRepositories, svcs *routeServices) (*RouteDeps, error) {
	return assembleRouteDeps(app, repos, svcs)
}

// InitializeRouteDeps is the Wire entry for HTTP route dependencies.
func InitializeRouteDeps(app *bootstrap.App) (*RouteDeps, error) {
	wire.Build(
		AppInfraSet,
		RepositorySet,
		ServiceSet,
		provideRouteDeps,
	)
	return nil, nil
}
