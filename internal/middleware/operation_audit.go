package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"yunshu/internal/model"
	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/dictmask"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

const (
	maxAuditBodyBytes  = 8192
	maxStoredBodyRunes = 4000
)

// YAML / env 文本中的敏感键值（pipelines_yml、env 文件、kubeconfig 等）。
var (
	sensitiveYAMLLinePattern = regexp.MustCompile(`(?im)^([ \t]*-[ \t]*|[ \t]*)(["']?(?:password|passwd|pwd|token|secret|api[_-]?key|access[_-]?key|private[_-]?key|secret[_-]?key|client[_-]?secret|client[_-]?key[_-]?data|private[_-]?key[_-]?data|credentials?|authorization)["']?[ \t]*:[ \t]*)(\S.*)$`)
	sensitiveEnvLinePattern  = regexp.MustCompile(`(?im)^([ \t]*(?:export[ \t]+)?[A-Za-z0-9_]*(?:PASSWORD|PASSWD|PWD|TOKEN|SECRET|API_KEY|ACCESS_KEY|PRIVATE_KEY|SECRET_KEY|CREDENTIAL|AUTHORIZATION)[A-Za-z0-9_]*[ \t]*=[ \t]*)(\S.*)$`)
)

type bodyCaptureWriter struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (w *bodyCaptureWriter) Write(b []byte) (int, error) {
	w.capture(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyCaptureWriter) WriteString(s string) (int, error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

// capture 仅保留审计需要的头部字节，避免大文件下载把响应整体缓存在内存。
func (w *bodyCaptureWriter) capture(b []byte) {
	if len(b) == 0 || w.buf.Len() >= maxAuditBodyBytes {
		return
	}
	if remain := maxAuditBodyBytes - w.buf.Len(); len(b) > remain {
		b = b[:remain]
	}
	_, _ = w.buf.Write(b)
}

// OperationAudit 记录已鉴权请求的审计信息（参考 gin-vue-admin 操作日志）
func OperationAudit(opSvc *service.OperationLogService, logger *logx.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := auth.CurrentUserFromContext(c)
		if !ok {
			c.Next()
			return
		}

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		reqBody := ""
		if shouldCaptureRequestBody(c) {
			raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxAuditBodyBytes))
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewReader(raw))
				reqBody = maskSensitivePayloadInPath(string(raw), path)
				reqBody = truncateRunes(reqBody, maxStoredBodyRunes)
			}
		}

		blw := &bodyCaptureWriter{ResponseWriter: c.Writer, buf: bytes.Buffer{}}
		c.Writer = blw

		start := time.Now()
		c.Next()

		latency := time.Since(start).Milliseconds()
		status := c.Writer.Status()
		if status == 0 {
			status = http.StatusOK
		}

		// 文件下载（env/pipelines/YAML/CSV 等）不保留正文，避免把凭据写入审计库。
		respStr := ""
		switch {
		case shouldRedactAuditResponseBody(path):
			respStr = `{"redacted":true,"reason":"high_sensitivity_response"}`
		case !isJSONResponseContentType(c.Writer.Header().Get("Content-Type")):
			respStr = `{"redacted":true,"reason":"non_json_response"}`
		default:
			rawBody := blw.buf.String()
			if !utf8.ValidString(rawBody) {
				respStr = `{"redacted":true,"reason":"non_text_response"}`
			} else {
				respStr = truncateRunes(maskSensitivePayloadInPath(rawBody, path), maxStoredBodyRunes)
			}
		}

		// capture headers and mask sensitive values
		headersMap := map[string][]string{}
		for k, v := range c.Request.Header {
			headersMap[k] = append([]string(nil), v...)
		}
		maskHeaders(headersMap)
		headersJSON := ""
		if b, err := json.Marshal(headersMap); err == nil {
			headersJSON = truncateRunes(string(b), maxStoredBodyRunes)
		}

		entry := model.OperationLog{
			UserID:         user.ID,
			Username:       user.Username,
			Nickname:       user.Nickname,
			IP:             c.ClientIP(),
			RequestHeaders: headersJSON,
			Method:         c.Request.Method,
			Path:           path,
			StatusCode:     status,
			RequestBody:    reqBody,
			ResponseBody:   respStr,
			LatencyMs:      latency,
		}

		if err := opSvc.Record(c.Request.Context(), entry); err != nil {
			logx.With(c.Request.Context(), "component", "http.audit").Error("operation audit persist failed", "error", err, "path", path)
		}
	}
}

func maskHeaders(h map[string][]string) {
	if h == nil {
		return
	}
	sensitive := map[string]struct{}{
		"Authorization": {},
		"Cookie":        {},
		"Set-Cookie":    {},
		"X-Auth-Token":  {},
		"X-Api-Key":     {},
		"Token":         {},
	}
	for k := range h {
		if _, ok := sensitive[k]; ok {
			h[k] = []string{"***"}
		}
	}
}

func shouldCaptureRequestBody(c *gin.Context) bool {
	method := c.Request.Method
	if method != http.MethodPost && method != http.MethodPut && method != http.MethodPatch {
		return false
	}
	ct := c.ContentType()
	if strings.HasPrefix(ct, "multipart/") {
		return false
	}
	if cl := c.Request.ContentLength; cl > 0 && cl > maxAuditBodyBytes {
		return false
	}
	return true
}

func shouldRedactAuditResponseBody(fullPath string) bool {
	p := strings.ToLower(strings.TrimSpace(fullPath))
	if p == "/api/v1/dict/entries/:id/reveal-value" {
		return true
	}
	if strings.Contains(p, "/accounts/") && strings.HasSuffix(p, "/password") {
		return true
	}
	if strings.Contains(p, "/auth/password") {
		return true
	}
	if strings.Contains(p, "/secrets/detail") || strings.Contains(p, "/secrets/reveal") {
		return true
	}
	if strings.Contains(p, "/clusters") && (strings.Contains(p, "kubeconfig") || strings.HasSuffix(p, "/create") || strings.HasSuffix(p, "/update")) {
		return true
	}
	return false
}

func isJSONResponseContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct == "" || ct == "application/json" || strings.HasSuffix(ct, "+json")
}

// maskSensitivePayload 优先按 JSON 递归脱敏（覆盖 pipelines_yml 等嵌套字符串）；
// 非 JSON 文本按 YAML / env 行规则脱敏。
func maskSensitivePayload(s string) string {
	return maskSensitivePayloadInPath(s, "")
}

// maskSensitivePayloadInPath 在通用脱敏基础上，对字典管理接口按条目 dict_type 额外脱敏 value。
func maskSensitivePayloadInPath(s, fullPath string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	dictEntryPath := isDictEntryAuditPath(fullPath)
	if masked, ok := maskJSONPayloadInPath(s, dictEntryPath); ok {
		return masked
	}
	return maskTextSecrets(s)
}

func isDictEntryAuditPath(fullPath string) bool {
	p := strings.ToLower(strings.TrimSpace(fullPath))
	return strings.HasPrefix(p, "/api/v1/dict/entries") || strings.HasPrefix(p, "/api/v1/dict-entries")
}

func maskJSONPayload(raw string) (string, bool) {
	return maskJSONPayloadInPath(raw, false)
}

func maskJSONPayloadInPath(raw string, dictEntryPath bool) (string, bool) {
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return "", false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return "", false
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(maskAuditNodeInPath(v, dictEntryPath)); err != nil {
		return "", false
	}
	return strings.TrimSpace(b.String()), true
}

func maskAuditNode(v any) any {
	return maskAuditNodeInPath(v, false)
}

func maskAuditNodeInPath(v any, dictEntryPath bool) any {
	switch t := v.(type) {
	case map[string]any:
		dictType, _ := t["dict_type"].(string)
		maskDictValue := dictEntryPath && dictmask.SensitiveDictType(dictType)
		for k, val := range t {
			if maskDictValue && strings.EqualFold(strings.TrimSpace(k), "value") {
				if s, isStr := val.(string); isStr && s != "" {
					t[k] = "***"
				}
				continue
			}
			if auditSecretKey(k) || auditCodeKey(k) {
				s, isStr := val.(string)
				if isStr && s == "" {
					continue
				}
				if !isStr && auditCodeKey(k) {
					continue // 数值错误码本身不含凭据，保留可读性
				}
				t[k] = "***"
				continue
			}
			t[k] = maskAuditNodeInPath(val, dictEntryPath)
		}
		return t
	case []any:
		for i := range t {
			t[i] = maskAuditNodeInPath(t[i], dictEntryPath)
		}
		return t
	case string:
		if strings.ContainsAny(t, "\r\n") {
			return maskTextSecrets(t)
		}
		return t
	default:
		return v
	}
}

var auditKeyNormalizer = strings.NewReplacer("_", "", "-", "", ".", "", " ", "")

func normalizeAuditKey(key string) string {
	return auditKeyNormalizer.Replace(strings.ToLower(strings.TrimSpace(key)))
}

func auditSecretKey(key string) bool {
	k := normalizeAuditKey(key)
	switch k {
	case "password", "passwd", "pwd", "token", "authorization", "secret",
		"apikey", "accesskey", "accesskeyid", "secretkey", "secretid", "privatekey",
		"clientsecret", "clientkeydata", "privatekeydata", "tlskey", "credential", "credentials":
		return true
	}
	return strings.HasSuffix(k, "password") || strings.HasSuffix(k, "secret") ||
		strings.HasSuffix(k, "token") || strings.HasSuffix(k, "accesskey") ||
		strings.HasSuffix(k, "privatekey")
}

func auditCodeKey(key string) bool {
	return normalizeAuditKey(key) == "code"
}

func maskTextSecrets(s string) string {
	s = sensitiveYAMLLinePattern.ReplaceAllString(s, "${1}${2}***")
	return sensitiveEnvLinePattern.ReplaceAllString(s, "${1}***")
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…(truncated)"
}
