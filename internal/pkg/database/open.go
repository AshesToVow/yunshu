package database

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunshu/internal/config"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open 按配置驱动（mysql / postgres）建立 GORM 连接并完成关联表注册与连接池设置。
func Open(cfg config.DatabaseConfig, sqlLogger *logx.Logger, logLevel string) (*gorm.DB, error) {
	if sqlLogger == nil {
		return nil, fmt.Errorf("logger is required for database")
	}

	driver := NormalizeDriver(cfg.Driver)
	dialector, err := openDialector(driver, cfg)
	if err != nil {
		return nil, err
	}

	var gormLog *logx.GormLogger
	if sqlLogger.SQL != nil {
		gormLog = logx.NewGormLogger(sqlLogger.SQL, logLevel)
	}

	db, err := gorm.Open(dialector, &gorm.Config{Logger: gormLog})
	if err != nil {
		return nil, err
	}
	if err = SetupJoinTables(db); err != nil {
		return nil, err
	}
	if err = ConfigurePool(db, cfg); err != nil {
		return nil, err
	}
	return db, nil
}

// NormalizeDriver 归一化驱动名；空值默认为 mysql。
func NormalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "", "mysql", "mariadb":
		return "mysql"
	case "postgres", "postgresql", "pg":
		return "postgres"
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

func openDialector(driver string, cfg config.DatabaseConfig) (gorm.Dialector, error) {
	switch driver {
	case "mysql":
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=%s",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.DBName,
			cfg.Charset,
			cfg.Loc,
		)
		return mysql.Open(dsn), nil
	case "postgres":
		return postgres.Open(buildPostgresDSN(cfg)), nil
	default:
		return nil, fmt.Errorf("unsupported database driver %q (supported: mysql, postgres)", cfg.Driver)
	}
}

func buildPostgresDSN(cfg config.DatabaseConfig) string {
	host := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	u := &url.URL{
		Scheme: "postgres",
		Host:   host,
		Path:   cfg.DBName,
	}
	if cfg.User != "" {
		if cfg.Password != "" {
			u.User = url.UserPassword(cfg.User, cfg.Password)
		} else {
			u.User = url.User(cfg.User)
		}
	}
	q := u.Query()
	q.Set("sslmode", cfg.SSLMode)
	q.Set("TimeZone", cfg.TimeZone)
	u.RawQuery = q.Encode()
	return u.String()
}

// SetupJoinTables 注册带扩展字段的 user_roles / user_group_users 关联表。
func SetupJoinTables(db *gorm.DB) error {
	if err := db.SetupJoinTable(&model.User{}, "Roles", &model.UserRole{}); err != nil {
		return err
	}
	return db.SetupJoinTable(&model.User{}, "Groups", &model.UserGroupUser{})
}

// ConfigurePool 应用连接池参数。
func ConfigurePool(db *gorm.DB, cfg config.DatabaseConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSeconds) * time.Second)
	return nil
}
