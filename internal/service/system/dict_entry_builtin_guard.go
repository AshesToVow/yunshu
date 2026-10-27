package system

import (
	"strings"
	"sync"
)

// 内置字典项保护：seed 会在启动时重建缺失条目，若允许改类型/标签或删除，
// 管理端操作会被静默回滚，甚至因标签改名在下次启动产生重复条目。
var (
	builtinDictGuardOnce    sync.Once
	builtinDictTypeSet      map[string]struct{}
	builtinDictTypeLabelSet map[string]struct{}
)

func loadBuiltinDictGuardSets() {
	builtinDictTypeSet = make(map[string]struct{}, len(dictSingletonTypes()))
	for dictType := range dictSingletonTypes() {
		builtinDictTypeSet[canonicalDictType(dictType)] = struct{}{}
	}
	seeds := builtinDictSeeds()
	builtinDictTypeLabelSet = make(map[string]struct{}, len(seeds))
	for _, seed := range seeds {
		builtinDictTypeLabelSet[dictTypeLabelKey(seed.DictType, seed.Label)] = struct{}{}
	}
}

func dictTypeLabelKey(dictType, label string) string {
	return canonicalDictType(dictType) + "\x00" + strings.TrimSpace(label)
}

// isSingletonDictType 判断是否为单值型字典（全局仅允许一条记录）。
func isSingletonDictType(dictType string) bool {
	builtinDictGuardOnce.Do(loadBuiltinDictGuardSets)
	_, ok := builtinDictTypeSet[canonicalDictType(dictType)]
	return ok
}

// isBuiltinDictEntry 判断条目是否属于内置种子：单值型字典按类型判定，其余按类型+标签判定。
func isBuiltinDictEntry(dictType, label string) bool {
	if isSingletonDictType(dictType) {
		return true
	}
	builtinDictGuardOnce.Do(loadBuiltinDictGuardSets)
	_, ok := builtinDictTypeLabelSet[dictTypeLabelKey(dictType, label)]
	return ok
}
