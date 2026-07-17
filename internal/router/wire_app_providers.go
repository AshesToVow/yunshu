package router

import (
	"yunshu/internal/bootstrap"
	"yunshu/internal/config"
	"yunshu/internal/plugin"
	"yunshu/internal/pkg/mailer"

	"github.com/casbin/casbin/v2"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type (
	AppDisplayName        string
	SecurityEncryptionKey string
)

type appRouteConfig struct {
	AppName       AppDisplayName
	EncryptionKey SecurityEncryptionKey
}

func provideDB(app *bootstrap.App) *gorm.DB {
	return app.DB
}

func provideRedis(app *bootstrap.App) *redis.Client {
	return app.Redis
}

func provideEnforcer(app *bootstrap.App) *casbin.SyncedEnforcer {
	return app.Enforcer
}

func provideMailer(app *bootstrap.App) mailer.Sender {
	return app.Mailer
}

func provideAuthConfig(app *bootstrap.App) config.AuthConfig {
	return app.Config.Auth
}

func provideAlertConfig(app *bootstrap.App) config.AlertConfig {
	return app.Config.Alert
}

func provideCicdConfig(app *bootstrap.App) config.CicdConfig {
	return app.Config.Cicd
}

func provideDbmgmtConfig(app *bootstrap.App) config.DbmgmtConfig {
	return app.Config.Dbmgmt
}

func provideAppRouteConfig(app *bootstrap.App) *appRouteConfig {
	return &appRouteConfig{
		AppName:       AppDisplayName(app.Config.App.Name),
		EncryptionKey: SecurityEncryptionKey(app.Config.Security.EncryptionKey),
	}
}

var appRouteConfigFields = wire.FieldsOf(
	new(*appRouteConfig),
	"AppName", "EncryptionKey",
)

func providePluginsEnabled(app *bootstrap.App) map[string]bool {
	return plugin.ResolveEnabled(&app.Config.Plugins)
}
