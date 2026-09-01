// @ts-nocheck
import { Button, Input, Space, Typography, message } from "antd";
import { useState } from "react";
import { extractApiErrorMessage } from "../../services/http";
import { generateK8sYAML } from "../../services/ai";

export type AiYamlGeneratePanelProps = {
  resourceKind: string;
  namespace?: string;
  clusterId?: number;
  /** 当前编辑器内容，作为生成提示 */
  hintYaml?: string;
  onGenerated: (yaml: string) => void;
};

/** YAML 创建区：自然语言描述 → AI 生成并回填编辑器 */
export function AiYamlGeneratePanel({
  resourceKind,
  namespace,
  clusterId,
  hintYaml,
  onGenerated,
}: AiYamlGeneratePanelProps) {
  const [desc, setDesc] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleGenerate() {
    const description = desc.trim();
    if (!description) {
      message.warning("请先描述要创建的资源需求");
      return;
    }
    setLoading(true);
    try {
      const res = await generateK8sYAML({
        resource_kind: resourceKind,
        namespace: namespace || undefined,
        cluster_id: clusterId,
        description,
        hint_yaml: hintYaml?.trim() || undefined,
      });
      onGenerated(res.yaml);
      message.success(`已生成 YAML（${res.provider}/${res.model}），请核对后创建`);
    } catch (e) {
      message.error(extractApiErrorMessage(e, "AI 生成 YAML 失败"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="small">
      <Typography.Text type="secondary">AI 生成：用自然语言描述需求，生成后写入下方编辑器，请人工核对再创建。</Typography.Text>
      <Input.TextArea
        value={desc}
        onChange={(e) => setDesc(e.target.value)}
        placeholder={`例如：创建 ${resourceKind}，镜像 nginx:1.25，2 副本，暴露 80 端口，命名空间 ${namespace || "default"}`}
        autoSize={{ minRows: 2, maxRows: 5 }}
        maxLength={2000}
        showCount
      />
      <Button type="default" loading={loading} onClick={() => void handleGenerate()}>
        AI 生成 YAML
      </Button>
    </Space>
  );
}
