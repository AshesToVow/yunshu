package dbmgmt

import (
	"encoding/json"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
)

type accessRequestMeta struct {
	Charset        string `json:"charset,omitempty"`
	Collation      string `json:"collation,omitempty"`
	DevOwnerUserID uint   `json:"dev_owner_user_id,omitempty"`
	DevOwnerName   string `json:"dev_owner_name,omitempty"`
	DbaUserID      uint   `json:"dba_user_id,omitempty"`
	DbaName        string `json:"dba_name,omitempty"`
	GrantHosts     string `json:"grant_hosts,omitempty"`
}

type AccessRequestMetaItem struct {
	Charset      string `json:"charset,omitempty"`
	Collation    string `json:"collation,omitempty"`
	DevOwnerName string `json:"dev_owner_name,omitempty"`
	DbaName      string `json:"dba_name,omitempty"`
	GrantHosts   string `json:"grant_hosts,omitempty"`
}

func parseAccessRequestMeta(raw string) accessRequestMeta {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return accessRequestMeta{}
	}
	var m accessRequestMeta
	_ = json.Unmarshal([]byte(raw), &m)
	return m
}

func marshalAccessRequestMeta(m accessRequestMeta) string {
	b, _ := json.Marshal(m)
	return string(b)
}

func toAccessRequestMetaItem(m accessRequestMeta) *AccessRequestMetaItem {
	if m.Charset == "" && m.Collation == "" && m.DevOwnerName == "" && m.DbaName == "" && m.GrantHosts == "" {
		return nil
	}
	return &AccessRequestMetaItem{
		Charset: m.Charset, Collation: m.Collation,
		DevOwnerName: m.DevOwnerName, DbaName: m.DbaName, GrantHosts: m.GrantHosts,
	}
}

var mysqlCharsetCollations = map[string][]string{
	"utf8":    {"utf8_general_ci", "utf8_bin"},
	"utf8mb4": {"utf8mb4_general_ci", "utf8mb4_bin", "utf8mb4_unicode_ci"},
}

func validateDatabaseCreateMeta(charset, collation string) error {
	cs := strings.ToLower(strings.TrimSpace(charset))
	if cs == "" {
		cs = "utf8mb4"
	}
	allowed, ok := mysqlCharsetCollations[cs]
	if !ok {
		return constants.ErrBadRequestWithMsg("字符集仅支持 utf8、utf8mb4")
	}
	col := strings.TrimSpace(collation)
	if col == "" {
		return constants.ErrBadRequestWithMsg("请选择校验规则")
	}
	for _, a := range allowed {
		if a == col {
			return nil
		}
	}
	return constants.ErrBadRequestWithMsg("字符集与校验规则不匹配")
}

func defaultCollationForCharset(charset string) string {
	cs := strings.ToLower(strings.TrimSpace(charset))
	if cs == "utf8" {
		return "utf8_general_ci"
	}
	return "utf8mb4_general_ci"
}

func isAllowedDbCharset(charset string) bool {
	_, ok := mysqlCharsetCollations[strings.ToLower(strings.TrimSpace(charset))]
	return ok
}

func isAllowedDbCollation(collation string) bool {
	col := strings.TrimSpace(collation)
	for _, list := range mysqlCharsetCollations {
		for _, item := range list {
			if item == col {
				return true
			}
		}
	}
	return false
}

func userDisplayName(u *model.User) string {
	if u == nil {
		return ""
	}
	if n := strings.TrimSpace(u.Nickname); n != "" {
		return n
	}
	return strings.TrimSpace(u.Username)
}
