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
	switch {
	case IsPostgres(dialect):
		return `
DELETE FROM dict_entries d1
USING dict_entries d2
WHERE d1.dict_type = d2.dict_type
  AND TRIM(d1.label) = TRIM(d2.label)
  AND d1.id > d2.id
  AND d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL`
	case IsDameng(dialect):
		return `
DELETE FROM dict_entries d1
WHERE d1.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM dict_entries d2
    WHERE d1.dict_type = d2.dict_type
      AND TRIM(d1.label) = TRIM(d2.label)
      AND d1.id > d2.id
      AND d2.deleted_at IS NULL
  )`
	default:
		return `
DELETE d1 FROM dict_entries d1
JOIN dict_entries d2 ON d1.dict_type = d2.dict_type AND TRIM(d1.label) = TRIM(d2.label) AND d1.id > d2.id
WHERE d1.deleted_at IS NULL AND d2.deleted_at IS NULL`
	}
}

// SQLDeleteDictDuplicatesByValue removes duplicate dict_entries rows (keep smallest id).
func SQLDeleteDictDuplicatesByValue(dialect string) string {
	switch {
	case IsPostgres(dialect):
		return `
DELETE FROM dict_entries d1
USING dict_entries d2
WHERE d1.dict_type = d2.dict_type
  AND TRIM(d1.value) = TRIM(d2.value)
  AND d1.id > d2.id
  AND d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL`
	case IsDameng(dialect):
		return `
DELETE FROM dict_entries d1
WHERE d1.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM dict_entries d2
    WHERE d1.dict_type = d2.dict_type
      AND TRIM(d1.value) = TRIM(d2.value)
      AND d1.id > d2.id
      AND d2.deleted_at IS NULL
  )`
	default:
		return `
DELETE d1 FROM dict_entries d1
JOIN dict_entries d2 ON d1.dict_type = d2.dict_type AND TRIM(d1.value) = TRIM(d2.value) AND d1.id > d2.id
WHERE d1.deleted_at IS NULL AND d2.deleted_at IS NULL`
	}
}

// SQLNormalizePermissionActionUpper 将 action 规范为大写，避免 GET/get 并存。
func SQLNormalizePermissionActionUpper(dialect string) string {
	switch {
	case IsPostgres(dialect), IsDameng(dialect):
		return `UPDATE permissions SET action = UPPER(action) WHERE action <> UPPER(action)`
	default:
		// MySQL 在 ci 校对下需 BINARY 才能发现大小写差异
		return `UPDATE permissions SET action = UPPER(action) WHERE BINARY action <> BINARY UPPER(action)`
	}
}

// SQLDeletePermissionActiveDuplicates 删除同 resource+action 的多余活跃行（保留最小 id）。
func SQLDeletePermissionActiveDuplicates(dialect string) string {
	switch {
	case IsPostgres(dialect):
		return `
DELETE FROM permissions d1
USING permissions d2
WHERE d1.resource = d2.resource
  AND UPPER(d1.action) = UPPER(d2.action)
  AND d1.id > d2.id
  AND d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL`
	case IsDameng(dialect):
		return `
DELETE FROM permissions d1
WHERE d1.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM permissions d2
    WHERE d1.resource = d2.resource
      AND UPPER(d1.action) = UPPER(d2.action)
      AND d1.id > d2.id
      AND d2.deleted_at IS NULL
  )`
	default:
		return `
DELETE d1 FROM permissions d1
JOIN permissions d2
  ON d1.resource = d2.resource AND UPPER(d1.action) = UPPER(d2.action) AND d1.id > d2.id
WHERE d1.deleted_at IS NULL AND d2.deleted_at IS NULL`
	}
}

// SQLDeletePermissionSoftDuplicatesWhenActive 删除已有活跃行时同键的软删除副本。
func SQLDeletePermissionSoftDuplicatesWhenActive(dialect string) string {
	switch {
	case IsPostgres(dialect):
		return `
DELETE FROM permissions p
USING permissions a
WHERE p.resource = a.resource
  AND UPPER(p.action) = UPPER(a.action)
  AND p.deleted_at IS NOT NULL
  AND a.deleted_at IS NULL`
	case IsDameng(dialect):
		return `
DELETE FROM permissions p
WHERE p.deleted_at IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM permissions a
    WHERE p.resource = a.resource
      AND UPPER(p.action) = UPPER(a.action)
      AND a.deleted_at IS NULL
  )`
	default:
		return `
DELETE p FROM permissions p
JOIN permissions a
  ON p.resource = a.resource AND UPPER(p.action) = UPPER(a.action) AND a.deleted_at IS NULL
WHERE p.deleted_at IS NOT NULL`
	}
}

// SQLDeletePermissionSoftOnlyDuplicates 仅软删除组内去重（保留最小 id）。
func SQLDeletePermissionSoftOnlyDuplicates(dialect string) string {
	switch {
	case IsPostgres(dialect):
		return `
DELETE FROM permissions d1
USING permissions d2
WHERE d1.resource = d2.resource
  AND UPPER(d1.action) = UPPER(d2.action)
  AND d1.id > d2.id
  AND d1.deleted_at IS NOT NULL
  AND d2.deleted_at IS NOT NULL`
	case IsDameng(dialect):
		return `
DELETE FROM permissions d1
WHERE d1.deleted_at IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM permissions d2
    WHERE d1.resource = d2.resource
      AND UPPER(d1.action) = UPPER(d2.action)
      AND d1.id > d2.id
      AND d2.deleted_at IS NOT NULL
  )`
	default:
		return `
DELETE d1 FROM permissions d1
JOIN permissions d2
  ON d1.resource = d2.resource AND UPPER(d1.action) = UPPER(d2.action) AND d1.id > d2.id
WHERE d1.deleted_at IS NOT NULL AND d2.deleted_at IS NOT NULL`
	}
}

// SQLDropDictEntriesLegacyCompositeIndex drops legacy MySQL/PG index when present.
func SQLDropDictEntriesLegacyCompositeIndex(dialect string) string {
	switch {
	case IsPostgres(dialect), IsDameng(dialect):
		return `DROP INDEX IF EXISTS idx_dict_type_value_deleted`
	default:
		return "ALTER TABLE `dict_entries` DROP INDEX `idx_dict_type_value_deleted`"
	}
}

// sanitizeSQLIdent 白名单过滤标识符：仅保留字母、数字、下划线与 $。
// 当前调用方传入的都是迁移代码内的常量，这里是纵深防御，
// 防止后续有人把外部输入接进来造成 DDL 注入。
func sanitizeSQLIdent(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c == '_', c == '$':
			b.WriteByte(c)
		}
	}
	return b.String()
}

// SQLDropIndexIfExists 生成删除指定表索引的 DDL（存在才删），用于软删除唯一索引重建前的清理。
func SQLDropIndexIfExists(dialect, table, index string) string {
	table = sanitizeSQLIdent(table)
	index = sanitizeSQLIdent(index)
	switch {
	case IsPostgres(dialect), IsDameng(dialect):
		return fmt.Sprintf(`DROP INDEX IF EXISTS "%s"`, index)
	default:
		return fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", table, index)
	}
}

// SQLCreateAgentDiscoveryUniqueIndex creates dialect-specific unique index on agent_discoveries.
func SQLCreateAgentDiscoveryUniqueIndex(dialect string) (string, error) {
	switch {
	case IsPostgres(dialect):
		return `CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_discovery_unique ON agent_discoveries (project_id, server_id, kind, (left(value, 512)))`, nil
	case IsDameng(dialect):
		return `CREATE UNIQUE INDEX idx_agent_discovery_unique ON agent_discoveries (project_id, server_id, kind, SUBSTR(value, 1, 512))`, nil
	case strings.EqualFold(strings.TrimSpace(dialect), "mysql"), strings.TrimSpace(dialect) == "":
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

// IsPostgres reports whether dialect is PostgreSQL.
func IsPostgres(dialect string) bool {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "postgres", "postgresql", "pg":
		return true
	default:
		return false
	}
}

// IsDameng reports whether dialect is 达梦（GORM dialector Name 为 dm）。
func IsDameng(dialect string) bool {
	switch strings.ToLower(strings.TrimSpace(dialect)) {
	case "dm", "dameng", "dm8":
		return true
	default:
		return false
	}
}

// SQLCSVContainsID CSV 列表（逗号分隔）是否包含整型 id 表达式。
// MySQL: FIND_IN_SET；PostgreSQL: string_to_array；达梦: INSTR + 拼接。
func SQLCSVContainsID(dialect, idExpr, csvCol string) string {
	idExpr = strings.TrimSpace(idExpr)
	csvCol = strings.TrimSpace(csvCol)
	switch {
	case IsPostgres(dialect):
		return fmt.Sprintf(`(%s)::text = ANY(string_to_array(%s, ','))`, idExpr, csvCol)
	case IsDameng(dialect):
		return fmt.Sprintf(`INSTR(',' || %s || ',', ',' || CAST(%s AS VARCHAR) || ',') > 0`, csvCol, idExpr)
	default:
		return fmt.Sprintf(`FIND_IN_SET(%s, %s)`, idExpr, csvCol)
	}
}

// SQLStaleTimestampBefore 生成「早于当前时间 interval」谓词（用于回收卡死 inflight）。
// mysqlInterval 示例："5 MINUTE"。
func SQLStaleTimestampBefore(dialect, column, mysqlInterval string) string {
	column = strings.TrimSpace(column)
	switch {
	case IsPostgres(dialect):
		return fmt.Sprintf("%s < NOW() - INTERVAL '%s'", column, mysqlIntervalToPostgres(mysqlInterval))
	case IsDameng(dialect):
		n, unit := splitMySQLInterval(mysqlInterval)
		return fmt.Sprintf("%s < DATEADD(%s, -%s, SYSDATE)", column, unit, n)
	default:
		return fmt.Sprintf("%s < DATE_SUB(NOW(), INTERVAL %s)", column, mysqlInterval)
	}
}

func mysqlIntervalToPostgres(mysqlInterval string) string {
	n, unit := splitMySQLInterval(mysqlInterval)
	switch unit {
	case "MINUTE":
		unit = "minutes"
	case "SECOND":
		unit = "seconds"
	case "HOUR":
		unit = "hours"
	case "DAY":
		unit = "days"
	default:
		unit = strings.ToLower(unit) + "s"
	}
	return n + " " + unit
}

func splitMySQLInterval(mysqlInterval string) (num, unit string) {
	fields := strings.Fields(strings.ToUpper(strings.TrimSpace(mysqlInterval)))
	if len(fields) != 2 {
		return "5", "MINUTE"
	}
	unit = fields[1]
	switch {
	case strings.HasPrefix(unit, "MINUTE"):
		unit = "MINUTE"
	case strings.HasPrefix(unit, "SECOND"):
		unit = "SECOND"
	case strings.HasPrefix(unit, "HOUR"):
		unit = "HOUR"
	case strings.HasPrefix(unit, "DAY"):
		unit = "DAY"
	}
	return fields[0], unit
}

// SQLOnConflictChanged 构造 upsert 时「目标列相对写入行是否变化」的布尔表达式。
// cols 为列名列表；MySQL 用 VALUES(col)，PostgreSQL 用 EXCLUDED.col，达梦 MERGE 用 excluded.col。
func SQLOnConflictChanged(dialect string, cols ...string) string {
	if len(cols) == 0 {
		return "FALSE"
	}
	parts := make([]string, 0, len(cols))
	switch {
	case IsPostgres(dialect):
		for _, c := range cols {
			c = sanitizeSQLIdent(c)
			parts = append(parts, fmt.Sprintf("%s IS DISTINCT FROM EXCLUDED.%s", c, c))
		}
	case IsDameng(dialect):
		for _, c := range cols {
			c = sanitizeSQLIdent(c)
			parts = append(parts, fmt.Sprintf(`"%s" <> "excluded"."%s"`, c, c))
		}
	default:
		for _, c := range cols {
			c = sanitizeSQLIdent(c)
			parts = append(parts, fmt.Sprintf("`%s` <> VALUES(`%s`)", c, c))
		}
	}
	return strings.Join(parts, " OR ")
}

// SQLBackfillAlertEventProjectFromDatasource 按数据源回填 alert_events.project_id。
func SQLBackfillAlertEventProjectFromDatasource(dialect string) string {
	switch {
	case IsPostgres(dialect):
		return `
UPDATE alert_events e
SET project_id = d.project_id
FROM alert_datasources d
WHERE e.datasource_id = d.id
  AND d.deleted_at IS NULL
  AND COALESCE(e.project_id, 0) = 0
  AND e.datasource_id > 0
  AND d.project_id > 0
  AND e.deleted_at IS NULL`
	case IsDameng(dialect):
		return `
UPDATE alert_events e
SET project_id = (
  SELECT d.project_id FROM alert_datasources d
  WHERE e.datasource_id = d.id AND d.deleted_at IS NULL AND d.project_id > 0
)
WHERE COALESCE(e.project_id, 0) = 0
  AND e.datasource_id > 0
  AND e.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM alert_datasources d
    WHERE e.datasource_id = d.id AND d.deleted_at IS NULL AND d.project_id > 0
  )`
	default:
		return `
UPDATE alert_events e
INNER JOIN alert_datasources d ON e.datasource_id = d.id AND d.deleted_at IS NULL
SET e.project_id = d.project_id
WHERE COALESCE(e.project_id, 0) = 0
  AND e.datasource_id > 0
  AND d.project_id > 0
  AND e.deleted_at IS NULL`
	}
}

// SQLBackfillAlertEventProjectFromSubscriptions 按订阅节点 CSV 回填 alert_events.project_id。
func SQLBackfillAlertEventProjectFromSubscriptions(dialect string) string {
	contains := SQLCSVContainsID(dialect, "n.id", "e.matched_policy_ids")
	switch {
	case IsPostgres(dialect):
		return fmt.Sprintf(`
UPDATE alert_events e
SET project_id = n.project_id
FROM alert_subscription_nodes n
WHERE n.deleted_at IS NULL
  AND n.project_id > 0
  AND %s
  AND COALESCE(e.project_id, 0) = 0
  AND e.matched_policy_ids IS NOT NULL
  AND TRIM(e.matched_policy_ids) <> ''
  AND e.deleted_at IS NULL`, contains)
	case IsDameng(dialect):
		return fmt.Sprintf(`
UPDATE alert_events e
SET project_id = (
  SELECT MIN(n.project_id) FROM alert_subscription_nodes n
  WHERE n.deleted_at IS NULL AND n.project_id > 0 AND %s
)
WHERE COALESCE(e.project_id, 0) = 0
  AND e.matched_policy_ids IS NOT NULL
  AND TRIM(e.matched_policy_ids) <> ''
  AND e.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM alert_subscription_nodes n
    WHERE n.deleted_at IS NULL AND n.project_id > 0 AND %s
  )`, contains, contains)
	default:
		return fmt.Sprintf(`
UPDATE alert_events e
INNER JOIN alert_subscription_nodes n
  ON n.deleted_at IS NULL AND n.project_id > 0
 AND %s
SET e.project_id = n.project_id
WHERE COALESCE(e.project_id, 0) = 0
  AND e.matched_policy_ids IS NOT NULL
  AND TRIM(e.matched_policy_ids) <> ''
  AND e.deleted_at IS NULL`, contains)
	}
}
