// Package k8scaps 定义 K8s 集群授权「能力包」码与解析（repository / service 共用，避免循环依赖）。
package k8scaps

import (
	"encoding/json"
	"sort"
	"strings"

	"yunshu/internal/model"
)

const (
	Read         = "read"
	Exec         = "exec"
	Restart      = "restart"
	Scale        = "scale"
	Apply        = "apply"
	Delete       = "delete"
	SecretReveal = "secret_reveal"
	Destructive  = "destructive"

	PresetReadonly     = "readonly"
	PresetReadonlyExec = "readonly_exec"
	PresetAdmin        = "admin"
	PresetCustom       = "custom"

	RankNone         = 0
	RankReadonly     = 1
	RankReadonlyExec = 2
	RankAdmin        = 3
)

type Meta struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func Catalog() []Meta {
	return []Meta{
		{Code: Read, Name: "只读浏览", Description: "列表/详情等 GET（不含 Exec、Secret 明文）"},
		{Code: Exec, Name: "Pod 终端", Description: "Exec 终端与 Pod 内文件上传/删除"},
		{Code: Restart, Name: "重启", Description: "重启 Deployment/StatefulSet/DaemonSet 等"},
		{Code: Scale, Name: "扩缩容", Description: "调整副本数"},
		{Code: Apply, Name: "YAML 变更", Description: "Apply / 创建更新配置类资源"},
		{Code: Delete, Name: "删除资源", Description: "删除 Pod/工作负载等"},
		{Code: SecretReveal, Name: "Secret 明文", Description: "揭示 Secret 内容"},
		{Code: Destructive, Name: "高危运维", Description: "Drain、Helm 装卸载、RBAC Apply 等"},
	}
}

var known = map[string]struct{}{
	Read: {}, Exec: {}, Restart: {}, Scale: {},
	Apply: {}, Delete: {}, SecretReveal: {}, Destructive: {},
}

func All() []string {
	out := make([]string, 0, len(Catalog()))
	for _, m := range Catalog() {
		out = append(out, m.Code)
	}
	return out
}

func ForPreset(preset string) []string {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case PresetReadonly:
		return []string{Read}
	case PresetReadonlyExec:
		return []string{Read, Exec}
	case PresetAdmin:
		return All()
	default:
		return nil
	}
}

// EnsureRead 任意非空能力包自动带上 read（否则控制台无法列表）。
func EnsureRead(caps []string) []string {
	caps = Normalize(caps)
	if len(caps) == 0 {
		return caps
	}
	if Has(caps, Read) {
		return caps
	}
	return Normalize(append(caps, Read))
}

func NameOf(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, m := range Catalog() {
		if m.Code == code {
			return m.Name
		}
	}
	return code
}

func Normalize(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		c := strings.ToLower(strings.TrimSpace(raw))
		if c == "" {
			continue
		}
		if _, ok := known[c]; !ok {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func Marshal(caps []string) string {
	caps = Normalize(caps)
	if len(caps) == 0 {
		return "[]"
	}
	b, err := json.Marshal(caps)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func ParseJSON(raw string) []string {
	s := strings.TrimSpace(raw)
	if s == "" || s == "[]" || s == "null" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(s), &list); err != nil {
		return nil
	}
	return Normalize(list)
}

func FromGrant(g model.K8sClusterAccessGrant) []string {
	if caps := ParseJSON(g.Capabilities); len(caps) > 0 {
		return caps
	}
	return Normalize(ForPreset(g.Preset))
}

func InferPreset(caps []string) string {
	caps = Normalize(caps)
	if len(caps) == 0 {
		return PresetReadonly
	}
	if equalSet(caps, ForPreset(PresetReadonly)) {
		return PresetReadonly
	}
	if equalSet(caps, ForPreset(PresetReadonlyExec)) {
		return PresetReadonlyExec
	}
	if equalSet(caps, All()) {
		return PresetAdmin
	}
	return PresetCustom
}

func equalSet(a, b []string) bool {
	a, b = Normalize(a), Normalize(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Rank(caps []string) int {
	caps = Normalize(caps)
	if len(caps) == 0 {
		return RankNone
	}
	has := func(c string) bool {
		for _, x := range caps {
			if x == c {
				return true
			}
		}
		return false
	}
	if has(Destructive) || has(Apply) || has(Delete) ||
		has(Restart) || has(Scale) || has(SecretReveal) {
		return RankAdmin
	}
	if has(Exec) {
		return RankReadonlyExec
	}
	if has(Read) {
		return RankReadonly
	}
	return RankNone
}

func Has(have []string, need string) bool {
	need = strings.ToLower(strings.TrimSpace(need))
	if need == "" {
		return true
	}
	for _, c := range Normalize(have) {
		if c == need {
			return true
		}
	}
	return false
}

func PresetLabelCN(preset string) string {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case PresetReadonly:
		return "只读"
	case PresetReadonlyExec:
		return "只读+Exec"
	case PresetAdmin:
		return "集群管理"
	case PresetCustom:
		return "自定义能力包"
	default:
		return preset
	}
}
