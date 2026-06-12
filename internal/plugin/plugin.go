// Package plugin 提供 GVA 风格的业务模块插件契约：注册、迁移与后台任务。
package plugin

import (
	"context"

	"gorm.io/gorm"
)

// Module 业务插件接口（编译期注册，配置控制启停）。
type Module interface {
	Name() string
	Description() string
	Models() []any
	PreMigrate(db *gorm.DB) error
	PostMigrate(db *gorm.DB) error
	StartWorkers(bgCtx context.Context, rt *Runtime) error
}

// Base 可选嵌入，减少空方法样板。
type Base struct{}

func (Base) Description() string                   { return "" }
func (Base) Models() []any                         { return nil }
func (Base) PreMigrate(*gorm.DB) error              { return nil }
func (Base) PostMigrate(*gorm.DB) error             { return nil }
func (Base) StartWorkers(context.Context, *Runtime) error { return nil }
