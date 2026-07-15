package dictcategory

import (
	"fmt"
	"strings"
)

// rule 描述某分类下 dict_type 的匹配规则（大小写不敏感）。
type rule struct {
	prefixes []string
	exact    []string
}

var rules = map[string]rule{
	"system": {prefixes: []string{"mail_"}, exact: []string{"common_status"}},
	"alert":  {prefixes: []string{"alert_", "wecom_", "dingtalk_"}},
	"log":    {prefixes: []string{"log_"}},
	"k8s":    {prefixes: []string{"k8s_"}},
	"cmdb":   {prefixes: []string{"server_", "cloud_"}},
	"backup": {prefixes: []string{"minio_", "mysql_backup_"}},
	"dbmgmt": {prefixes: []string{"dbmgmt_"}},
	"cicd":   {prefixes: []string{"cicd_"}},
}

// NormalizeID 规范化分类 ID；未知值返回空字符串。
func NormalizeID(category string) string {
	c := strings.ToLower(strings.TrimSpace(category))
	if c == "" || c == "all" {
		return ""
	}
	if c == "other" {
		return "other"
	}
	if _, ok := rules[c]; ok {
		return c
	}
	return ""
}

// ApplyFilter 为 GORM 查询追加分类 WHERE（返回 clause 与参数）。
func ApplyFilter(category string) (clause string, args []interface{}, ok bool) {
	id := NormalizeID(category)
	if id == "" {
		return "", nil, false
	}
	if id == "other" {
		return buildOtherClause()
	}
	r, exists := rules[id]
	if !exists {
		return "", nil, false
	}
	parts := make([]string, 0, len(r.prefixes)+len(r.exact))
	for _, p := range r.prefixes {
		parts = append(parts, "LOWER(dict_type) LIKE ?")
		args = append(args, strings.ToLower(p)+"%")
	}
	for _, e := range r.exact {
		parts = append(parts, "LOWER(dict_type) = ?")
		args = append(args, strings.ToLower(e))
	}
	if len(parts) == 0 {
		return "", nil, false
	}
	return "(" + strings.Join(parts, " OR ") + ")", args, true
}

func buildOtherClause() (string, []interface{}, bool) {
	parts := make([]string, 0, 16)
	args := make([]interface{}, 0, 16)
	for _, r := range rules {
		for _, p := range r.prefixes {
			parts = append(parts, "LOWER(dict_type) NOT LIKE ?")
			args = append(args, strings.ToLower(p)+"%")
		}
		for _, e := range r.exact {
			parts = append(parts, "LOWER(dict_type) <> ?")
			args = append(args, strings.ToLower(e))
		}
	}
	if len(parts) == 0 {
		return "", nil, false
	}
	return "(" + strings.Join(parts, " AND ") + ")", args, true
}

// Resolve 根据 dict_type 推断分类（大小写不敏感）。
func Resolve(dictType string) string {
	key := strings.ToLower(strings.TrimSpace(dictType))
	if key == "" {
		return "other"
	}
	for id, r := range rules {
		for _, e := range r.exact {
			if key == strings.ToLower(e) {
				return id
			}
		}
		for _, p := range r.prefixes {
			if strings.HasPrefix(key, strings.ToLower(p)) {
				return id
			}
		}
	}
	return "other"
}

// Label 返回分类中文名。
func Label(category string) string {
	switch NormalizeID(category) {
	case "system":
		return "系统"
	case "alert":
		return "告警"
	case "log":
		return "日志"
	case "k8s":
		return "Kubernetes"
	case "cmdb":
		return "CMDB / 服务器"
	case "backup":
		return "备份 / MinIO"
	case "dbmgmt":
		return "数据库管理"
	case "cicd":
		return "CI/CD"
	case "other":
		return "其他"
	default:
		return category
	}
}

// Validate 校验分类 ID 是否合法（含 all）。
func Validate(category string) error {
	c := strings.ToLower(strings.TrimSpace(category))
	if c == "" || c == "all" {
		return nil
	}
	if NormalizeID(c) != "" || c == "other" {
		return nil
	}
	return fmt.Errorf("unknown dict category %q", category)
}
