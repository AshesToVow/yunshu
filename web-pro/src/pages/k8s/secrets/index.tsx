// @ts-nocheck
import { Alert, Button, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef, useState } from "react";
import { SecretFormCreateDrawer } from "@/components/k8s/k8s-resource-form-drawers";
import { YamlCrudPage } from "@/components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "@/services/clusters";
import {
  applySecret,
  deleteSecret,
  getSecretDetail,
  listSecrets,
  revealSecret,
  type ConfigDetail,
  type SecretItem,
} from "@/services/configs";
import { extractApiErrorMessage } from "@/services/http";

export default function SecretsPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const [revealed, setRevealed] = useState<ConfigDetail | null>(null);
  const [revealCtx, setRevealCtx] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);

  const columns: ColumnsType<SecretItem> = [
    { title: "名称", dataIndex: "name" },
    { title: "类型", dataIndex: "type", width: 180 },
    { title: "键数量", dataIndex: "data_count", width: 120 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<SecretItem, ConfigDetail>
        title="Secret 管理"
        needNamespace
        watchResource="secrets"
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        columns={columns}
        onToolbarReady={(ctx) => {
          listReloadRef.current = ctx.reload;
        }}
        renderCreateFormTab={(ctx) => (
          <SecretFormCreateDrawer
            embedded
            open
            clusterId={ctx.clusterId}
            namespace={ctx.namespace ?? "default"}
            onClose={ctx.closeCreateDrawer}
            onSuccess={() => {
              listReloadRef.current();
              ctx.closeCreateDrawer();
            }}
          />
        )}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listSecrets(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => {
            setRevealed(null);
            setRevealCtx({ clusterId, namespace: namespace ?? "default", name });
            return await getSecretDetail(clusterId, namespace ?? "default", name);
          },
          apply: async ({ clusterId, manifest }) => await applySecret(clusterId, manifest),
          remove: async (args) => await deleteSecret(args.clusterId, args.namespace ?? "default", args.name, args),
        }}
        createTemplate={({ namespace }) => `apiVersion: v1
kind: Secret
metadata:
  name: example-secret
  namespace: ${namespace || "default"}
type: Opaque
stringData:
  username: admin
  password: Admin@123
`}
        detailExtra={(d) => {
          const view = revealed ?? d;
          return (
            <div>
              <Alert
                type="warning"
                showIcon
                message="注意"
                description={
                  d.redacted
                    ? "默认已脱敏。查看明文需点击「揭示明文」（操作将记入审计且响应体脱敏）。"
                    : "Secret 含敏感信息，请勿截图或外传。"
                }
              />
              {d.redacted && revealCtx ? (
                <Button
                  style={{ marginTop: 12 }}
                  danger
                  onClick={() => {
                    void revealSecret(revealCtx.clusterId, revealCtx.namespace, revealCtx.name)
                      .then((r) => {
                        setRevealed(r);
                        message.success("已揭示明文（请谨慎使用）");
                      })
                      .catch((e) => message.error(extractApiErrorMessage(e, "揭示失败")));
                  }}
                >
                  揭示明文
                </Button>
              ) : null}
              {view.decoded_data ? (
                <div style={{ marginTop: 12 }}>
                  <Typography.Title level={5} style={{ marginTop: 0 }}>
                    decoded_data{view.redacted ? "（已脱敏）" : ""}
                  </Typography.Title>
                  <Typography.Paragraph style={{ whiteSpace: "pre-wrap" }}>
                    {Object.entries(view.decoded_data)
                      .map(([k, v]) => `${k}: ${v}`)
                      .join("\n")}
                  </Typography.Paragraph>
                </div>
              ) : null}
            </div>
          );
        }}
      />
    </>
  );
}
