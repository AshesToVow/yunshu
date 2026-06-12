//go:build wireinject
// +build wireinject

package providers

import (
	"github.com/google/wire"
)

//go:generate go run -mod=mod github.com/google/wire/cmd/wire

// InitializeCore wires config, logger, and MySQL (no Redis; for seed/migrate).
func InitializeCore(configPath string) (*Infra, error) {
	wire.Build(
		NewConfig,
		NewLogger,
		NewDB,
		wire.Struct(new(Infra), "Config", "Logger", "DB"),
	)
	return nil, nil
}

// InitializeInfra wires full infrastructure including Redis (for server).
func InitializeInfra(configPath string) (*Infra, error) {
	wire.Build(
		NewConfig,
		NewLogger,
		NewDB,
		NewRedis,
		wire.Struct(new(Infra), "Config", "Logger", "DB", "Redis"),
	)
	return nil, nil
}
