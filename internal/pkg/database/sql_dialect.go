package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// DialectName returns gorm dialector name or empty when db is nil.
func DialectName(db *gorm.DB) string {
	if db == nil {
		return ""
	}
	return db.Dialector.Name()
}

// SQLDeleteDictDuplicatesByLabel removes duplicate dict_entries rows (keep smallest id).
func SQLDeleteDictDuplicatesByLabel(dialect string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres":
		return `
DELETE FROM dict_entries d1
USING dict_entries d2
WHERE d1.dict_type = d2.dict_type
  AND TRIM(d1.label) = TRIM(d2.label)
  AND d1.id > d2.id
  AND d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL`
	default:
		return `
DELETE d1 FROM dict_entries d1
JOIN dict_entries d2 ON d1.dict_type = d2.dict_type AND TRIM(d1.label) = TRIM(d2.label) AND d1.id > d2.id
WHERE d1.deleted_at IS NULL AND d2.deleted_at IS NULL`
	}
}

// SQLDeleteDictDuplicatesByValue removes duplicate dict_entries rows (keep smallest id).
func SQLDeleteDictDuplicatesByValue(dialect string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres":
		return `
DELETE FROM dict_entries d1
USING dict_entries d2
WHERE d1.dict_type = d2.dict_type
  AND TRIM(d1.value) = TRIM(d2.value)
  AND d1.id > d2.id
  AND d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL`
	default:
		return `
DELETE d1 FROM dict_entries d1
JOIN dict_entries d2 ON d1.dict_type = d2.dict_type AND TRIM(d1.value) = TRIM(d2.value) AND d1.id > d2.id
WHERE d1.deleted_at IS NULL AND d2.deleted_at IS NULL`
	}
}

// SQLDropDictEntriesLegacyCompositeIndex drops legacy MySQL/PG index when present.
func SQLDropDictEntriesLegacyCompositeIndex(dialect string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres":
		return `DROP INDEX IF EXISTS idx_dict_type_value_deleted`
	default:
		return "ALTER TABLE `dict_entries` DROP INDEX `idx_dict_type_value_deleted`"
	}
}

// SQLDropIndexIfExists 生成删除指定表索引的 DDL（存在才删），用于软删除唯一索引重建前的清理。
func SQLDropIndexIfExists(dialect, table, index string) string {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres":
		return fmt.Sprintf(`DROP INDEX IF EXISTS %s`, index)
	default:
		return fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", table, index)
	}
}

// SQLCreateAgentDiscoveryUniqueIndex creates dialect-specific unique index on agent_discoveries.
func SQLCreateAgentDiscoveryUniqueIndex(dialect string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres":
		return `CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_discovery_unique ON agent_discoveries (project_id, server_id, kind, (left(value, 512)))`, nil
	case "mysql":
		return `CREATE UNIQUE INDEX idx_agent_discovery_unique ON agent_discoveries (project_id, server_id, kind, value(512))`, nil
	default:
		return "", fmt.Errorf("unsupported dialect for agent discovery unique index: %q", dialect)
	}
}

// IsDuplicateIndexError reports whether err is a benign duplicate-index error during migration.
func IsDuplicateIndexError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "42p07")
}
