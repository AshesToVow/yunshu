package dbmgmt

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"
)

// --- Metadata ---

type DatabaseInfo struct {
	Name string `json:"name"`
}

type TableInfo struct {
	Name    string `json:"name"`
	Schema  string `json:"schema,omitempty"`
	Comment string `json:"comment,omitempty"`
}

type ColumnInfo struct {
	Name       string `json:"name"`
	DataType   string `json:"data_type"`
	Nullable   bool   `json:"nullable"`
	DefaultVal string `json:"default,omitempty"`
	Comment    string `json:"comment,omitempty"`
}

func (s *Service) ListDatabases(ctx context.Context, projectID, instanceID uint, actor *auth.CurrentUser) ([]DatabaseInfo, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if err := s.checkMetadataPermission(ctx, projectID, inst, actor); err != nil {
		return nil, err
	}
	release := s.acquireInstance(instanceID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	var rows *sql.Rows
	var err2 error
	switch strings.ToLower(inst.Driver) {
	case model.DbDriverPostgres:
		rows, err2 = sess.DB.QueryContext(ctx, `SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY 1`)
	default:
		rows, err2 = sess.DB.QueryContext(ctx, `SHOW DATABASES`)
	}
	if err2 != nil {
		return nil, err2
	}
	defer rows.Close()
	var out []DatabaseInfo
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if shouldHideDatabase(name) {
			continue
		}
		out = append(out, DatabaseInfo{Name: name})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sc, err := s.resolveMetadataScope(ctx, projectID, inst, actor)
	if err != nil {
		return nil, err
	}
	out = sc.filterDatabases(out)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Service) ListTables(ctx context.Context, projectID, instanceID uint, database string, actor *auth.CurrentUser) ([]TableInfo, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if err := s.checkMetadataPermission(ctx, projectID, inst, actor); err != nil {
		return nil, err
	}
	release := s.acquireInstance(instanceID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	db := strings.TrimSpace(database)
	switch strings.ToLower(inst.Driver) {
	case model.DbDriverPostgres:
		if db == "" {
			db = "public"
		}
		if err := s.requireMetadataDatabaseAccess(ctx, projectID, inst, db, actor); err != nil {
			return nil, err
		}
		rows, err := sess.DB.QueryContext(ctx, `SELECT tablename FROM pg_tables WHERE schemaname = $1 ORDER BY 1`, db)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		list, err := scanTableNames(rows)
		if err != nil {
			return nil, err
		}
		sc, err := s.resolveMetadataScope(ctx, projectID, inst, actor)
		if err != nil {
			return nil, err
		}
		return sc.filterTables(db, list), nil
	default:
		if db == "" {
			return nil, constants.ErrBadRequestWithMsg("须指定 database")
		}
		if err := s.requireMetadataDatabaseAccess(ctx, projectID, inst, db, actor); err != nil {
			return nil, err
		}
		q := fmt.Sprintf("SHOW TABLES FROM `%s`", escapeMySQLIdent(db))
		rows, err := sess.DB.QueryContext(ctx, q)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		list, err := scanTableNames(rows)
		if err != nil {
			return nil, err
		}
		sc, err := s.resolveMetadataScope(ctx, projectID, inst, actor)
		if err != nil {
			return nil, err
		}
		return sc.filterTables(db, list), nil
	}
}

func scanTableNames(rows *sql.Rows) ([]TableInfo, error) {
	var out []TableInfo
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, TableInfo{Name: name})
	}
	return out, rows.Err()
}

func (s *Service) ListColumns(ctx context.Context, projectID, instanceID uint, database, table string, actor *auth.CurrentUser) ([]ColumnInfo, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if err := s.checkMetadataPermission(ctx, projectID, inst, actor); err != nil {
		return nil, err
	}
	release := s.acquireInstance(instanceID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	db := strings.TrimSpace(database)
	tbl := strings.TrimSpace(table)
	if tbl == "" {
		return nil, constants.ErrBadRequestWithMsg("须指定 table")
	}
	if db == "" && strings.ToLower(inst.Driver) == model.DbDriverPostgres {
		db = "public"
	}
	if db == "" {
		return nil, constants.ErrBadRequestWithMsg("须指定 database")
	}
	if err := s.requireMetadataTableAccess(ctx, projectID, inst, db, tbl, actor); err != nil {
		return nil, err
	}

	switch strings.ToLower(inst.Driver) {
	case model.DbDriverPostgres:
		if db == "" {
			db = "public"
		}
		rows, err := sess.DB.QueryContext(ctx,
			`SELECT column_name, data_type, is_nullable, COALESCE(column_default::text,''), '' FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`,
			db, tbl)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanColumns(rows)
	default:
		if db == "" {
			return nil, constants.ErrBadRequestWithMsg("须指定 database")
		}
		q := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`.`%s`", escapeMySQLIdent(db), escapeMySQLIdent(tbl))
		rows, err := sess.DB.QueryContext(ctx, q)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []ColumnInfo
		for rows.Next() {
			var field, colType, collation, nullStr, key, def, extra, priv, comment sql.NullString
			if err := rows.Scan(&field, &colType, &collation, &nullStr, &key, &def, &extra, &priv, &comment); err != nil {
				return nil, err
			}
			out = append(out, ColumnInfo{
				Name: field.String, DataType: colType.String,
				Nullable: nullStr.String == "YES", DefaultVal: def.String, Comment: comment.String,
			})
		}
		return out, rows.Err()
	}
}

func scanColumns(rows *sql.Rows) ([]ColumnInfo, error) {
	var out []ColumnInfo
	for rows.Next() {
		var name, dtype, nullable, def, comment string
		if err := rows.Scan(&name, &dtype, &nullable, &def, &comment); err != nil {
			return nil, err
		}
		out = append(out, ColumnInfo{
			Name: name, DataType: dtype, Nullable: nullable == "YES", DefaultVal: def, Comment: comment,
		})
	}
	return out, rows.Err()
}

// --- Query ---

type QueryRequest struct {
	Database string `json:"database"`
	Sql      string `json:"sql" binding:"required"`
}

type QueryResult struct {
	Columns    []string `json:"columns"`
	Rows       [][]any  `json:"rows"`
	RowCount   int      `json:"row_count"`
	DurationMs int64    `json:"duration_ms"`
	Truncated  bool     `json:"truncated"`
}

func (s *Service) ExecuteQuery(ctx context.Context, projectID, instanceID uint, req QueryRequest, actor *auth.CurrentUser) (*QueryResult, error) {
	inst, err := s.repo.GetInstanceInProject(ctx, projectID, instanceID)
	if err != nil {
		return nil, err
	}
	if _, err := resolveQueryDatabase(inst, req.Database); err != nil {
		return nil, err
	}
	if err := s.checkQueryPermission(ctx, projectID, inst, req.Database, actor); err != nil {
		return nil, err
	}
	sqlText, err := normalizeQuerySQL(req.Sql)
	if err != nil {
		return nil, err
	}
	if !isReadOnlySQL(sqlText) {
		return nil, constants.ErrBadRequestWithMsg("控制台仅允许 SELECT/SHOW/DESCRIBE/EXPLAIN")
	}
	cfg := s.resolvedConfig(ctx)
	maxRows := cfg.MaxResultRows
	if maxRows <= 0 {
		maxRows = 1000
	}
	rowLimit, err := s.resolveQueryAccess(ctx, projectID, inst, req.Database, sqlText, actor, maxRows)
	if err != nil {
		return nil, err
	}
	sqlText = enforceLimit(sqlText, rowLimit)

	release := s.acquireInstance(instanceID)
	defer release()
	sess, err := s.openSession(ctx, inst)
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	timeout := time.Duration(cfg.QueryTimeoutSeconds) * time.Second
	qctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	var cols []string
	var data [][]any
	err = withDatabase(qctx, sess, inst, req.Database, func(ctx context.Context, db *sql.DB) error {
		rows, qerr := db.QueryContext(ctx, sqlText)
		if qerr != nil {
			return qerr
		}
		defer rows.Close()
		cols, data, qerr = scanRows(rows, rowLimit)
		return qerr
	})
	if err != nil {
		return nil, err
	}
	dur := time.Since(start).Milliseconds()
	truncated := len(data) >= rowLimit

	preview, _ := json.Marshal(map[string]any{"columns": cols, "rows": data})
	ex := &model.DbSqlExecution{
		ProjectID: projectID, InstanceID: instanceID,
		ExecutorUserID: actorUserID(actor), ExecutorName: actorUsername(actor),
		DatabaseName: req.Database, SqlExcerpt: truncateSQL(sqlText, 2000),
		RowsAffected: int64(len(data)), DurationMs: dur,
		ResultPreviewJSON: string(preview), RiskLevel: model.DbRiskLow,
	}
	_ = s.repo.CreateSqlExecution(ctx, ex)
	_ = s.writeAudit(ctx, projectID, &instanceID, actor, "console_query", map[string]any{
		"database": req.Database,
		"sql":      truncateSQL(sqlText, 500),
	})

	return &QueryResult{Columns: nonNilCols(cols), Rows: nonNilRows(data), RowCount: len(data), DurationMs: dur, Truncated: truncated}, nil
}

func truncateSQL(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (s *Service) ListExecutions(ctx context.Context, projectID uint, instanceID uint, executorUserID uint, queryOnly bool, actor *auth.CurrentUser, page, pageSize int) (*pagination.Result[model.DbSqlExecution], error) {
	if actor != nil {
		selfID := actorUserID(actor)
		if executorUserID == 0 {
			executorUserID = selfID
		} else if executorUserID != selfID && !auth.IsSuperAdminRole(actor.RoleCodes) {
			if ok, _ := s.isProjectAdminOrOwner(ctx, projectID, actor); !ok {
				executorUserID = selfID
			}
		}
	}
	list, total, err := s.repo.ListSqlExecutions(ctx, repository.DbSqlExecutionListParams{
		ProjectID: projectID, InstanceID: instanceID, ExecutorUserID: executorUserID,
		QueryOnly: queryOnly, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, err
	}
	return paginate(list, total, page, pageSize), nil
}

func (s *Service) ListQueryHistory(ctx context.Context, projectID uint, instanceID uint, actor *auth.CurrentUser, page, pageSize int) (*pagination.Result[model.DbSqlExecution], error) {
	return s.ListExecutions(ctx, projectID, instanceID, actorUserID(actor), true, actor, page, pageSize)
}

