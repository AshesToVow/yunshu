package eventforward

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"yunshu/internal/pkg/outbound"
)

type WebhookClient struct {
	externalClient *http.Client
	internalClient *http.Client
	internalURL    string
	token          string
}

func NewWebhookClient(token string, timeout time.Duration) *WebhookClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &WebhookClient{
		externalClient: outbound.NewSafeClient(timeout),
		internalClient: &http.Client{Timeout: timeout},
		token:          strings.TrimSpace(token),
	}
}

// SetInternalURL 指定本机告警平台入站地址（服务端生成，允许回环）。
func (c *WebhookClient) SetInternalURL(raw string) {
	c.internalURL = strings.TrimSpace(raw)
}

func (c *WebhookClient) PostAlertmanager(ctx context.Context, url string, payload alertManagerPayload) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("webhook url is empty")
	}
	if len(payload.Alerts) == 0 {
		return nil
	}
	trusted := c.isInternalURL(url)
	if !trusted {
		if err := outbound.ValidatePublicHTTPURL(url); err != nil {
			return fmt.Errorf("webhook 地址不安全: %w", err)
		}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Webhook-Token", c.token)
		req.Header.Set("X-Alert-Token", c.token)
	}
	client := c.externalClient
	if trusted {
		client = c.internalClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

func (c *WebhookClient) isInternalURL(url string) bool {
	target := strings.TrimSpace(c.internalURL)
	return target != "" && strings.TrimSpace(url) == target
}

// validateWebhookTarget 校验规则中的 webhook_url；空值/internal/alertmanager 表示走本机告警入站。
func validateWebhookTarget(raw string) error {
	u := strings.TrimSpace(raw)
	if u == "" || strings.EqualFold(u, "internal") || strings.EqualFold(u, "alertmanager") {
		return nil
	}
	return outbound.ValidatePublicHTTPURL(u)
}
