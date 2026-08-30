import { CloudDownloadOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
} from "antd";
import type { FormInstance } from "antd/es/form";
import type { ClusterItem } from "../../services/clusters";
import type { CicdDeployConfig, CicdServiceItem } from "../../services/cicd";
import { K8S_DEPLOY_CONFIG_TYPES, K8S_DEPLOY_TEMPLATES } from "./release-ops";

type DictOption = { label: string; value: string | number };

type Props = {
  open: boolean;
  deployKind: "regular" | "container";
  deployStep: number;
  editingDeployConfig: CicdDeployConfig | null;
  deployService: CicdServiceItem | null;
  form: FormInstance;
  tenvOpts: DictOption[];
  usedTenvSet: Set<string>;
  importanceLevels: DictOption[];
  deployActions: DictOption[];
  deployMethodWatch: string | undefined;
  clusters: ClusterItem[];
  helmScaffoldLoading: boolean;
  serverOptions: Array<{ label: string; value: number }>;
  serverLabelById: Map<number, string>;
  startScriptTypes: DictOption[];
  onCancel: () => void;
  onPrevStep: () => void;
  onNextStep: () => void | Promise<void>;
  onSubmit: () => void | Promise<void>;
  onDownloadHelmScaffold: () => void | Promise<void>;
};

export function DeployConfigModal({
  open,
  deployKind,
  deployStep,
  editingDeployConfig,
  deployService,
  form,
  tenvOpts,
  usedTenvSet,
  importanceLevels,
  deployActions,
  deployMethodWatch,
  clusters,
  helmScaffoldLoading,
  serverOptions,
  serverLabelById,
  startScriptTypes,
  onCancel,
  onPrevStep,
  onNextStep,
  onSubmit,
  onDownloadHelmScaffold,
}: Props) {
  const isFrontend = deployService?.service_type === "frontend";

  return (
    <Modal
      title={
        editingDeployConfig
          ? `编辑${deployKind === "regular" ? "常规" : "容器化"}发布配置`
          : deployKind === "regular"
            ? "新增常规发布配置"
            : "容器化发布配置"
      }
      open={open}
      width={640}
      onCancel={onCancel}
      footer={
        <Space>
          <Button onClick={onCancel}>取消</Button>
          {deployStep > 0 && <Button onClick={onPrevStep}>上一步</Button>}
          {deployStep < (deployKind === "regular" ? 1 : 1) ? (
            <Button type="primary" onClick={() => void onNextStep()}>
              下一步
            </Button>
          ) : (
            <Button type="primary" onClick={() => void onSubmit()}>
              {editingDeployConfig ? "保存" : "确认"}
            </Button>
          )}
        </Space>
      }
    >
      <Form form={form} layout="vertical">
        <div style={{ display: deployStep === 0 ? "block" : "none" }}>
          <Alert type="warning" showIcon message="新建发布配置将按已录入的应用生成新的发布配置信息！" style={{ marginBottom: 16 }} />
          <Form.Item name="name" label="配置名称" rules={[{ required: true, message: "请填写配置名称" }]}>
            <Input placeholder="如 k8s-demo-生产环境" />
          </Form.Item>
          <Form.Item name="tenv" label="发布环境" rules={[{ required: true }]} extra="同一应用下同类型发布配置每个环境仅允许一条">
            <Select
              options={tenvOpts.map((o) => ({
                label: usedTenvSet.has(String(o.value)) ? `${o.label}（已配置）` : o.label,
                value: o.value,
                disabled: usedTenvSet.has(String(o.value)),
              }))}
            />
          </Form.Item>
          <Form.Item name="audit_enabled" label="发布审核" valuePropName="checked">
            <Switch checkedChildren="开启" unCheckedChildren="关闭" />
          </Form.Item>
          <Form.Item name="importance" label="重要级别">
            <Select allowClear options={importanceLevels.map((o) => ({ label: o.label, value: o.value }))} />
          </Form.Item>
          {deployKind === "regular" ? (
            <Form.Item name="server_port" label="服务端口">
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
          ) : (
            <>
              <Form.Item name="deploy_action" label="默认操作类型" extra="发布时可覆盖">
                <Select options={deployActions.map((o) => ({ label: o.label, value: o.value }))} />
              </Form.Item>
            </>
          )}
        </div>
        {deployKind === "container" && (
          <div style={{ display: deployStep === 1 ? "block" : "none" }}>
            <Alert type="info" showIcon message="配置 K8s 集群、命名空间与镜像信息" style={{ marginBottom: 16 }} />
            <Form.Item name="k8s_cluster_id" label="K8s 集群" rules={[{ required: true, message: "请选择集群" }]}>
              <Select
                showSearch
                optionFilterProp="label"
                options={clusters.map((c) => ({ label: c.name, value: c.id }))}
                placeholder="选择已接入的 K8s 集群"
              />
            </Form.Item>
            <Form.Item name="k8s_namespace" label="Namespace" rules={[{ required: true, message: "请填写命名空间" }]}>
              <Input placeholder="default" />
            </Form.Item>
            <Form.Item name="deploy_method" label="部署方式" rules={[{ required: true }]}>
              <Select options={[{ label: "kubectl", value: "kubectl" }, { label: "helm", value: "helm" }]} />
            </Form.Item>
            {deployMethodWatch === "helm" ? (
              <Alert
                type="success"
                showIcon
                style={{ marginBottom: 16 }}
                message="Helm 部署：已对齐「Application + base charts」目录架构"
                description={
                  <Space direction="vertical" size={8} style={{ width: "100%" }}>
                    <Typography.Text type="secondary">
                      下载 zip 解压到仓库根目录，得到 <Typography.Text code>helm/</Typography.Text>
                      （含 charts/deployment-base 等公共模块、config-files、多环境 values）与可选{" "}
                      <Typography.Text code>setup/</Typography.Text>
                      。研发只改 values.yaml；Jenkins 使用 <Typography.Text code>helm/Chart.yaml</Typography.Text>
                      。默认写入 Consul 注册：注解 <Typography.Text code>consul.register/enabled=true</Typography.Text>
                      、<Typography.Text code>consul.register/service.name</Typography.Text>
                      与标签 <Typography.Text code>yunshu-metrics: tag</Typography.Text>
                      （不需要时在 values 里关 <Typography.Text code>deployment-base.consulRegister.enabled</Typography.Text>）。
                    </Typography.Text>
                    <Button
                      type="primary"
                      icon={<CloudDownloadOutlined />}
                      loading={helmScaffoldLoading}
                      onClick={() => void onDownloadHelmScaffold()}
                    >
                      下载 Helm 脚手架
                    </Button>
                  </Space>
                }
              />
            ) : null}
            {deployMethodWatch !== "helm" ? (
              <>
                <Form.Item name="deploy_config_type" label="工作负载类型" rules={[{ required: true }]}>
                  <Select options={K8S_DEPLOY_CONFIG_TYPES.map((o) => ({ label: o.label, value: o.value }))} />
                </Form.Item>
                <Form.Item name="deploy_config_template" label="部署模板" rules={[{ required: true }]} extra="共享库 k8s-basic / k8s-skywalking 的 Pod 模板须含 Consul 必填项：consul.register/enabled、service.name、标签 yunshu-metrics=tag">
                  <Select options={K8S_DEPLOY_TEMPLATES.map((o) => ({ label: o.label, value: o.value }))} />
                </Form.Item>
              </>
            ) : (
              <>
                <Form.Item name="deploy_config_type" hidden>
                  <Input />
                </Form.Item>
                <Form.Item name="deploy_config_template" hidden>
                  <Input />
                </Form.Item>
              </>
            )}
            <Form.Item name="replicas" label="副本数" rules={[{ required: true }]}>
              <InputNumber min={1} max={100} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item
              name="deploy_strategy"
              label="发布策略"
              rules={[{ required: true }]}
              extra="金丝雀/蓝绿：Jenkins 接收 deployStrategy 等参数；平台侧可在发布详情中晋级/中止"
            >
              <Select
                options={[
                  { label: "滚动发布", value: "rolling" },
                  { label: "金丝雀发布", value: "canary" },
                  { label: "蓝绿发布", value: "blue_green" },
                ]}
              />
            </Form.Item>
            <Form.Item noStyle shouldUpdate={(prev, cur) => prev.deploy_strategy !== cur.deploy_strategy}>
              {({ getFieldValue }) => {
                const strategy = getFieldValue("deploy_strategy");
                if (strategy === "canary") {
                  return (
                    <>
                      <Form.Item name="canary_replicas" label="金丝雀初始副本" rules={[{ required: true }]}>
                        <InputNumber min={1} max={100} style={{ width: "100%" }} />
                      </Form.Item>
                      <Form.Item name="canary_percent" label="金丝雀流量占比(%)" rules={[{ required: true }]}>
                        <InputNumber min={1} max={100} style={{ width: "100%" }} />
                      </Form.Item>
                      <Form.Item
                        name="canary_steps_json"
                        label="晋级步骤(%)"
                        rules={[{ required: true }]}
                        extra="逗号分隔，如 10,50,100"
                      >
                        <Input placeholder="10,50,100" />
                      </Form.Item>
                    </>
                  );
                }
                if (strategy === "blue_green") {
                  return (
                    <Form.Item
                      name="blue_green_service"
                      label="蓝绿 Service 名"
                      extra="留空则使用工作负载名；切换 selector 标签 yunshu.io/color"
                    >
                      <Input placeholder="可选，默认与工作负载同名" />
                    </Form.Item>
                  );
                }
                return null;
              }}
            </Form.Item>
            <Form.Item name="container_port" label="容器端口" rules={[{ required: true }]}>
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item name="image_name" label="镜像名" rules={[{ required: true }]} extra="Harbor 仓库中的镜像名称">
              <Input />
            </Form.Item>
            <Form.Item name="image_tag" label="默认镜像 Tag">
              <Input placeholder="latest（CI 构建时会追加时间戳）" />
            </Form.Item>
          </div>
        )}
        {deployKind === "regular" && (
          <div style={{ display: deployStep === 1 ? "block" : "none" }}>
            <Alert type="warning" showIcon message="需要选择部署的目标主机及部署路径" style={{ marginBottom: 16 }} />
            <Form.Item name="dest_path" label="部署路径" rules={[{ required: true }]}>
              <Input placeholder="/export/icity/app-name" />
            </Form.Item>
            <Form.Item name="server_ids" label="发布主机" rules={[{ required: true, message: "请选择发布主机" }]}>
              <Select
                mode="multiple"
                allowClear
                showSearch
                optionFilterProp="label"
                options={serverOptions}
                placeholder="请选择发布主机"
                tagRender={(props) => {
                  const { value, closable, onClose } = props;
                  const text = serverLabelById.get(Number(value)) ?? serverOptions.find((o) => o.value === Number(value))?.label;
                  return (
                    <Tag closable={closable} onClose={onClose} style={{ marginInlineEnd: 4 }}>
                      {text || String(value)}
                    </Tag>
                  );
                }}
              />
            </Form.Item>
            <Form.Item name="artifact_retain_count" label="历史版本数量">
              <InputNumber min={1} max={100} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item name="deploy_user" label="部署用户">
              <Input />
            </Form.Item>
            <Form.Item name="deploy_group" label="部署属组" hidden>
              <Input />
            </Form.Item>
            {!isFrontend && (
              <>
                <Form.Item name="run_user" label="运行用户" extra="后端 JAR/进程的运行用户">
                  <Input />
                </Form.Item>
                <Form.Item name="start_script_type" label="启动脚本类型" extra="后端部署时在目标机生成 bin/launch.sh">
                  <Select options={startScriptTypes.map((o) => ({ label: o.label, value: o.value }))} />
                </Form.Item>
                <Form.Item noStyle shouldUpdate={(p, c) => p.start_script_type !== c.start_script_type}>
                  {({ getFieldValue }) =>
                    getFieldValue("start_script_type") === "自定义脚本" ? (
                      <Form.Item
                        name="custom_script_content"
                        label="自定义 launch.sh 内容"
                        rules={[{ required: true, message: "请填写自定义启动脚本" }]}
                        extra="整段 bash 脚本，将写入目标机 destPath/bin/launch.sh"
                      >
                        <Input.TextArea rows={6} placeholder="#!/bin/bash&#10;..." />
                      </Form.Item>
                    ) : null
                  }
                </Form.Item>
              </>
            )}
            {isFrontend && (
              <Alert
                type="info"
                showIcon
                message="前端静态资源部署：制品解压到部署路径即可，由 Nginx 等 Web 服务器提供访问，无需 launch.sh 启动脚本。"
                style={{ marginBottom: 8 }}
              />
            )}
            <Form.Item name="clean_deploy_dir" label="部署前清空目录" valuePropName="checked">
              <Switch />
            </Form.Item>
          </div>
        )}
      </Form>
    </Modal>
  );
}
