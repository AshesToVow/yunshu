package database

import (
	"gorm.io/gorm"
)

// PruneInvalidCasbinRules 删除 ptype 为空或不符合 Casbin 命名规则的脏数据，避免 LoadPolicy 时 panic。
func PruneInvalidCasbinRules(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	switch db.Dialector.Name() {
	case "postgres":
		return db.Exec(`DELETE FROM casbin_rule WHERE ptype IS NULL OR ptype = '' OR ptype !~ '^(p|g)[0-9]*$'`).Error
	default:
		return db.Exec(`DELETE FROM casbin_rule WHERE ptype IS NULL OR ptype = '' OR ptype NOT REGEXP '^(p|g)[0-9]*$'`).Error
	}
}
