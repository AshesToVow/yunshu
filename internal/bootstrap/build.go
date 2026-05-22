package bootstrap

import "yunshu/internal/providers"

func BuildServerApp(configPath string) (*App, error) {
	infra, err := providers.InitializeInfra(configPath)
	if err != nil {
		return nil, err
	}
	return NewBuilder().
		WithInfra(infra).
		WithDictOverrides().
		WithCasbin().
		WithMailer().
		WithGin().
		Build()
}

func BuildCoreApp(configPath string) (*App, error) {
	infra, err := providers.InitializeCore(configPath)
	if err != nil {
		return nil, err
	}
	return NewBuilder().
		WithInfra(infra).
		WithDictOverrides().
		WithCasbin().
		Build()
}
