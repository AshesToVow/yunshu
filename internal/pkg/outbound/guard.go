// Package outbound 提供出站 HTTP 的 SSRF 防护：URL 白名单校验 + 拨号期 IP 校验。
// 拨号期校验可防止 DNS rebinding（校验时解析到公网、连接时解析到内网）。
package outbound

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// ValidatePublicHTTPURL 校验用户可控的出站 URL：仅 http/https，禁止本机、私网、
// 链路本地、云元数据地址；解析失败一律拒绝。
func ValidatePublicHTTPURL(raw string) error {
	u, host, err := parseHTTPURL(raw)
	if err != nil {
		return err
	}
	_ = u
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") || isMetadataHost(lower) {
		return fmt.Errorf("禁止访问内网或元数据地址: %s", host)
	}
	if ip := net.ParseIP(lower); ip != nil {
		if IsBlockedIP(ip) {
			return fmt.Errorf("禁止访问内网或保留地址: %s", ip.String())
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip", host)
	if err != nil {
		return fmt.Errorf("无法解析目标主机 %s: %w", host, err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("无法解析目标主机 %s", host)
	}
	for _, ip := range ips {
		if IsBlockedIP(ip) {
			return fmt.Errorf("禁止访问内网或保留地址: %s", ip.String())
		}
	}
	return nil
}

// ValidateClusterEndpointURL 校验集群 API Server 地址：允许 RFC1918 私网（自建集群常见），
// 但拒绝非 http/https、回环、链路本地与云元数据地址。
func ValidateClusterEndpointURL(raw string) error {
	_, host, err := parseHTTPURL(raw)
	if err != nil {
		return err
	}
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".localhost") || isMetadataHost(lower) {
		return fmt.Errorf("禁止使用本机或元数据地址作为 API Server: %s", host)
	}
	if ip := net.ParseIP(lower); ip != nil {
		if IsLocalOrLinkLocalIP(ip) {
			return fmt.Errorf("禁止使用本机或链路本地地址作为 API Server: %s", ip.String())
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIP(context.Background(), "ip", host)
	if err != nil {
		// 域名可能仅在集群内部 DNS 可解析，此时放行由实际连接时报错。
		return nil
	}
	for _, ip := range ips {
		if IsLocalOrLinkLocalIP(ip) {
			return fmt.Errorf("禁止使用本机或链路本地地址作为 API Server: %s", ip.String())
		}
	}
	return nil
}

func parseHTTPURL(raw string) (*url.URL, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, "", fmt.Errorf("URL 为空")
	}
	if strings.ContainsAny(raw, "\r\n\t") {
		return nil, "", fmt.Errorf("URL 含非法控制字符")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, "", fmt.Errorf("URL 无效: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, "", fmt.Errorf("仅允许 http/https，当前为 %q", u.Scheme)
	}
	if u.User != nil {
		return nil, "", fmt.Errorf("URL 不允许携带用户信息")
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return nil, "", fmt.Errorf("URL 缺少主机名")
	}
	return u, host, nil
}

func isMetadataHost(host string) bool {
	switch host {
	case "metadata.google.internal", "metadata.goog", "instance-data", "metadata.azure.internal":
		return true
	}
	return false
}

// IsBlockedIP 判断出站目标是否属于必须拒绝的地址（本机、私网、链路本地、未指定、云元数据）。
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
	}
	return false
}

// IsLocalOrLinkLocalIP 仅拒绝本机与链路本地/未指定地址，保留私网可用。
func IsLocalOrLinkLocalIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

// NewSafeClient 返回带拨号期 IP 校验与重定向校验的 HTTP 客户端。
func NewSafeClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	tr, ok := http.DefaultTransport.(*http.Transport)
	var transport http.RoundTripper
	if ok {
		clone := tr.Clone()
		clone.DialContext = newGuardedDialer().DialContext
		clone.TLSHandshakeTimeout = 10 * time.Second
		clone.ResponseHeaderTimeout = 30 * time.Second
		transport = clone
	} else {
		transport = &http.Transport{DialContext: newGuardedDialer().DialContext}
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("重定向次数过多")
			}
			return ValidatePublicHTTPURL(req.URL.String())
		},
	}
}

func newGuardedDialer() *net.Dialer {
	return &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(network, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return fmt.Errorf("非法的目标地址 %s: %w", address, err)
			}
			ip := net.ParseIP(strings.Trim(host, "[]"))
			if ip == nil {
				return fmt.Errorf("无法解析目标地址: %s", host)
			}
			if IsBlockedIP(ip) {
				return fmt.Errorf("禁止访问内网或保留地址: %s", ip.String())
			}
			return nil
		},
	}
}
