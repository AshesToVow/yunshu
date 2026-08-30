package system

import "testing"

// TestBuiltinDictSeedsNoDuplicate 守门：拆分为 dict_seed_*.go 后，
// 各域种子合并时不得出现重复的「类型 + 标签」（否则 seed 幂等判重会漏掉后者，产生重复行）。
func TestBuiltinDictSeedsNoDuplicate(t *testing.T) {
	seen := make(map[string]struct{})
	for _, item := range builtinDictSeeds() {
		key := item.DictType + "\x00" + item.Label
		if _, dup := seen[key]; dup {
			t.Errorf("duplicated builtin dict seed: dict_type=%s label=%s", item.DictType, item.Label)
		}
		seen[key] = struct{}{}
	}
}

// TestDictSingletonTypesAtMostOneSeed 守门：单值型字典最多只能有一条种子。
// seed 对这类类型按「类型」判重（ExistsByType），多条种子会导致后续项被静默丢弃，
// 表现为「改了常量但数据库里始终是第一条」的隐性 bug。
//
// 允许 0 条：mail_username / mail_password / mail_from_email / mail_from_name 等
// 敏感项刻意不预置示例值，由管理员在数据字典中手工填写。
func TestDictSingletonTypesAtMostOneSeed(t *testing.T) {
	counts := make(map[string]int)
	for _, item := range builtinDictSeeds() {
		counts[item.DictType]++
	}

	for dictType := range dictSingletonTypes() {
		if n := counts[dictType]; n > 1 {
			t.Errorf("singleton dict type %s has %d builtin seeds, want at most 1", dictType, n)
		}
	}
}

// TestBuiltinDictSeedsNonSingletonHaveDistinctValue 守门：非单值型字典按「类型 + 标签」判重，
// 同一类型下不同标签应对应不同 value，避免下拉选项出现两个标签同值的歧义项。
func TestBuiltinDictSeedsNonSingletonHaveDistinctValue(t *testing.T) {
	singletons := dictSingletonTypes()
	seen := make(map[string]string)

	for _, item := range builtinDictSeeds() {
		if _, ok := singletons[item.DictType]; ok {
			continue
		}
		if item.Value == "" {
			continue
		}
		key := item.DictType + "\x00" + item.Value
		if prev, dup := seen[key]; dup {
			t.Errorf("dict_type=%s value=%q used by both label %q and %q", item.DictType, item.Value, prev, item.Label)
		}
		seen[key] = item.Label
	}
}
