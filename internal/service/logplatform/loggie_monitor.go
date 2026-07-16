package logplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const loggieMonitorProbeTimeout = 4 * time.Second

// LoggieMonitorProbeResult Yunshu 侧直连 Loggie HTTP 监控端口的探测结果。
type LoggieMonitorProbeResult struct {
	Reachable           bool   `json:"reachable"`
	ActiveFdCount       int    `json:"active_fd_count"`
	InActiveFdCount     int    `json:"inactive_fd_count"`
	ActivePipelineCount int    `json:"active_pipeline_count"`
	PipelineNames       []string `json:"pipeline_names,omitempty"`
	DetailJSON          string `json:"detail_json,omitempty"`
	Error               string `json:"error,omitempty"`
}

type loggieHelpLogResponse struct {
	FdStatus struct {
		ActiveFdCount   int `json:"activeFdCount"`
		InActiveFdCount int `json:"inActiveFdCount"`
	} `json:"fdStatus"`
	FileStatus struct {
		Pipeline map[string]any `json:"pipeline"`
	} `json:"fileStatus"`
}

// ProbeLoggieMonitor 从 Yunshu 探测目标主机 Loggie HTTP 端口（默认 9196 /api/v1/help/log）。
func ProbeLoggieMonitor(ctx context.Context, host string, port int) LoggieMonitorProbeResult {
	host = strings.TrimSpace(host)
	if host == "" {
		return LoggieMonitorProbeResult{Error: "host empty"}
	}
	if port <= 0 {
		port = 9196
	}
	url := fmt.Sprintf("http://%s:%d/api/v1/help/log", host, port)

	reqCtx, cancel := context.WithTimeout(ctx, loggieMonitorProbeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return LoggieMonitorProbeResult{Error: err.Error()}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return LoggieMonitorProbeResult{Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return LoggieMonitorProbeResult{Error: fmt.Sprintf("HTTP %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return LoggieMonitorProbeResult{Error: err.Error()}
	}

	active, inactive, names, parseOK := parseLoggieHelpLogJSON(body)
	detail, _ := json.Marshal(map[string]any{
		"active_fd":   active,
		"inactive_fd": inactive,
		"pipelines":   names,
	})
	if !parseOK {
		return LoggieMonitorProbeResult{
			Reachable:  true,
			DetailJSON: truncateString(string(body), 2048),
			Error:      "parse help/log json failed",
		}
	}

	return LoggieMonitorProbeResult{
		Reachable:           true,
		ActiveFdCount:       active,
		InActiveFdCount:     inactive,
		ActivePipelineCount: len(names),
		PipelineNames:       names,
		DetailJSON:          string(detail),
	}
}

func parseLoggieHelpLogJSON(body []byte) (active, inactive int, pipelineNames []string, ok bool) {
	var parsed loggieHelpLogResponse
	if err := json.Unmarshal(body, &parsed); err == nil {
		active = parsed.FdStatus.ActiveFdCount
		inactive = parsed.FdStatus.InActiveFdCount
		for name := range parsed.FileStatus.Pipeline {
			pipelineNames = append(pipelineNames, name)
		}
		// 即使结构部分为空也算解析成功（可能暂时无 FD）
		ok = true
	}

	// 兜底：在任意嵌套中查找 activeFdCount
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return active, inactive, pipelineNames, ok
	}
	if active == 0 {
		if n, found := findJSONInt(raw, "activeFdCount", "active_fd_count"); found {
			active = n
			ok = true
		}
	}
	if inactive == 0 {
		if n, found := findJSONInt(raw, "inActiveFdCount", "inactiveFdCount", "inactive_fd_count"); found {
			inactive = n
			ok = true
		}
	}
	if len(pipelineNames) == 0 {
		if m := findJSONMap(raw, "pipeline"); m != nil {
			for name := range m {
				pipelineNames = append(pipelineNames, name)
			}
			ok = true
		}
	}
	return active, inactive, pipelineNames, ok
}

func findJSONInt(v any, keys ...string) (int, bool) {
	switch t := v.(type) {
	case map[string]any:
		for _, k := range keys {
			if raw, exists := t[k]; exists {
				switch n := raw.(type) {
				case float64:
					return int(n), true
				case json.Number:
					i, err := n.Int64()
					if err == nil {
						return int(i), true
					}
				case int:
					return n, true
				}
			}
		}
		for _, child := range t {
			if n, found := findJSONInt(child, keys...); found {
				return n, true
			}
		}
	case []any:
		for _, child := range t {
			if n, found := findJSONInt(child, keys...); found {
				return n, true
			}
		}
	}
	return 0, false
}

func findJSONMap(v any, key string) map[string]any {
	switch t := v.(type) {
	case map[string]any:
		if raw, ok := t[key]; ok {
			if m, ok := raw.(map[string]any); ok {
				return m
			}
		}
		for _, child := range t {
			if m := findJSONMap(child, key); m != nil {
				return m
			}
		}
	case []any:
		for _, child := range t {
			if m := findJSONMap(child, key); m != nil {
				return m
			}
		}
	}
	return nil
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
