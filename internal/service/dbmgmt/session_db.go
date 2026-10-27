package dbmgmt

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/dbconn"
)

func resolveQueryDatabase(inst *model.DbInstance, database string) (string, error) {
	db := strings.TrimSpace(database)
	if db == "" {
		db = strings.TrimSpace(inst.Database)
	}
	if db == "" {
		return "", constants.ErrBadRequestWithMsg("请先选择要操作的数据库")
	}
	return db, nil
}

func withDatabase(ctx context.Context, sess *dbconn.Session, inst *model.DbInstance, database string, fn func(context.Context, *sql.Conn) error) error {
	db, err := resolveQueryDatabase(inst, database)
	if err != nil {
		return err
	}
	// USE / SET search_path 是会话级状态：必须与后续语句落在同一条连接上，
	// 否则会在连接池里串到别的请求，且语句本身可能跑到未切库的连接。
	conn, err := sess.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	switch strings.ToLower(strings.TrimSpace(inst.Driver)) {
	case model.DbDriverPostgres:
		if _, err := conn.ExecContext(ctx, fmt.Sprintf(`SET search_path TO "%s"`, strings.ReplaceAll(db, `"`, `""`))); err != nil {
			return err
		}
	default:
		if _, err := conn.ExecContext(ctx, fmt.Sprintf("USE `%s`", escapeMySQLIdent(db))); err != nil {
			return err
		}
	}
	return fn(ctx, conn)
}
