package router

import (
	"fmt"

	"yunshu/internal/plugin"

	"github.com/gin-gonic/gin"
)

func init() {
	plugin.SetRouteBinder(bindPluginRoutes)
}

func bindPluginRoutes(name string, api *gin.RouterGroup, rt *plugin.Runtime) error {
	if rt == nil || rt.Deps == nil {
		return fmt.Errorf("route deps not set on plugin runtime")
	}
	d, ok := rt.Deps.(*RouteDeps)
	if !ok || d == nil {
		return fmt.Errorf("invalid route deps type")
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
