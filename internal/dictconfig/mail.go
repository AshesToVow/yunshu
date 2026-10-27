package dictconfig

import (
	"context"

	"yunshu/internal/config"

	"gorm.io/gorm"
)

// MailDictTypes 数据字典中覆盖 mail.* 的 dict_type（与 bootstrap 一致）。
type MailDictTypes struct {
	Host      string
	Port      string
	Username  string
	Password  string
	FromEmail string
	FromName  string
	UseTLS    string
}

func DefaultMailDictTypes() MailDictTypes {
	return MailDictTypes{
		Host:      "mail_host",
		Port:      "mail_port",
		Username:  "mail_username",
		Password:  "mail_password",
		FromEmail: "mail_from_email",
		FromName:  "mail_from_name",
		UseTLS:    "mail_use_tls",
	}
}

func parseBoolLoose(raw string) (bool, bool) { return ParseBoolLoose(raw) }

func parseInt(raw string) (int, bool) { return ParseIntLoose(raw) }

// ResolveMailConfig 以 yamlBase 为底，用已启用的数据字典项覆盖（字典存在则优先）。
func ResolveMailConfig(ctx context.Context, db *gorm.DB, yamlBase config.MailConfig, types MailDictTypes) config.MailConfig {
	cfg := yamlBase
	if db == nil {
		return cfg
	}
	snap := fetchDictSnapshot(ctx, db,
		types.Host, types.Port, types.Username, types.Password,
		types.FromEmail, types.FromName, types.UseTLS,
	)

	if v, ok := snap.getNonEmpty(types.Host); ok {
		cfg.Host = v
	}
	if v, ok := snap.get(types.Port); ok {
		if n, ok2 := parseInt(v); ok2 && n > 0 {
			cfg.Port = n
		}
	}
	if v, ok := snap.getNonEmpty(types.Username); ok {
		cfg.Username = v
	}
	if v, ok := snap.getNonEmpty(types.Password); ok {
		cfg.Password = v
	}
	if v, ok := snap.getNonEmpty(types.FromEmail); ok {
		cfg.FromEmail = v
	}
	if v, ok := snap.getNonEmpty(types.FromName); ok {
		cfg.FromName = v
	}
	if v, ok := snap.get(types.UseTLS); ok {
		if bv, ok2 := parseBoolLoose(v); ok2 {
			cfg.UseTLS = bv
		}
	}
	return cfg
}
