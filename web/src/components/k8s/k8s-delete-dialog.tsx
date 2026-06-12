import { InputNumber, Modal, Select, Space, Typography } from "antd";
import { useEffect, useState } from "react";
import type { K8sDeleteOptions } from "../../services/service-factory";
import { pickK8sDeleteOpts } from "../../services/service-factory";

type K8sDeleteDialogProps = {
  open: boolean;
  resourceName: string;
  loading?: boolean;
  onCancel: () => void;
  onConfirm: (opts?: K8sDeleteOptions) => void | Promise<void>;
};

export function K8sDeleteDialog({ open, resourceName, loading, onCancel, onConfirm }: K8sDeleteDialogProps) {
  const [gracePeriod, setGracePeriod] = useState<number | undefined>();
  const [propagationPolicy, setPropagationPolicy] = useState<K8sDeleteOptions["propagation_policy"]>();

  useEffect(() => {
    if (open) {
      setGracePeriod(undefined);
      setPropagationPolicy(undefined);
    }
  }, [open, resourceName]);

  return (
    <Modal
      title={`确认删除 ${resourceName} 吗？`}
      open={open}
      okType="danger"
      okText="删除"
      cancelText="取消"
      confirmLoading={loading}
      destroyOnClose
      onCancel={onCancel}
      onOk={() => onConfirm(pickK8sDeleteOpts({ grace_period_seconds: gracePeriod, propagation_policy: propagationPolicy }))}
    >
      <Space direction="vertical" style={{ width: "100%" }} size="middle">
        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
          可选配置 Kubernetes DeleteOptions。留空则使用集群默认删除行为。
        </Typography.Paragraph>
        <div>
          <Typography.Text>优雅关闭时间（秒）</Typography.Text>
          <InputNumber
            style={{ width: "100%", marginTop: 6 }}
            min={0}
            placeholder="留空使用 Pod terminationGracePeriodSeconds（默认 30）"
            value={gracePeriod}
            onChange={(v) => setGracePeriod(v ?? undefined)}
          />
        </div>
        <div>
          <Typography.Text>级联删除策略</Typography.Text>
          <Select
            allowClear
            style={{ width: "100%", marginTop: 6 }}
            placeholder="留空使用 Kubernetes 默认策略"
            value={propagationPolicy}
            onChange={(v) => setPropagationPolicy(v)}
            options={[
              { label: "Background（后台级联）", value: "Background" },
              { label: "Foreground（前台级联）", value: "Foreground" },
              { label: "Orphan（孤立，不删子资源）", value: "Orphan" },
            ]}
          />
        </div>
      </Space>
    </Modal>
  );
}
