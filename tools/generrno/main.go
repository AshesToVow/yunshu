// Command generrno 从 internal/pkg/constants/constant.go 提取手写业务错误码，
// 生成 docs/错误码清单.md（业务码 / Reason / HTTP 状态 / 产品话术 / 构造入口）。
// Reason 与 HTTP 状态由 internal/pkg/constants.ReasonForBizCode 与
// internal/pkg/errors.InferHTTPStatus 单一来源推导，文档仅承载展示。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
)

const typedSectionStart = "// —— 通用 10xxx ——"
const typedSectionEnd = "// 展示用语（非 error）"

type row struct {
	code     int
	name     string
	reason   string
	http     int
	message  string
	entrance string
}

// fixedRows：可变文案固定码（Err*WithMsg 构造器，位于 typed 区之外）。
var fixedRows = []row{
	{10901, "ErrInternalWithMsg", "", 0, "内部错误（可变文案）", "ErrInternalWithMsg(msg)"},
	{10902, "ErrTooManyRequestsWithMsg", "", 0, "限流（可变文案）", "ErrTooManyRequestsWithMsg(msg)"},
	{11020, "ErrBadRequestWithMsg", "", 0, "请求参数错误（可变文案）", "ErrBadRequestWithMsg(msg)"},
	{11021, "ErrNotFoundWithMsg", "", 0, "资源不存在（可变文案）", "ErrNotFoundWithMsg(msg)"},
	{11022, "ErrForbiddenWithMsg", "", 0, "无权操作（可变文案）", "ErrForbiddenWithMsg(msg)"},
	{11023, "ErrUnauthorizedWithMsg", "", 0, "未授权（可变文案）", "ErrUnauthorizedWithMsg(msg)"},
	{11024, "ErrConflictWithMsg", "", 0, "状态冲突（可变文案）", "ErrConflictWithMsg(msg)"},
}

func main() {
	out := flag.String("out", "docs/错误码清单.md", "输出文件路径")
	check := flag.Bool("check", false, "校验模式：生成到临时文件并与现有文件比对，漂移则退出码 1")
	flag.Parse()

	rows, err := collect()
	if err != nil {
		fmt.Fprintln(os.Stderr, "generrno:", err)
		os.Exit(2)
	}
	doc := render(rows)
	if *check {
		existing, err := os.ReadFile(*out)
		if err == nil && bytes.Equal(existing, doc) {
			fmt.Println("错误码清单与代码一致")
			return
		}
		fmt.Fprintf(os.Stderr, "错误码清单与代码漂移，请运行：go run ./tools/generrno -out %s\n", *out)
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "generrno:", err)
		os.Exit(2)
	}
	if err := os.WriteFile(*out, doc, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "generrno:", err)
		os.Exit(2)
	}
	fmt.Printf("已生成 %s（%d 个错误码）\n", *out, len(rows))
}

func collect() ([]row, error) {
	raw, err := os.ReadFile(filepath.Join("internal", "pkg", "constants", "constant.go"))
	if err != nil {
		return nil, fmt.Errorf("read constant.go: %w", err)
	}
	msgConsts, err := loadMsgConsts()
	if err != nil {
		return nil, err
	}
	text := string(raw)
	start := strings.Index(text, typedSectionStart)
	end := strings.Index(text, typedSectionEnd)
	if start < 0 || end < 0 || end <= start {
		return nil, fmt.Errorf("failed to locate typed BizError section")
	}
	section := text[start:end]

	lineRe := regexp.MustCompile(`Err(\w+)\s*=\s*BizError\((http\.Status\w+|[^,]+),\s*(\d+),\s*(?:"([^"]*)"|([A-Za-z_]\w*))`)
	rows := make([]row, 0, 128)
	seen := map[int]bool{}
	for _, m := range lineRe.FindAllStringSubmatch(section, -1) {
		code, err := strconv.Atoi(m[3])
		if err != nil {
			return nil, fmt.Errorf("invalid code %q: %w", m[3], err)
		}
		if seen[code] {
			return nil, fmt.Errorf("duplicate typed code %d", code)
		}
		seen[code] = true
		msg := m[4]
		if m[5] != "" {
			v, ok := msgConsts[m[5]]
			if !ok {
				return nil, fmt.Errorf("message constant %s not found in messages_generated.go", m[5])
			}
			msg = v
		}
		rows = append(rows, row{
			code:     code,
			name:     "Err" + m[1],
			reason:   constants.ReasonForBizCode(code),
			http:     httpStatusFromExpr(m[2], code),
			message:  msg,
			entrance: "Err" + m[1],
		})
	}

	// 与既有覆盖率口径交叉校验：typed 区内所有 BizError 码都应被逐行解析到。
	allRe := regexp.MustCompile(`BizError\([^,]+,\s*(\d+),`)
	for _, m := range allRe.FindAllStringSubmatch(section, -1) {
		code, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("invalid code %q: %w", m[1], err)
		}
		if !seen[code] {
			return nil, fmt.Errorf("code %d present in typed section but not parsed (multi-line entry?)", code)
		}
	}

	for _, f := range fixedRows {
		if seen[f.code] {
			return nil, fmt.Errorf("fixed row code %d collides with typed section", f.code)
		}
		seen[f.code] = true
		f.reason = constants.ReasonForBizCode(f.code)
		f.http = bizerrors.InferHTTPStatus(f.code)
		rows = append(rows, f)
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].code < rows[j].code })
	return rows, nil
}

// httpStatusFromExpr 解析 BizError 第一参数：优先取显式 http.Status*（BizError 实际生效的
// HTTPStatus），无法识别时回退按码段推断。
func httpStatusFromExpr(expr string, code int) int {
	statusByToken := map[string]int{
		"http.StatusOK":                  200,
		"http.StatusBadRequest":          400,
		"http.StatusUnauthorized":        401,
		"http.StatusForbidden":           403,
		"http.StatusNotFound":            404,
		"http.StatusConflict":            409,
		"http.StatusTooManyRequests":     429,
		"http.StatusInternalServerError": 500,
	}
	if s, ok := statusByToken[strings.TrimSpace(expr)]; ok {
		return s
	}
	return bizerrors.InferHTTPStatus(code)
}

// loadMsgConsts 解析 messages_generated.go 中的常量定义（文案单一来源，供 typed 区引用）。
func loadMsgConsts() (map[string]string, error) {
	raw, err := os.ReadFile(filepath.Join("internal", "pkg", "constants", "messages_generated.go"))
	if err != nil {
		return nil, fmt.Errorf("read messages_generated.go: %w", err)
	}
	re := regexp.MustCompile(`([A-Za-z_]\w*)\s*=\s*"((?:[^"\\]|\\.)*)"`)
	out := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
		v, err := strconv.Unquote(`"` + m[2] + `"`)
		if err != nil {
			v = m[2]
		}
		out[m[1]] = v
	}
	return out, nil
}

func render(rows []row) []byte {
	var b strings.Builder
	b.WriteString("# 错误码清单\n\n")
	b.WriteString("> 本文件由 `go run ./tools/generrno` 生成，勿手改。\n")
	b.WriteString("> Reason 与 HTTP 状态单一来源：`internal/pkg/constants/bizReasonByCode`、BizError 第一参数（http.Status*，兜底按码段推断）。\n")
	b.WriteString("> 手写业务码位于 `internal/pkg/constants/constant.go`（10xxx 通用 / 11xxx 请求校验 / 20xxx–26xxx 按功能域分段）；\n")
	b.WriteString("> 可变文案固定码（Err*WithMsg）允许调用方传入产品话术。\n\n")
	b.WriteString("| 业务码 | Reason | HTTP 状态 | 产品话术 | 构造入口 |\n")
	b.WriteString("| :---: | :--- | :---: | :--- | :--- |\n")
	for _, r := range rows {
		msg := strings.ReplaceAll(r.message, "|", "\\|")
		b.WriteString(fmt.Sprintf("| %d | %s | %d | %s | `%s` |\n", r.code, r.reason, r.http, msg, r.entrance))
	}
	return []byte(b.String())
}
