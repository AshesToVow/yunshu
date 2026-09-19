package router

import (
	"fmt"

	"yunshu/internal/plugin"
	"yunshu/internal/routedeps"

	"github.com/gin-gonic/gin"
)

// bindPluginRoutes 按插件名注册 HTTP；deps 由 Register 闭包注入（不经 plugin.Runtime）。
func bindPluginRoutes(name string, api *gin.RouterGroup, d routedeps.Bundle) error {
	if d == nil {
		return fmt.Errorf("route deps not set")
	}
	switch name {
	case "core":
		RegisterCoreRoutes(api, d)
	case "k8s":
		RegisterK8sRoutes(api, d)
	case "alert":
		RegisterAlertRoutes(api, d)
	case "project":
		RegisterProjectRoutes(api, d)
		RegisterLogPlatformRoutes(api, d)
	case "cmdb":
		RegisterCMDBRoutes(api, d)
	case "backup":
		RegisterBackupRoutes(api, d)
	case "cicd":
		RegisterCicdRoutes(api, d)
	case "dbmgmt":
		RegisterDbmgmtRoutes(api, d)
	case "inspect":
		RegisterInspectRoutes(api, d)
	case "ai":
		RegisterAIRoutes(api, d)
	case "esmgmt":
		RegisterEsmgmtRoutes(api, d)
	default:
		return fmt.Errorf("unknown plugin %q", name)
	}
	return nil
}

func installPluginRouteBinder(d routedeps.Bundle) {
	plugin.SetRouteBinder(func(name string, api *gin.RouterGroup, _ *plugin.Runtime) error {
		return bindPluginRoutes(name, api, d)
	})
}
