package constants

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// 手写 BizError 业务码须与 bizReasonByCode 一一对应，避免 API reason 退化为 BizError{code}。
func TestBizReasonCoversTypedBizErrors(t *testing.T) {
	raw, err := os.ReadFile("constant.go")
	if err != nil {
		t.Fatalf("read constant.go: %v", err)
	}
	text := string(raw)
	// 仅扫描手写 Err* = BizError 区（脚本生成的 ErrMsg* 不在此列）。
	start := strings.Index(text, "// —— 通用 10xxx ——")
	end := strings.Index(text, "// 展示用语（非 error）")
	if start < 0 || end < 0 || end <= start {
		t.Fatal("failed to locate typed BizError section in constant.go")
	}
	section := text[start:end]
	re := regexp.MustCompile(`BizError\([^,]+,\s*(\d+),`)
	for _, m := range re.FindAllStringSubmatch(section, -1) {
		code, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("invalid biz code %q: %v", m[1], err)
		}
		if _, ok := bizReasonByCode[code]; !ok {
			t.Errorf("bizReasonByCode missing typed BizError code %d", code)
		}
	}
}

func TestBizReasonNoOrphanEntries(t *testing.T) {
	typedCodes := map[int]bool{
		10901: true, 10902: true,
		11020: true, 11021: true, 11022: true, 11023: true, 11024: true,
	}
	raw, err := os.ReadFile("constant.go")
	if err != nil {
		t.Fatalf("read constant.go: %v", err)
	}
	text := string(raw)
	start := strings.Index(text, "// —— 通用 10xxx ——")
	end := strings.Index(text, "// 展示用语（非 error）")
	section := text[start:end]
	re := regexp.MustCompile(`BizError\([^,]+,\s*(\d+),`)
	for _, m := range re.FindAllStringSubmatch(section, -1) {
		code, _ := strconv.Atoi(m[1])
		typedCodes[code] = true
	}
	for code := range bizReasonByCode {
		if !typedCodes[code] {
			t.Errorf("bizReasonByCode has orphan code %d without typed BizError in constant.go", code)
		}
	}
}
