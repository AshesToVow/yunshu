//go:build wireinject
// +build wireinject

package router

import (
	"yunshu/internal/bootstrap"
	grpcclient "yunshu/internal/grpc/client"

	"github.com/google/wire"
)

//go:generate go run -mod=mod github.com/google/wire/cmd/wire

func provideRouteDeps(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient, repos *routeRepositories, svcs *routeServices) (*RouteDeps, error) {
	return assembleRouteDeps(app, runtimeClient, repos, svcs)
}

// InitializeRouteDeps is the Wire entry for HTTP route dependencies.
func InitializeRouteDeps(app *bootstrap.App, runtimeClient *grpcclient.RuntimeClient) (*RouteDeps, error) {
	wire.Build(
		AppInfraSet,
		RepositorySet,
		ServiceSet,
		provideRouteDeps,
	)
	return nil, nil
}
