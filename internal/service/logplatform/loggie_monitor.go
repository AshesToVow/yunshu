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

	var parsed loggieHelpLogResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return LoggieMonitorProbeResult{
			Reachable:  true,
			DetailJSON: truncateString(string(body), 2048),
			Error:      "parse help/log json failed",
		}
	}

	names := make([]string, 0, len(parsed.FileStatus.Pipeline))
	for name := range parsed.FileStatus.Pipeline {
		names = append(names, name)
	}

	detail, _ := json.Marshal(map[string]any{
		"active_fd":   parsed.FdStatus.ActiveFdCount,
		"inactive_fd": parsed.FdStatus.InActiveFdCount,
		"pipelines":   names,
	})

	return LoggieMonitorProbeResult{
		Reachable:           true,
		ActiveFdCount:       parsed.FdStatus.ActiveFdCount,
		InActiveFdCount:     parsed.FdStatus.InActiveFdCount,
		ActivePipelineCount: len(names),
		PipelineNames:       names,
		DetailJSON:          string(detail),
	}
}

func truncateString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
