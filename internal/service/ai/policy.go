package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

var protectedDeleteNamespaces = map[string]struct{}{
	"kube-system":     {},
	"kube-public":     {},
	"yunshu-logging":  {},
}

// checkWriteToolPolicy 在创建审批单前做硬策略校验；拒绝时返回明确错误，不创建审批。
func checkWriteToolPolicy(toolName, argsJSON, namespace, reason string) error {
	ns := strings.TrimSpace(namespace)
	tool := strings.TrimSpace(toolName)

	switch tool {
	case "restart_deployment", "scale_deployment":
		if ns == "" {
			return fmt.Errorf("策略拒绝: %s 要求 namespace 非空", tool)
		}
	case "delete_pod":
		if ns == "" {
			return fmt.Errorf("策略拒绝: delete_pod 要求 namespace 非空")
		}
		if _, ok := protectedDeleteNamespaces[ns]; ok {
			return fmt.Errorf("策略拒绝: 禁止在受保护命名空间 %s 删除 Pod", ns)
		}
	}

	if tool == "scale_deployment" {
		replicas := extractReplicas(argsJSON)
		if replicas > 50 {
			r := strings.ToLower(reason)
			if !strings.Contains(r, "emergency") {
				return fmt.Errorf("策略拒绝: replicas=%d 超过 50，须在 reason 中说明 emergency", replicas)
			}
		}
	}
	return nil
}

func extractReplicas(argsJSON string) int {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return 0
	}
	v, ok := args["replicas"]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}
