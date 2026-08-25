package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"yunshu/internal/interfaces"
	"yunshu/internal/model"
	"log/slog"

	"gorm.io/gorm"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// getDirectConfigFromDict 从数据字典读取直连配置
func getDirectConfigFromDict(ctx context.Context, dictRepo interfaces.DictEntryRepository, configKey string) (*DirectConfig, error) {
	// 优先按 label（配置键）查；兼容历史“按 value 作为键”查法。
	entry, err := dictRepo.GetByDictTypeAndLabel(ctx, "k8s_direct_config", configKey)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		entry, err = dictRepo.GetByDictTypeAndValue(ctx, "k8s_direct_config", configKey)
	}
	if err != nil {
		return nil, fmt.Errorf("获取数据字典配置失败: %w", err)
	}

	var config DirectConfig
	if err := json.Unmarshal([]byte(entry.Value), &config); err != nil {
		return nil, fmt.Errorf("解析配置JSON失败: %w", err)
	}

	return &config, nil
}

// mergeDirectConfig 合并字典配置和用户配置（用户配置优先）
func mergeDirectConfig(base, override *DirectConfig) {
	if override.Server != "" {
		base.Server = override.Server
	}
	if override.CAData != "" {
		base.CAData = override.CAData
	}
	if override.Token != "" {
		base.Token = override.Token
	}
	if override.Username != "" {
		base.Username = override.Username
	}
	if override.Password != "" {
		base.Password = override.Password
	}
	if override.ClientCertData != "" {
		base.ClientCertData = override.ClientCertData
	}
	if override.ClientKeyData != "" {
		base.ClientKeyData = override.ClientKeyData
	}
	base.InsecureSkipTLSVerify = override.InsecureSkipTLSVerify || base.InsecureSkipTLSVerify
}

// preserveDirectAuthFromStored 编辑集群时前端常不回传已保存的 token/密码/证书（JSON 省略或空串），
// 若直接生成 kubeconfig 会丢失 Bearer，表现为：心跳能拿到版本（部分集群允许匿名 /version）、List Namespace 等返回 401 Unauthorized。
func preserveDirectAuthFromStored(storedJSON string, next *DirectConfig) {
	if next == nil || strings.TrimSpace(storedJSON) == "" {
		return
	}
	var prev DirectConfig
	if err := json.Unmarshal([]byte(storedJSON), &prev); err != nil {
		slog.Default().With("component", "k8s.cluster").Warn("preserve direct auth: stored config unmarshal failed", "error", err)
		return
	}
	if shouldPreserveSecret(next.Token, prev.Token) {
		next.Token = prev.Token
	}
	if shouldPreserveSecret(next.Password, prev.Password) {
		next.Password = prev.Password
	}
	if shouldPreserveSecret(next.ClientCertData, prev.ClientCertData) {
		next.ClientCertData = prev.ClientCertData
	}
	if shouldPreserveSecret(next.ClientKeyData, prev.ClientKeyData) {
		next.ClientKeyData = prev.ClientKeyData
	}
	if shouldPreserveSecret(next.CAData, prev.CAData) {
		next.CAData = prev.CAData
	}
}

func shouldPreserveSecret(incoming, stored string) bool {
	in := strings.TrimSpace(incoming)
	if in == "" || isMaskedSecretValue(in) {
		return strings.TrimSpace(stored) != ""
	}
	return false
}

func isMaskedSecretValue(s string) bool {
	return strings.Contains(s, "***")
}

// resolveClusterKubeconfig 解析集群连接配置：直连模式从 direct_config 实时生成（保证 Token 与库内 JSON 一致）。
// 库内字段可能为 AES-GCM 密文，需经 OpenCredential 解密（无 runtime 时仅兼容明文）。
func resolveClusterKubeconfig(cluster *model.K8sCluster) (string, error) {
	return resolveClusterKubeconfigWithOpener(cluster, nil)
}

func (s *K8sRuntimeService) resolveClusterKubeconfig(cluster *model.K8sCluster) (string, error) {
	return resolveClusterKubeconfigWithOpener(cluster, s)
}

type clusterSecretOpener interface {
	OpenCredential(stored string) (string, error)
}

func resolveClusterKubeconfigWithOpener(cluster *model.K8sCluster, opener clusterSecretOpener) (string, error) {
	if cluster == nil {
		return "", fmt.Errorf("cluster is nil")
	}
	open := func(stored string) (string, error) {
		if opener != nil {
			return opener.OpenCredential(stored)
		}
		return stored, nil
	}
	mode := strings.TrimSpace(cluster.ConnectionMode)
	if mode == "direct" && strings.TrimSpace(cluster.DirectConfig) != "" {
		raw, err := open(cluster.DirectConfig)
		if err != nil {
			return "", fmt.Errorf("解密直连配置失败: %w", err)
		}
		var dc DirectConfig
		if err := json.Unmarshal([]byte(raw), &dc); err != nil {
			return "", fmt.Errorf("解析直连配置失败: %w", err)
		}
		return buildKubeconfigFromDirectConfig(&dc)
	}
	kcStored, err := open(cluster.Kubeconfig)
	if err != nil {
		return "", fmt.Errorf("解密 kubeconfig 失败: %w", err)
	}
	kc := strings.TrimSpace(kcStored)
	if kc == "" {
		return "", fmt.Errorf("集群 kubeconfig 为空")
	}
	return normalizeKubeconfigForClientGo(kc)
}

// normalizeKubeconfigForClientGo 修正 kubeconfig 中与 client-go 冲突的组合。
// 常见场景：同时存在 certificate-authority-data 与 insecure-skip-tls-verify，
// client-go 会报错 "specifying a root certificates file with the insecure flag is not allowed"。
func normalizeKubeconfigForClientGo(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("kubeconfig 为空")
	}
	cfg, err := clientcmd.Load([]byte(raw))
	if err != nil {
		return raw, err
	}
	changed := false
	for _, cluster := range cfg.Clusters {
		if cluster == nil || !cluster.InsecureSkipTLSVerify {
			continue
		}
		if len(cluster.CertificateAuthorityData) > 0 || strings.TrimSpace(cluster.CertificateAuthority) != "" {
			cluster.CertificateAuthorityData = nil
			cluster.CertificateAuthority = ""
			changed = true
		}
	}
	if !changed {
		return raw, nil
	}
	out, err := clientcmd.Write(*cfg)
	if err != nil {
		return raw, err
	}
	return string(out), nil
}

func maskDirectConfigForAPI(dc *DirectConfig) *DirectConfig {
	if dc == nil {
		return nil
	}
	out := *dc
	if strings.TrimSpace(out.Token) != "" {
		out.Token = maskSecretEdge(out.Token, 4)
	}
	if strings.TrimSpace(out.Password) != "" {
		out.Password = maskSecretEdge(out.Password, 0)
	}
	if strings.TrimSpace(out.ClientCertData) != "" {
		out.ClientCertData = maskSecretEdge(out.ClientCertData, 6)
	}
	if strings.TrimSpace(out.ClientKeyData) != "" {
		out.ClientKeyData = maskSecretEdge(out.ClientKeyData, 6)
	}
	if strings.TrimSpace(out.CAData) != "" {
		out.CAData = maskSecretEdge(out.CAData, 6)
	}
	return &out
}

// buildKubeconfigFromDirectConfig 从直连配置生成kubeconfig
func buildKubeconfigFromDirectConfig(config *DirectConfig) (string, error) {
	serverRaw := strings.TrimSpace(config.Server)
	if serverRaw == "" {
		return "", fmt.Errorf("API Server 地址不能为空")
	}
	// 兜底清洗：UI/复制粘贴可能混入空白字符，导致 token/base64 校验失败。
	token := compactNoSpace(config.Token)
	username := strings.TrimSpace(config.Username)
	password := strings.TrimSpace(config.Password)
	caRaw := compactNoSpace(config.CAData)
	certRaw := compactNoSpace(config.ClientCertData)
	keyRaw := compactNoSpace(config.ClientKeyData)
	// 解析服务器地址
	serverURL, err := url.Parse(serverRaw)
	if err != nil {
		return "", fmt.Errorf("无效的服务器地址: %w", err)
	}
	if serverURL.Scheme == "" || serverURL.Host == "" {
		return "", fmt.Errorf("无效的服务器地址: 需要完整 URL（如 https://10.0.0.1:6443）")
	}

	// 构建REST配置
	restConfig := &rest.Config{
		Host: serverRaw,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: config.InsecureSkipTLSVerify,
		},
	}

	// 设置CA证书。若已开启 insecure，client-go 不允许同时带 root CA。
	if !config.InsecureSkipTLSVerify && caRaw != "" {
		caData, err := base64.StdEncoding.DecodeString(caRaw)
		if err != nil {
			return "", fmt.Errorf("CA证书解码失败: %w", err)
		}
		restConfig.CAData = caData
	}

	// 设置认证方式
	if token != "" {
		restConfig.BearerToken = token
	} else if username != "" && password != "" {
		restConfig.Username = username
		restConfig.Password = password
	} else if certRaw != "" && keyRaw != "" {
		certData, err := base64.StdEncoding.DecodeString(certRaw)
		if err != nil {
			return "", fmt.Errorf("客户端证书解码失败: %w", err)
		}
		keyData, err := base64.StdEncoding.DecodeString(keyRaw)
		if err != nil {
			return "", fmt.Errorf("客户端密钥解码失败: %w", err)
		}
		restConfig.CertData = certData
		restConfig.KeyData = keyData
	}

	hasAuth := strings.TrimSpace(restConfig.BearerToken) != "" ||
		(strings.TrimSpace(restConfig.Username) != "" && strings.TrimSpace(restConfig.Password) != "") ||
		(len(restConfig.CertData) > 0 && len(restConfig.KeyData) > 0)
	if !hasAuth {
		return "", fmt.Errorf("直连未配置有效认证：请填写 Token、或用户名+密码、或客户端证书+私钥（部分集群匿名可读版本信息，但无法列举命名空间）")
	}
	if !config.InsecureSkipTLSVerify && caRaw == "" {
		return "", fmt.Errorf("直连未配置 CA：请粘贴集群根 CA（如 /etc/kubernetes/pki/ca.crt 的 base64），或开启「跳过 TLS 验证」；勿使用 ServiceAccount Secret 内 ca.crt（常与 APIServer 证书不匹配）")
	}

	// 将REST配置转换为kubeconfig格式
	kubeconfig := generateKubeconfigYAML(restConfig, serverURL.Hostname())
	return kubeconfig, nil
}

// classifyClusterConnectError 将 TLS/认证等连通错误转成可读提示（供状态探测展示）。
func classifyClusterConnectError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "x509") || strings.Contains(low, "certificate signed by unknown authority") || strings.Contains(low, "tls:"):
		return "TLS 证书校验失败：请填写正确的集群根 CA（kubeadm: /etc/kubernetes/pki/ca.crt），或开启「跳过 TLS 验证」。不要使用 SA Secret 里的 ca.crt。"
	case strings.Contains(low, "unauthorized") || strings.Contains(low, "401"):
		return "集群认证失败：请检查 Token 是否为解码后的 JWT（一般以 eyJ 开头），且对应 ServiceAccount 仍有效。"
	case strings.Contains(low, "forbidden") || strings.Contains(low, "403"):
		return "集群拒绝访问：入库凭证缺少 list namespaces 等权限，请为该凭证绑定足够 RBAC。"
	case strings.Contains(low, "timeout") || strings.Contains(low, "i/o timeout") || strings.Contains(low, "deadline exceeded"):
		return "连接超时：请确认 Yunshu 后端所在网络能访问 API Server 地址与端口。"
	case strings.Contains(low, "connection refused") || strings.Contains(low, "no such host") || strings.Contains(low, "dial tcp"):
		return "无法连通 API Server：请检查地址、端口与网络策略。"
	default:
		if len(msg) > 300 {
			return msg[:300] + "…"
		}
		return msg
	}
}

func compactNoSpace(s string) string {
	parts := strings.Fields(strings.TrimSpace(s))
	return strings.Join(parts, "")
}

func maskSecretEdge(s string, edge int) string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return ""
	}
	if edge <= 0 || len(raw) <= edge*2 {
		return fmt.Sprintf("***len=%d***", len(raw))
	}
	return raw[:edge] + "..." + raw[len(raw)-edge:]
}

// generateKubeconfigYAML 生成kubeconfig YAML格式
func generateKubeconfigYAML(config *rest.Config, clusterName string) string {
	var caData, certData, keyData string
	if len(config.CAData) > 0 {
		caData = base64.StdEncoding.EncodeToString(config.CAData)
	}
	if len(config.CertData) > 0 {
		certData = base64.StdEncoding.EncodeToString(config.CertData)
	}
	if len(config.KeyData) > 0 {
		keyData = base64.StdEncoding.EncodeToString(config.KeyData)
	}

	// 构建YAML
	yaml := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
`, config.Host)

	if caData != "" && !config.TLSClientConfig.Insecure {
		yaml += fmt.Sprintf("    certificate-authority-data: %s\n", caData)
	}
	if config.TLSClientConfig.Insecure {
		yaml += "    insecure-skip-tls-verify: true\n"
	}

	yaml += fmt.Sprintf("  name: %s\n", clusterName)

	// Contexts
	yaml += fmt.Sprintf(`contexts:
- context:
    cluster: %s
    user: %s-user
  name: %s-context
current-context: %s-context
`, clusterName, clusterName, clusterName, clusterName)

	// Users
	yaml += fmt.Sprintf("users:\n- name: %s-user\n", clusterName)

	if config.BearerToken != "" {
		yaml += fmt.Sprintf("  user:\n    token: %s\n", config.BearerToken)
	} else if config.Username != "" {
		yaml += fmt.Sprintf("  user:\n    username: %s\n    password: %s\n", config.Username, config.Password)
	} else if certData != "" {
		yaml += fmt.Sprintf("  user:\n    client-certificate-data: %s\n    client-key-data: %s\n", certData, keyData)
	}

	return yaml
}
