package jenkins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client Jenkins REST API 客户端（buildWithParameters、构建状态、Console 日志）。
type Client struct {
	BaseURL    string
	Username   string
	APIToken   string
	JobFolder  string
	HTTPClient *http.Client
}

func NewClient(baseURL, username, apiToken, jobFolder string) *Client {
	return &Client{
		BaseURL:   strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Username:  strings.TrimSpace(username),
		APIToken:  strings.TrimSpace(apiToken),
		JobFolder: strings.Trim(strings.TrimSpace(jobFolder), "/"),
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTPClient != nil {
		return c.HTTPClient
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c *Client) jobBasePath(jobName string) string {
	jobName = strings.TrimSpace(jobName)
	if jobName == "" {
		return ""
	}
	parts := []string{"job", url.PathEscape(jobName)}
	if c.JobFolder != "" {
		parts = append([]string{"job", url.PathEscape(c.JobFolder)}, parts...)
	}
	return "/" + strings.Join(parts, "/")
}

func (c *Client) apiURL(path string) string {
	return c.BaseURL + path + "/api/json"
}

func (c *Client) setAuth(req *http.Request) {
	if c.Username != "" || c.APIToken != "" {
		req.SetBasicAuth(c.Username, c.APIToken)
	}
}

type crumbHolder struct {
	Crumb             string `json:"crumb"`
	CrumbRequestField string `json:"crumbRequestField"`
}

func (c *Client) fetchCrumb(ctx context.Context) (field, value string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL("/crumbIssuer"), nil)
	if err != nil {
		return "", "", err
	}
	c.setAuth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", "", nil
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", "", fmt.Errorf("crumb issuer: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var ch crumbHolder
	if err := json.NewDecoder(resp.Body).Decode(&ch); err != nil {
		return "", "", err
	}
	if ch.Crumb == "" {
		return "", "", nil
	}
	field = ch.CrumbRequestField
	if field == "" {
		field = "Jenkins-Crumb"
	}
	return field, ch.Crumb, nil
}

// BuildWithParameters 触发参数化构建，返回 queue item URL path。
func (c *Client) BuildWithParameters(ctx context.Context, jobName string, params map[string]string) (queueURL string, err error) {
	if c.BaseURL == "" {
		return "", fmt.Errorf("jenkins base url is empty")
	}
	jobPath := c.jobBasePath(jobName)
	if jobPath == "" {
		return "", fmt.Errorf("job name is empty")
	}
	form := url.Values{}
	for k, v := range params {
		if strings.TrimSpace(k) == "" {
			continue
		}
		form.Set(k, v)
	}
	endpoint := c.BaseURL + jobPath + "/buildWithParameters"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.setAuth(req)
	if field, crumb, err := c.fetchCrumb(ctx); err != nil {
		return "", err
	} else if field != "" && crumb != "" {
		req.Header.Set(field, crumb)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return "", fmt.Errorf("buildWithParameters HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", nil
	}
	if strings.HasPrefix(loc, "http") {
		u, err := url.Parse(loc)
		if err == nil {
			return u.Path, nil
		}
	}
	return loc, nil
}

// BuildInfo Jenkins 构建信息。
type BuildInfo struct {
	Number    int    `json:"number"`
	Result    string `json:"result"`
	Building  bool   `json:"building"`
	URL       string `json:"url"`
	Timestamp int64  `json:"timestamp"`
	Duration  int64  `json:"duration"`
}

// GetBuild 获取指定构建信息。
func (c *Client) GetBuild(ctx context.Context, jobName string, buildNumber int) (*BuildInfo, error) {
	jobPath := c.jobBasePath(jobName)
	endpoint := fmt.Sprintf("%s/%d/api/json", c.BaseURL+jobPath, buildNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("build not found")
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("get build HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var info BuildInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// GetLastBuildNumber 获取 Job 最近一次构建编号（无构建时返回 0）。
func (c *Client) GetLastBuildNumber(ctx context.Context, jobName string) (int, error) {
	jobPath := c.jobBasePath(jobName)
	endpoint := c.apiURL(jobPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?tree=lastBuild[number]", nil)
	if err != nil {
		return 0, err
	}
	c.setAuth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return 0, fmt.Errorf("get job HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		LastBuild *struct {
			Number int `json:"number"`
		} `json:"lastBuild"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	if payload.LastBuild == nil {
		return 0, nil
	}
	return payload.LastBuild.Number, nil
}

// JobExists 检查 Job 是否存在。
func (c *Client) JobExists(ctx context.Context, jobName string) (bool, error) {
	jobPath := c.jobBasePath(jobName)
	endpoint := c.apiURL(jobPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	c.setAuth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return false, fmt.Errorf("job exists HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return true, nil
}

// GetConsoleLog 获取构建 Console 输出（非 progressive，适合已完成构建）。
func (c *Client) GetConsoleLog(ctx context.Context, jobName string, buildNumber int) (string, error) {
	jobPath := c.jobBasePath(jobName)
	endpoint := fmt.Sprintf("%s/%d/consoleText", c.BaseURL+jobPath, buildNumber)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	c.setAuth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("console HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// BuildURL 返回 Jenkins 构建页 URL。
func (c *Client) BuildURL(jobName string, buildNumber int) string {
	if buildNumber <= 0 {
		return ""
	}
	return fmt.Sprintf("%s%s/%d/", c.BaseURL, c.jobBasePath(jobName), buildNumber)
}

// ResolveQueueBuildNumber 从 queue item 解析实际 build number（轮询最多 wait）。
func (c *Client) ResolveQueueBuildNumber(ctx context.Context, queuePath string, lastNumber int, wait time.Duration) (int, error) {
	if wait <= 0 {
		wait = 2 * time.Minute
	}
	deadline := time.Now().Add(wait)
	queuePath = strings.TrimSpace(queuePath)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		if queuePath != "" {
			endpoint := c.BaseURL + queuePath + "/api/json?tree=executable[number]"
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
			if err == nil {
				c.setAuth(req)
				resp, err := c.httpClient().Do(req)
				if err == nil {
					var payload struct {
						Executable *struct {
							Number int `json:"number"`
						} `json:"executable"`
					}
					_ = json.NewDecoder(resp.Body).Decode(&payload)
					resp.Body.Close()
					if payload.Executable != nil && payload.Executable.Number > 0 {
						return payload.Executable.Number, nil
					}
				}
			}
		}
		time.Sleep(2 * time.Second)
	}
	return lastNumber + 1, nil
}

func MapResultToStatus(result string, building bool) string {
	if building {
		return "running"
	}
	switch strings.ToUpper(strings.TrimSpace(result)) {
	case "SUCCESS":
		return "success"
	case "FAILURE":
		return "failure"
	case "ABORTED":
		return "aborted"
	case "UNSTABLE":
		return "failure"
	default:
		if result == "" {
			return "running"
		}
		return strings.ToLower(result)
	}
}

func splitFolderPath(folder string) []string {
	folder = strings.Trim(folder, "/")
	if folder == "" {
		return nil
	}
	parts := strings.Split(folder, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func folderBasePath(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		b.WriteString("/job/")
		b.WriteString(url.PathEscape(p))
	}
	return b.String()
}

// EnsureFolderPath 逐级创建 Jenkins Folder（已存在则跳过）。
func (c *Client) EnsureFolderPath(ctx context.Context, folderPath string) error {
	parts := splitFolderPath(folderPath)
	if len(parts) == 0 {
		return nil
	}
	parent := make([]string, 0, len(parts))
	for _, name := range parts {
		exists, err := c.folderExists(ctx, parent, name)
		if err != nil {
			return err
		}
		if !exists {
			if err := c.createFolder(ctx, parent, name); err != nil {
				return err
			}
		}
		parent = append(parent, name)
	}
	return nil
}

func (c *Client) folderExists(ctx context.Context, parentParts []string, name string) (bool, error) {
	path := folderBasePath(append(parentParts, name))
	endpoint := c.apiURL(path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	c.setAuth(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return false, fmt.Errorf("folder exists HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return true, nil
}

func (c *Client) createFolder(ctx context.Context, parentParts []string, name string) error {
	parentPath := folderBasePath(parentParts)
	endpoint := fmt.Sprintf("%s%s/createItem?name=%s", c.BaseURL, parentPath, url.QueryEscape(name))
	return c.postConfigXML(ctx, endpoint, FolderConfigXML())
}

// EnsurePipelineJob 在 Jenkins 创建或更新 Pipeline Job（Pipeline script from SCM）。
func (c *Client) EnsurePipelineJob(ctx context.Context, jobName, configXML string) error {
	jobName = strings.TrimSpace(jobName)
	if jobName == "" {
		return fmt.Errorf("job name is empty")
	}
	if err := c.EnsureFolderPath(ctx, c.JobFolder); err != nil {
		return fmt.Errorf("ensure jenkins folder: %w", err)
	}
	exists, err := c.JobExists(ctx, jobName)
	if err != nil {
		return err
	}
	if exists {
		return c.updateJobConfig(ctx, jobName, configXML)
	}
	return c.createJob(ctx, jobName, configXML)
}

func (c *Client) createJob(ctx context.Context, jobName, configXML string) error {
	parentPath := folderBasePath(splitFolderPath(c.JobFolder))
	endpoint := fmt.Sprintf("%s%s/createItem?name=%s", c.BaseURL, parentPath, url.QueryEscape(jobName))
	return c.postConfigXML(ctx, endpoint, configXML)
}

func (c *Client) updateJobConfig(ctx context.Context, jobName, configXML string) error {
	jobPath := c.jobBasePath(jobName)
	endpoint := c.BaseURL + jobPath + "/config.xml"
	return c.postConfigXML(ctx, endpoint, configXML)
}

func (c *Client) postConfigXML(ctx context.Context, endpoint, configXML string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(configXML))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/xml; charset=UTF-8")
	c.setAuth(req)
	if field, crumb, err := c.fetchCrumb(ctx); err != nil {
		return err
	} else if field != "" && crumb != "" {
		req.Header.Set(field, crumb)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return fmt.Errorf("jenkins config HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
