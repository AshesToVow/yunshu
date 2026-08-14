package alert

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

// assertSafeOutboundURL 阻止告警渠道出站请求打到内网/本机/链路本地，降低 SSRF 风险。
func assertSafeOutboundURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fmt.Errorf("webhook URL 为空")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("webhook URL 无效: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("webhook 仅允许 http/https")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return fmt.Errorf("webhook 缺少主机名")
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") || lower == "metadata.google.internal" {
		return fmt.Errorf("禁止访问内网或元数据地址")
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		// 无法解析时拒绝，避免 DNS rebinding 绕过（解析失败也不放行）。
		return fmt.Errorf("无法解析 webhook 主机: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("无法解析 webhook 主机")
	}
	for _, ip := range ips {
		if isBlockedOutboundIP(ip) {
			return fmt.Errorf("禁止访问内网或保留地址: %s", ip.String())
		}
	}
	return nil
}

func isBlockedOutboundIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	// 运维平台常见出站到内网 Webhook，故不封 RFC1918；仍禁止本机与链路本地/云元数据。
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}
