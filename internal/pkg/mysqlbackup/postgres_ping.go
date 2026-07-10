package mysqlbackup

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PingPostgres 检查远端 PostgreSQL 实例是否可连接（备份目标探测，与平台库驱动无关）。
func PingPostgres(ctx context.Context, host string, port int, user, password, database, sslmode string) error {
	if port <= 0 {
		port = 5432
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	user = strings.TrimSpace(user)
	if user == "" {
		return fmt.Errorf("postgres user is required")
	}
	database = strings.TrimSpace(database)
	if database == "" {
		database = "postgres"
	}
	if strings.TrimSpace(sslmode) == "" {
		sslmode = "disable"
	}

	u := &url.URL{Scheme: "postgres", Host: fmt.Sprintf("%s:%d", host, port), Path: database}
	if password != "" {
		u.User = url.UserPassword(user, password)
	} else {
		u.User = url.User(user)
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	q.Set("connect_timeout", "5")
	u.RawQuery = q.Encode()

	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return err
	}
	defer db.Close()

	pctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return db.PingContext(pctx)
}

// FormatPostgresConnectLog 生成可读连接描述（日志回显）。
func FormatPostgresConnectLog(host string, port int, user, database, sslmode string) string {
	if port <= 0 {
		port = 5432
	}
	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1"
	}
	if strings.TrimSpace(database) == "" {
		database = "postgres"
	}
	if strings.TrimSpace(sslmode) == "" {
		sslmode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s db=%s sslmode=%s", host, port, strings.TrimSpace(user), database, sslmode)
}

// FormatPgDumpConnectArgs 生成 pg_dump 连接参数（不含密码，密码由调用方通过环境变量 PGPASSWORD 注入）。
func FormatPgDumpConnectArgs(host string, port int, user, database string, quote func(string) string) string {
	if quote == nil {
		quote = func(s string) string { return s }
	}
	if port <= 0 {
		port = 5432
	}
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	if strings.TrimSpace(database) == "" {
		database = "postgres"
	}
	return fmt.Sprintf("--host=%s --port=%d --username=%s --dbname=%s", quote(host), port, quote(strings.TrimSpace(user)), quote(database))
}
