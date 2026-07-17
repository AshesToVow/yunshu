// genopenapi 从 internal/router/register_*.go 解析 Gin 路由，生成 OpenAPI 3.0.3 文档。
// 用法：go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type route struct {
	method string
	path   string
}

func joinURLPath(base, p string) string {
	if p == "" {
		return base
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if base == "" {
		return p
	}
	return strings.TrimSuffix(base, "/") + p
}

func ginToOpenAPIPath(p string) string {
	re := regexp.MustCompile(`:([a-zA-Z0-9_]+)`)
	return re.ReplaceAllString(p, "{$1}")
}

func isPublic(method, path string) bool {
	if method == "GET" && path == "/api/v1/health" {
		return true
	}
	public := map[string]bool{
		"POST /api/v1/auth/verification-code":      true,
		"POST /api/v1/auth/login-code":              true,
		"POST /api/v1/auth/password-login-code":     true,
		"POST /api/v1/auth/login":                   true,
		"POST /api/v1/auth/email-login":             true,
		"POST /api/v1/auth/register":                true,
		"POST /api/v1/alerts/webhook/alertmanager":  true,
		"POST /api/v1/loggie/heartbeat/report":      true,
	}
	return public[method+" "+path]
}

func tagForPath(path string) string {
	p := strings.TrimPrefix(path, "/api/v1/")
	if p == "" {
		return "System"
	}
	parts := strings.Split(p, "/")
	seg := parts[0]
	switch seg {
	case "auth":
		return "Auth"
	case "health":
		return "System"
	case "plugins":
		return "Plugins"
	case "alerts":
		if len(parts) > 1 && parts[1] == "webhook" {
			return "AlertsWebhook"
		}
		return "Alerts"
	case "loggie":
		return "Loggie"
	case "log-platform":
		return "LogPlatform"
	case "dict":
		return "Dict"
	case "k8s-policies":
		return "K8sScopedPolicy"
	case "k8s":
		if len(parts) > 1 && parts[1] == "event-forward" {
			return "K8sEventForward"
		}
		return "K8s"
	default:
		return upperFirst(seg)
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

var (
	reEngine = regexp.MustCompile(`^\s*(\w+)\s*:=\s*app\.Engine\.Group\("([^"]*)"\)`)
	// Allow trailing middleware args: Group("/:id", middleware.Foo(...))
	reGroup  = regexp.MustCompile(`^\s*(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"`)
	reRoute  = regexp.MustCompile(`^\s*(\w+)\.(GET|POST|PUT|DELETE|PATCH)\("([^"]*)"`)
)

func parseRouterFile(path string, groups map[string]string, routes *[]route, seedAPI bool) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	if seedAPI {
		groups["api"] = "/api/v1"
	}

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.Split(sc.Text(), "//")[0]
		if m := reEngine.FindStringSubmatch(line); m != nil {
			groups[m[1]] = m[2]
			continue
		}
		if m := reGroup.FindStringSubmatch(line); m != nil {
			parent := groups[m[2]]
			if parent != "" {
				groups[m[1]] = joinURLPath(parent, m[3])
			}
			continue
		}
		if m := reRoute.FindStringSubmatch(line); m != nil {
			gv, method, suffix := m[1], m[2], m[3]
			base := groups[gv]
			if base == "" {
				continue
			}
			full := joinURLPath(base, suffix)
			*routes = append(*routes, route{method: method, path: full})
		}
	}
	return sc.Err()
}

func collectRouterFiles(routerDir string) ([]string, error) {
	pattern := filepath.Join(routerDir, "register_*.go")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no register_*.go under %s", routerDir)
	}
	return matches, nil
}

func operationDescription(openAPIPath, method, fullPath string) string {
	if strings.Contains(openAPIPath, "terminal") || strings.Contains(openAPIPath, "exec/ws") {
		return "WebSocket 需先 POST /api/v1/auth/ws-ticket 获取一次性 ticket，再在连接 URL 查询参数中携带 ticket=。\n"
	}
	if method == "POST" && fullPath == "/api/v1/alerts/webhook/alertmanager" {
		return "Alertmanager Webhook。鉴权：请求头 X-Alert-Token 或 Authorization: Bearer <webhook_token>（不支持 URL ?token=）。\n"
	}
	return ""
}

func main() {
	routerDir := flag.String("router-dir", "internal/router", "directory containing register_*.go")
	outPath := flag.String("out", "docs/apipost/permission-system.openapi.yaml", "output openapi yaml")
	flag.Parse()

	files, err := collectRouterFiles(*routerDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect router files: %v\n", err)
		os.Exit(1)
	}

	groups := map[string]string{}
	var routes []route
	for _, f := range files {
		if err := parseRouterFile(f, groups, &routes, true); err != nil {
			fmt.Fprintf(os.Stderr, "parse %s: %v\n", f, err)
			os.Exit(1)
		}
	}

	keySeen := map[string]bool{}
	var uniq []route
	for _, r := range routes {
		k := r.method + " " + r.path
		if keySeen[k] {
			continue
		}
		keySeen[k] = true
		uniq = append(uniq, r)
	}
	sort.Slice(uniq, func(i, j int) bool {
		if uniq[i].path != uniq[j].path {
			return uniq[i].path < uniq[j].path
		}
		return uniq[i].method < uniq[j].method
	})

	var sb strings.Builder
	sb.WriteString(`openapi: 3.0.3
info:
  title: YunShu CMDB / Permission System API
  version: "1.0.0"
  description: |
    由 tools/genopenapi 从 internal/router/register_*.go 自动生成，请勿手工编辑本文件。
    重新生成：go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml
servers:
  - url: http://127.0.0.1:8080
    description: Local
  - url: /
    description: Relative (behind reverse proxy)
tags: []
paths:
`)

	pathOps := map[string]map[string]route{}
	for _, r := range uniq {
		o := ginToOpenAPIPath(r.path)
		if pathOps[o] == nil {
			pathOps[o] = map[string]route{}
		}
		pathOps[o][strings.ToLower(r.method)] = r
	}
	var paths []string
	for p := range pathOps {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		ops := pathOps[p]
		var methods []string
		for m := range ops {
			methods = append(methods, m)
		}
		sort.Strings(methods)

		fmt.Fprintf(&sb, "  %s:\n", p)
		for _, lm := range methods {
			r := ops[lm]
			tag := tagForPath(r.path)
			opID := strings.ReplaceAll(strings.TrimPrefix(p, "/"), "/", "_")
			opID = strings.ReplaceAll(opID, "{", "")
			opID = strings.ReplaceAll(opID, "}", "_")
			opID = fmt.Sprintf("%s_%s", strings.ToLower(r.method), opID)
			sec := ""
			if !isPublic(r.method, r.path) {
				sec = `      security:
        - bearerAuth: []
`
			}
			desc := operationDescription(p, r.method, r.path)
			fmt.Fprintf(&sb, "    %s:\n", strings.ToLower(r.method))
			fmt.Fprintf(&sb, "      tags: [%q]\n", tag)
			fmt.Fprintf(&sb, "      summary: Auto-generated from router\n")
			fmt.Fprintf(&sb, "      operationId: %s\n", opID)
			if desc != "" {
				fmt.Fprintf(&sb, "      description: |\n        %s", desc)
			}
			if sec != "" {
				fmt.Fprintf(&sb, "%s", sec)
			}
			fmt.Fprintf(&sb, `      responses:
        "200":
          description: OK（统一 JSON，见 StandardResponse.data）
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/StandardResponse"
        "400":
          description: 参数错误
        "401":
          description: 未授权
        "403":
          description: 禁止访问
        "500":
          description: 服务器错误
`)
		}
	}

	sb.WriteString(`components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: Authorization Bearer <access_token>
    alertWebhookToken:
      type: apiKey
      in: header
      name: X-Alert-Token
      description: Alertmanager Webhook 鉴权（与 config alert.webhook_token 一致）
  schemas:
    StandardResponse:
      type: object
      properties:
        code:
          type: integer
          example: 200
        message:
          type: string
          example: success
        error_code:
          type: string
          description: 业务错误码（失败时）
        data:
          description: 成功时的载荷，结构因接口而异
`)

	out := sb.String()
	if err := os.WriteFile(*outPath, []byte(out), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "parsed %d files, wrote %d paths (%d operations) to %s\n", len(files), len(paths), len(uniq), *outPath)
}
