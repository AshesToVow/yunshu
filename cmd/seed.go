package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"yunshu/internal/bootstrap"
	"yunshu/internal/config"
	"yunshu/internal/menu"
	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/pkg/password"
	"yunshu/internal/plugingate"
	"yunshu/internal/service"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func init() {
	rootCmd.AddCommand(seedCmd)
}

var seedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Seed default admin user, roles and permissions",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := bootstrap.BuildCoreApp(configPath)
		if err != nil {
			return err
		}
		defer app.Close()

		logx.Init(app.Logger)

		ctx := context.Background()
		// 确保新增字段（如 permissions.k8s_scope_enabled）在 seed 前已完成迁移
		if err := bootstrap.AutoMigrateModels(app.DB, &app.Config.Plugins); err != nil {
			return err
		}

		// 历史错误：撤销策略实际路由为 DELETE /api/v1/policies（JSON body），与 /api/v1/policies/:id 不匹配，会导致无法撤销授权
		if err := service.RemovePermissionPolicies(app.Enforcer, "/api/v1/policies/:id", "DELETE"); err != nil {
			return err
		}
		removedStale, err := removeStalePermissions(ctx, app.Enforcer, app.DB)
		if err != nil {
			return fmt.Errorf("remove stale permissions: %w", err)
		}
		if removedStale > 0 {
			slog.Default().With("component", "seed").Info("removed stale permissions", "count", removedStale)
			fmt.Printf("removed %d stale permission records\n", removedStale)
		}
		removedWild, err := removeWildcardPermissions(ctx, app.Enforcer, app.DB)
		if err != nil {
			return fmt.Errorf("remove wildcard permissions: %w", err)
		}
		if removedWild > 0 {
			slog.Default().With("component", "seed").Info("removed wildcard permissions", "count", removedWild)
			fmt.Printf("removed %d wildcard permission records\n", removedWild)
		}

		permissions := defaultPermissions()
		adminRole := model.Role{
			Name:        "Super Admin",
			Code:        "super-admin",
			Description: "Built-in administrator role with full access.",
			Status:      model.StatusEnabled,
		}
		adminEmail := "rootwxd@163.com"
		adminUser := model.User{
			Username: "admin",
			Email:    &adminEmail,
			Nickname: "System Admin",
			Status:   model.StatusEnabled,
		}
		var adminCreated bool

		err = app.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("resource = ? AND action = ?", "/api/v1/policies/:id", "DELETE").
				Delete(&model.Permission{}).Error; err != nil {
				return err
			}
			if err := seedPermissions(ctx, tx, permissions); err != nil {
				return err
			}
			if err := upsertByKey(ctx, tx, &adminRole,
				func(db *gorm.DB) *gorm.DB { return db.Where("code = ?", adminRole.Code) },
				func(existing, incoming *model.Role) {
					existing.Name = incoming.Name
					existing.Description = incoming.Description
					existing.Status = incoming.Status
				},
				nil,
			); err != nil {
				return err
			}
			if err := upsertByKey(ctx, tx, &adminUser,
				func(db *gorm.DB) *gorm.DB { return db.Where("username = ?", adminUser.Username) },
				func(existing, incoming *model.User) {
					existing.Email = incoming.Email
					existing.Nickname = incoming.Nickname
					existing.Status = incoming.Status
				},
				func(incoming *model.User) error {
					adminCreated = true
					hashed, err := password.Hash("Admin@123")
					if err != nil {
						return err
					}
					incoming.Password = hashed
					now := time.Now()
					incoming.PasswordChangedAt = &now
					return nil
				},
			); err != nil {
				return err
			}
			if err := tx.Model(&adminUser).Association("Roles").Replace([]model.Role{adminRole}); err != nil {
				return err
			}
			return seedMenus(ctx, tx, &app.Config.Plugins)
		})
		if err != nil {
			return err
		}

		if err := service.AddRolePolicies(app.Enforcer, adminRole.Code, permissions); err != nil {
			return err
		}
		if err := service.SyncUserRoles(app.Enforcer, adminUser.ID, []model.Role{adminRole}); err != nil {
			return err
		}

		if adminCreated {
			slog.Default().With("component", "seed").Info("Seed completed", "username", adminUser.Username, "email", adminUser.Email, "password", "Admin@123")
			fmt.Println("seed completed: created admin user admin / Admin@123")
		} else {
			slog.Default().With("component", "seed").Info("Seed completed", "username", adminUser.Username, "email", adminUser.Email)
			fmt.Println("seed completed")
		}
		return nil
	},
}

func seedPermissions(ctx context.Context, db *gorm.DB, permissions []model.Permission) error {
	normalized := make([]model.Permission, 0, len(permissions))
	seen := make(map[string]struct{}, len(permissions))
	for _, p := range permissions {
		p.Resource = strings.TrimSpace(p.Resource)
		p.Action = strings.ToUpper(strings.TrimSpace(p.Action))
		if p.Resource == "" || p.Action == "" {
			continue
		}
		key := p.Resource + "::" + p.Action
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if constants.HasPermissionResourceWildcard(p.Resource) {
			continue
		}
		normalized = append(normalized, p)
	}
	return db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "resource"}, {Name: "action"}},
		// 不覆盖 k8s_scope_enabled / description：避免 seed 冲掉运营开关与 [k8s-scope=off] 标记。
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "updated_at", "deleted_at",
		}),
	}).CreateInBatches(normalized, 200).Error
}

func defaultPermissions() []model.Permission {
	out := make([]model.Permission, 0, 700)
	out = append(out, seedPermissionsSystem()...)
	out = append(out, seedPermissionsK8s()...)
	out = append(out, seedPermissionsAlert()...)
	out = append(out, seedPermissionsProject()...)
	out = append(out, seedPermissionsCicd()...)
	out = append(out, seedPermissionsDbmgmt()...)
	out = append(out, seedPermissionsAI()...)
	out = append(out, seedPermissionsLog()...)
	out = append(out, seedPermissionsInspect()...)
	return out
}

func seedMenus(ctx context.Context, db *gorm.DB, cfg *config.PluginsConfig) error {
	if err := menu.Sync(ctx, db); err != nil {
		return err
	}
	return plugingate.SyncMenuVisibility(ctx, db, cfg)
}
