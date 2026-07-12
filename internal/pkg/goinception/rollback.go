package goinception

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// RollbackItem 单条 SQL 与其回滚语句。
type RollbackItem struct {
	OriginalSQL string `json:"original_sql"`
	RollbackSQL string `json:"rollback_sql"`
	Sequence    string `json:"sequence,omitempty"`
	BackupDB    string `json:"backup_db,omitempty"`
}

var safeIdentRe = regexp.MustCompile(`^[a-zA-Z0-9_$]+$`)

// goInception 备份元数据表名（不同版本前缀略有差异，查询时依次尝试）。
var inceptionBackupInfoTables = []string{
	"$_$Inception_backup_information$_$",
	"_$Inception_backup_information$_$",
}

func normalizeOpidTime(sequence string) string {
	s := strings.TrimSpace(sequence)
	s = strings.Trim(s, "'\"")
	return s
}

func quoteIdent(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// FetchRollbackFromExecuteRows 根据 goInception 执行结果，从目标库备份库读取回滚 SQL。
func FetchRollbackFromExecuteRows(ctx context.Context, db *sql.DB, rows []ReviewRow) ([]RollbackItem, error) {
	if db == nil || len(rows) == 0 {
		return nil, nil
	}
	items := make([]RollbackItem, 0)
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		backupDB := strings.TrimSpace(row.BackupDBName)
		if backupDB == "" || strings.EqualFold(backupDB, "None") || strings.EqualFold(backupDB, "null") {
			continue
		}
		if !safeIdentRe.MatchString(backupDB) {
			continue
		}
		opid := normalizeOpidTime(row.Sequence)
		if opid == "" {
			continue
		}
		tableName, err := lookupBackupTable(ctx, db, backupDB, opid)
		if err != nil || tableName == "" {
			continue
		}
		if !safeIdentRe.MatchString(tableName) {
			continue
		}
		rollbackSQL, err := loadRollbackStatements(ctx, db, backupDB, tableName, opid)
		if err != nil {
			return items, err
		}
		if rollbackSQL == "" {
			continue
		}
		items = append(items, RollbackItem{
			OriginalSQL: row.SQL,
			RollbackSQL: rollbackSQL,
			Sequence:    row.Sequence,
			BackupDB:    backupDB,
		})
	}
	return items, nil
}

func lookupBackupTable(ctx context.Context, db *sql.DB, backupDB, opid string) (string, error) {
	for _, infoTable := range inceptionBackupInfoTables {
		if !safeIdentRe.MatchString(infoTable) {
			continue
		}
		q := fmt.Sprintf("SELECT tablename FROM %s.%s WHERE opid_time = ?",
			quoteIdent(backupDB), quoteIdent(infoTable))
		var tableName string
		err := db.QueryRowContext(ctx, q, opid).Scan(&tableName)
		if err == nil && strings.TrimSpace(tableName) != "" {
			return tableName, nil
		}
		if err != nil && err != sql.ErrNoRows {
			return "", err
		}
	}
	return "", nil
}

func loadRollbackStatements(ctx context.Context, db *sql.DB, backupDB, tableName, opid string) (string, error) {
	q := fmt.Sprintf("SELECT rollback_statement FROM %s.%s WHERE opid_time = ? ORDER BY id",
		quoteIdent(backupDB), quoteIdent(tableName))
	rows, err := db.QueryContext(ctx, q, opid)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var stmt sql.NullString
		if err := rows.Scan(&stmt); err != nil {
			return "", err
		}
		if stmt.Valid && strings.TrimSpace(stmt.String) != "" {
			parts = append(parts, strings.TrimSpace(stmt.String))
		}
	}
	return strings.Join(parts, "\n"), rows.Err()
}

func ParseExecuteRows(raw string) []ReviewRow {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var rs ReviewSet
	if err := jsonUnmarshal(raw, &rs); err == nil && len(rs.Rows) > 0 {
		return rs.Rows
	}
	var rows []ReviewRow
	if err := jsonUnmarshal(raw, &rows); err == nil {
		return rows
	}
	return nil
}

func jsonUnmarshal(raw string, v any) error {
	return json.Unmarshal([]byte(raw), v)
}
