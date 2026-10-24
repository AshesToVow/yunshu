package ai

import (
	"yunshu/internal/pkg/database"

	"gorm.io/gorm"
)

// PostMigrate 将 AI 相关表转为 utf8mb4，避免 emoji/4 字节 UTF-8 写入 mediumtext 失败（MySQL 1366）。
func (m *module) PostMigrate(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if database.DialectName(db) != "mysql" {
		return nil
	}
	tables := []string{
		"ai_investigations",
		"ai_chat_sessions",
		"ai_chat_messages",
		"ai_prompt_versions",
		"ai_kb_chunks",
		"ai_kb_documents",
		"ai_incident_cases",
		"ai_sops",
		"ai_eval_results",
		"ai_audit_events",
	}
	for _, table := range tables {
		if !db.Migrator().HasTable(table) {
			continue
		}
		sql := "ALTER TABLE `" + table + "` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}
