package router

import (
	"strings"

	swaggerDocs "yunshu/docs/swagger"
	"yunshu/internal/bootstrap"
	"yunshu/internal/middleware"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/response"
	"yunshu/internal/repository"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func registerSwagger(app *bootstrap.App) {
	cfg := app.Config.Swagger
	if !cfg.Enabled {
		return
	}

	basePath := normalizeSwaggerPath(cfg.Path)
	swaggerDocs.SwaggerInfo.Title = "YunShu CMDB API"
	swaggerDocs.SwaggerInfo.Description = "YunShu CMDB is an operations CMDB console built with Gin, MySQL, Redis, SMTP mail delivery, Casbin, Cobra and slog."
	swaggerDocs.SwaggerInfo.Version = "1.0"
	swaggerDocs.SwaggerInfo.BasePath = "/"

	// 接口文档暴露全部路由与字段，必须登录且仅超管可见；启用 auth 后浏览器携带 ys_at Cookie 即可访问。
	authGuard := middleware.Auth(app.Config.Auth.JWTSecret, app.Redis, repository.NewUserRepository(app.DB), app.Logger, app.DB)
	app.Engine.GET(basePath+"/*any", authGuard, swaggerSuperAdminOnly, ginSwagger.WrapHandler(
		swaggerfiles.Handler,
		ginSwagger.DeepLinking(true),
	))
}

func swaggerSuperAdminOnly(c *gin.Context) {
	user, ok := auth.CurrentUserFromContext(c)
	if !ok || user == nil || !auth.IsSuperAdminRole(user.RoleCodes) {
		response.Error(c, constants.ErrForbiddenWithMsg("仅超级管理员可访问接口文档"))
		c.Abort()
		return
	}
	c.Next()
}

func normalizeSwaggerPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/swagger"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" {
		path = strings.TrimRight(path, "/")
	}
	return path
}
