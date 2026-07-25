package menu

import "strings"

// PathStatusMap 返回内置目录 path → 期望 status（含 Hidden→0）。
func PathStatusMap() map[string]int {
	out := make(map[string]int)
	var walk func([]Spec)
	walk = func(specs []Spec) {
		for _, s := range specs {
			p := strings.TrimSpace(s.Path)
			if p != "" {
				out[p] = s.statusOrDefault()
			}
			if len(s.Children) > 0 {
				walk(s.Children)
			}
		}
	}
	walk(DefaultCatalog())
	return out
}
