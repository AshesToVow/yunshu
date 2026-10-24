package kafkamgmt

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

func (m *module) Name() string { return "kafkamgmt" }
func (m *module) Description() string {
	return "Kafka 管理控制台：连接管理、Topic、Broker、消费组与抽样生产/消费"
}

func (m *module) Manifest() plugin.Manifest {
	return plugin.Manifest{
		MenuPathPrefixes: []string{"/kafkamgmt"},
		APIPrefixes:      []string{"/api/v1/kafkamgmt"},
	}
}

func (m *module) Models() []any {
	return []any{
		&model.KafkamgmtConnection{},
	}
}
