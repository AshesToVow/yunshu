/**
 * 告警配置中心：Webhook 联调演示入库体模板（RF-07 第一步拆分产物）
 *
 * 从 `alert-config-center-panel.tsx` 原地搬迁，字段与取值逐字保留。
 * 结构对齐 Alertmanager webhook 的 `{receiver, status, alerts[]}`，
 * 后端 `alert_event` 入库与路由匹配均依赖 `labels.severity` /
 * `labels.cluster` / `status` 三个字段，改这些示例值会直接影响
 * 「发送测试告警」的路由命中结果，请勿随手改成占位文本。
 *
 * key 与页面上的模板下拉选项一一对应：warning_prod / critical_prod / resolved_prod。
 */
export const webhookPayloadTemplates: Record<string, Record<string, unknown>> = {
  warning_prod: {
    receiver: "yunshu-webhook",
    status: "firing",
    alerts: [
      {
        status: "firing",
        labels: {
          alertname: "KubernetesPodUnhealthy",
          severity: "warning",
          cluster: "prodK8s",
          namespace: "default",
          pod: "demo-pod-1",
        },
        annotations: {
          summary: "Pod 异常（warning）",
          description: "演示告警：warning 路由",
        },
        startsAt: "2026-04-18T09:20:00Z",
        endsAt: "0001-01-01T00:00:00Z",
        generatorURL: "http://prometheus.example/graph?g0.expr=up",
        fingerprint: "demo-warning-prod-001",
      },
    ],
  },
  critical_prod: {
    receiver: "yunshu-webhook",
    status: "firing",
    alerts: [
      {
        status: "firing",
        labels: {
          alertname: "KubernetesNodeNotReady",
          severity: "critical",
          cluster: "prodK8s",
          namespace: "kube-system",
          node: "worker-1",
        },
        annotations: {
          summary: "节点不可用（critical）",
          description: "演示告警：critical 路由",
        },
        startsAt: "2026-04-18T09:21:00Z",
        endsAt: "0001-01-01T00:00:00Z",
        generatorURL: "http://prometheus.example/graph?g0.expr=node_ready",
        fingerprint: "demo-critical-prod-001",
      },
    ],
  },
  // 恢复通知：与 critical_prod 共用同一 fingerprint，用于联调「触发 → 恢复」闭环
  resolved_prod: {
    receiver: "yunshu-webhook",
    status: "resolved",
    alerts: [
      {
        status: "resolved",
        labels: {
          alertname: "KubernetesNodeNotReady",
          severity: "critical",
          cluster: "prodK8s",
          namespace: "kube-system",
          node: "worker-1",
        },
        annotations: {
          summary: "节点恢复（resolved）",
          description: "演示恢复通知",
        },
        startsAt: "2026-04-18T09:21:00Z",
        endsAt: "2026-04-18T09:25:00Z",
        generatorURL: "http://prometheus.example/graph?g0.expr=node_ready",
        fingerprint: "demo-critical-prod-001",
      },
    ],
  },
};
