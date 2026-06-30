package dictconfig

import (
	"strconv"
	"strings"
)

// ParseBoolLoose 解析字典布尔值（true/1/yes/on 等）。
func ParseBoolLoose(raw string) (bool, bool) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return false, false
	}
	switch s {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}

// ParseIntLoose 解析字典整数值。
func ParseIntLoose(raw string) (int, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
