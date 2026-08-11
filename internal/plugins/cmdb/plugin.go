package cmdb

import (
	"yunshu/internal/model"
	"yunshu/internal/plugin"
)

func init() {
	plugin.Register(&module{})
}

type module struct {
	plugin.Base
}

func (m *module) Name() string        { return "cmdb" }
func (m *module) Description() string { return "CMDB 服务器资产：主机、分组、云账号、SSH 与 Web 终端" }

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{"/project-servers", "/server-console"},
		APIPrefixes: []string{"/api/v1/cloud-accounts", "/api/v1/server-groups"},
		DependsOn:   []string{"project"},
	}
}

func (m *module) Models() []any {
	return []any{
		&model.ServerGroup{},
		&model.Server{},
		&model.ServerCredential{},
		&model.CloudAccount{},
		&model.ServerAccessGrant{},
	}
}
