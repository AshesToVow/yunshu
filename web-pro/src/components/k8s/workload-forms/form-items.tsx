// @ts-nocheck
import { Alert, Button, Card, Col, Drawer, Form, Input, InputNumber, Row, Select, Space, Typography } from "antd";
import type { FormInstance } from "antd";
import { SectionDivider } from "../../ops/section-divider";
import {
  EnvPair,
  KVPair,
  ProbeType,
  envPairsToMap,
  kvPairsToMap,
  mapToEnvPairs,
  mapToKvPairs,
  parseExecCommandJson,
  parseIntOrStringPort,
  probeFromForm,
  probeToForm,
  safeGet,
  safeParseYaml,
  toNumberOrUndefined
} from "./helpers";

export function EnvPairsFormItem({ name }: { name: string }) {
  return (
    <Form.List name={name}>
      {(fields, { add, remove }) => (
        <Space direction="vertical" style={{ width: "100%" }}>
          {fields.map((f) => (
            <Space key={f.key} style={{ display: "flex" }} align="baseline">
              <Form.Item name={[f.name, "key"]} rules={[{ required: true, message: "Key 必填" }]} style={{ marginBottom: 0 }}>
                <Input placeholder="KEY" style={{ width: 220 }} />
              </Form.Item>
              <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}>
                <Input placeholder="Value" style={{ width: 360 }} />
              </Form.Item>
              <Button onClick={() => remove(f.name)}>删除</Button>
            </Space>
          ))}
          <Button onClick={() => add({ key: "", value: "" })}>新增环境变量</Button>
        </Space>
      )}
    </Form.List>
  );
}

export function WorkloadFormModal<T extends object>(props: {
  title: string;
  open: boolean;
  loading?: boolean;
  form: FormInstance<T>;
  onCancel: () => void;
  onSubmit: (values: T) => void | Promise<void>;
  children: React.ReactNode;
  /** 右侧抽屉宽度，默认 920 */
  drawerWidth?: number | string;
  /** 仅表单 + 底部确定，不包 Drawer（嵌入 YamlCrudPage「表单创建」Tab） */
  embedded?: boolean;
}) {
  const { title, open, form, onCancel, onSubmit, children, loading, drawerWidth = 920, embedded } = props;
  const formBody = (
    <Form form={form} layout="vertical" requiredMark="optional" scrollToFirstError>
      {children}
    </Form>
  );
  const footer = (
    <Space style={{ marginTop: 16 }}>
      <Button type="primary" loading={loading} onClick={() => void form.validateFields().then((v) => void onSubmit(v))}>
        确定
      </Button>
    </Space>
  );
  if (embedded) {
    return (
      <>
        {formBody}
        {footer}
      </>
    );
  }
  return (
    <Drawer
      title={title}
      placement="right"
      width={drawerWidth}
      open={open}
      onClose={onCancel}
      destroyOnClose
      maskClosable={false}
      styles={{ body: { paddingBottom: 24 } }}
      extra={
        <Space>
          <Button onClick={onCancel}>取消</Button>
          <Button type="primary" loading={loading} onClick={() => void form.validateFields().then((v) => void onSubmit(v))}>
            确定
          </Button>
        </Space>
      }
    >
      {formBody}
    </Drawer>
  );
}

export function NameNamespaceItems() {
  return (
    <Card size="small" title="基础信息" styles={{ body: { paddingBottom: 8 } }}>
      <Row gutter={16}>
        <Col xs={24} md={14}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input />
          </Form.Item>
        </Col>
        <Col xs={24} md={10}>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: "请选择命名空间" }]}>
            <Input />
          </Form.Item>
        </Col>
      </Row>
    </Card>
  );
}

export function ContainerCommonItems(opts?: { showPort?: boolean; showRestartPolicy?: boolean }) {
  return (
    <Card size="small" title="容器信息" styles={{ body: { paddingBottom: 8 } }}>
      <Row gutter={16}>
        <Col xs={24} md={10}>
          <Form.Item name="container_name" label="容器名" rules={[{ required: true, message: "请输入容器名" }]}>
            <Input />
          </Form.Item>
        </Col>
        <Col xs={24} md={14}>
          <Form.Item name="image" label="镜像" rules={[{ required: true, message: "请输入镜像" }]}>
            <Input placeholder="nginx:latest" />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        {opts?.showPort ? (
          <Col xs={24} md={8}>
            <Form.Item name="port" label="容器端口（可选）">
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        ) : null}
        {opts?.showPort ? (
          <Col xs={24} md={8}>
            <Form.Item name="port_name" label="端口名称（可选）" extra="例如：http、metrics（供 Probe/Service 引用）">
              <Input placeholder="http" />
            </Form.Item>
          </Col>
        ) : null}
        {opts?.showRestartPolicy ? (
          <Col xs={24} md={8}>
            <Form.Item name="restart_policy" label="RestartPolicy" rules={[{ required: true, message: "请选择" }]}>
              <Select
                options={[
                  { label: "Never", value: "Never" },
                  { label: "OnFailure", value: "OnFailure" },
                ]}
              />
            </Form.Item>
          </Col>
        ) : null}
      </Row>
      <Form.Item name="command" label="启动命令" extra="sh -c 执行；可不填">
        <Input placeholder='例如：echo hello && sleep 5' />
      </Form.Item>
      <Form.Item label="环境变量">
        <EnvPairsFormItem name="env_pairs" />
      </Form.Item>
    </Card>
  );
}

export function WorkloadAdvancedItems() {
  return (
    <Card size="small" title="资源与调度" styles={{ body: { paddingBottom: 8 } }}>
      <Row gutter={16}>
        <Col xs={24} md={8}>
          <Form.Item name="image_pull_policy" label="镜像拉取策略">
            <Select
              options={[
                { label: "IfNotPresent", value: "IfNotPresent" },
                { label: "Always", value: "Always" },
                { label: "Never", value: "Never" },
              ]}
            />
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="requests_cpu" label="CPU Request">
            <Input placeholder="100m" />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="limits_cpu" label="CPU Limit">
            <Input placeholder="500m" />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="requests_memory" label="Memory Request">
            <Input placeholder="128Mi" />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="limits_memory" label="Memory Limit">
            <Input placeholder="512Mi" />
          </Form.Item>
        </Col>
      </Row>

      <SectionDivider style={{ marginTop: 0 }}>容忍（Tolerations）</SectionDivider>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        容忍本身是合理的，用来让 Pod 可以调度到带 `taint` 的节点上。
        `tolerationSeconds` 仅在 `effect=NoExecute` 时有意义，表示 Pod 被驱逐前还能继续保留多少秒；不填通常表示一直容忍。
      </Typography.Paragraph>
      <Form.List name="tolerations">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%" }} size={12}>
            {fields.map((f) => (
              <Card key={f.key} size="small">
                <Row gutter={12}>
                  <Col xs={24} md={6}>
                    <Form.Item name={[f.name, "key"]} label="Key" style={{ marginBottom: 12 }}>
                      <Input placeholder="node-role.kubernetes.io/master" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={4}>
                    <Form.Item name={[f.name, "operator"]} label="Operator" initialValue="Equal" style={{ marginBottom: 12 }}>
                      <Select
                        options={[
                          { label: "Equal", value: "Equal" },
                          { label: "Exists", value: "Exists" },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "value"]} label="Value" style={{ marginBottom: 12 }}>
                      <Input placeholder="true" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "effect"]} label="Effect" style={{ marginBottom: 12 }}>
                      <Select
                        allowClear
                        options={[
                          { label: "NoSchedule", value: "NoSchedule" },
                          { label: "PreferNoSchedule", value: "PreferNoSchedule" },
                          { label: "NoExecute", value: "NoExecute" },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={4}>
                    <Form.Item name={[f.name, "toleration_seconds"]} label="持续秒数" tooltip="仅 NoExecute 时常用" style={{ marginBottom: 12 }}>
                      <InputNumber placeholder="3600" style={{ width: "100%" }} min={0} />
                    </Form.Item>
                  </Col>
                </Row>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Card>
            ))}
            <Button onClick={() => add({ key: "", operator: "Equal", value: "" })}>新增容忍</Button>
          </Space>
        )}
      </Form.List>

      <SectionDivider>卷（Volumes）</SectionDivider>
      <Form.List name="volumes">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%" }} size={12}>
            {fields.map((f) => (
              <Card key={f.key} size="small">
                <Row gutter={12}>
                  <Col xs={24} md={7}>
                    <Form.Item name={[f.name, "name"]} label="卷名" rules={[{ required: true, message: "卷名必填" }]} style={{ marginBottom: 12 }}>
                      <Input placeholder="config-volume" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "type"]} label="类型" initialValue="emptyDir" style={{ marginBottom: 12 }}>
                      <Select
                        options={[
                          { label: "emptyDir", value: "emptyDir" },
                          { label: "configMap", value: "configMap" },
                          { label: "secret", value: "secret" },
                          { label: "pvc", value: "pvc" },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name={[f.name, "source_name"]} label="来源名称" tooltip="configMap / secret / pvc 时填写" style={{ marginBottom: 12 }}>
                      <Input placeholder="source name (cm/secret/pvc)" />
                    </Form.Item>
                  </Col>
                </Row>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Card>
            ))}
            <Button onClick={() => add({ name: "", type: "emptyDir", source_name: "" })}>新增卷</Button>
          </Space>
        )}
      </Form.List>

      <SectionDivider>卷挂载（VolumeMounts）</SectionDivider>
      <Form.List name="volume_mounts">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%" }} size={12}>
            {fields.map((f) => (
              <Card key={f.key} size="small">
                <Row gutter={12}>
                  <Col xs={24} md={6}>
                    <Form.Item name={[f.name, "name"]} label="卷名" rules={[{ required: true, message: "卷名必填" }]} style={{ marginBottom: 12 }}>
                      <Input placeholder="volume name" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item name={[f.name, "mount_path"]} label="挂载路径" rules={[{ required: true, message: "挂载路径必填" }]} style={{ marginBottom: 12 }}>
                      <Input placeholder="/data" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "sub_path"]} label="SubPath" style={{ marginBottom: 12 }}>
                      <Input placeholder="subPath" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "read_only"]} label="权限" initialValue={false} style={{ marginBottom: 12 }}>
                      <Select
                        options={[
                          { label: "读写", value: false },
                          { label: "只读", value: true },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                </Row>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Card>
            ))}
            <Button onClick={() => add({ name: "", mount_path: "", read_only: false })}>新增挂载</Button>
          </Space>
        )}
      </Form.List>
    </Card>
  );
}

/** 零中断滚动发布推荐：RollingUpdate + maxSurge 1 + maxUnavailable 0 */
export function applyZeroDowntimeDeploymentStrategy(form: FormInstance) {
  form.setFieldsValue({
    strategy_type: "RollingUpdate",
    rolling_update_max_surge: "1",
    rolling_update_max_unavailable: "0",
    min_ready_seconds: 5,
    progress_deadline_seconds: 600,
  });
}

export function WorkloadPolicyItems(opts?: {
  showDeployStrategy?: boolean;
  showStatefulSetStrategy?: boolean;
  showDaemonSetStrategy?: boolean;
  showCronJobPolicy?: boolean;
  showJobPolicy?: boolean;
}) {
  const form = Form.useFormInstance();
  return (
    <Card size="small" title="发布与调度策略" styles={{ body: { paddingBottom: 8 } }}>
      <Form.Item label="NodeSelector">
        <EnvPairsFormItem name="node_selector_pairs" />
      </Form.Item>
      <Form.Item name="affinity_yaml" label="Affinity YAML（可选）">
        <Input.TextArea rows={4} placeholder={"nodeAffinity:\n  requiredDuringSchedulingIgnoredDuringExecution:\n    nodeSelectorTerms: []"} />
      </Form.Item>
      {opts?.showDeployStrategy ? (
        <>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message="滚动发布保持业务可访问"
            description="推荐 RollingUpdate，maxUnavailable=0 保证更新期间可用副本不减少；需配置 readinessProbe 后新 Pod 才计入就绪。Recreate 会短暂中断。"
          />
          <Space style={{ marginBottom: 12 }}>
            <Button type="primary" ghost onClick={() => form && applyZeroDowntimeDeploymentStrategy(form)}>
              应用零中断推荐
            </Button>
          </Space>
          <Row gutter={16}>
            <Col xs={24} md={7}>
              <Form.Item name="strategy_type" label="部署策略">
                <Select allowClear options={[{ label: "RollingUpdate", value: "RollingUpdate" }, { label: "Recreate", value: "Recreate" }]} />
              </Form.Item>
            </Col>
            <Col xs={24} md={5}>
              <Form.Item name="rolling_update_max_surge" label="maxSurge" tooltip="可多起的 Pod 数，如 1 或 25%">
                <Input placeholder="1 / 25%" />
              </Form.Item>
            </Col>
            <Col xs={24} md={5}>
              <Form.Item name="rolling_update_max_unavailable" label="maxUnavailable" tooltip="更新时允许不可用的 Pod 数，零中断建议 0">
                <Input placeholder="0" />
              </Form.Item>
            </Col>
            <Col xs={24} md={7}>
              <Form.Item name="revision_history_limit" label="历史版本数">
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col xs={24} md={8}>
              <Form.Item name="min_ready_seconds" label="minReadySeconds" tooltip="Pod 就绪后额外等待秒数再视为可用">
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
            <Col xs={24} md={8}>
              <Form.Item name="progress_deadline_seconds" label="progressDeadlineSeconds">
                <InputNumber min={1} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>
        </>
      ) : null}
      {opts?.showStatefulSetStrategy ? (
        <Row gutter={16}>
          <Col xs={24} md={8}>
            <Form.Item name="update_strategy_type" label="StatefulSet UpdateStrategy">
              <Select allowClear options={[{ label: "RollingUpdate", value: "RollingUpdate" }, { label: "OnDelete", value: "OnDelete" }]} />
            </Form.Item>
          </Col>
          <Col xs={24} md={8}>
            <Form.Item name="rolling_update_partition" label="rollingUpdate.partition">
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col xs={24} md={8}>
            <Form.Item name="revision_history_limit" label="revisionHistoryLimit">
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
      ) : null}
      {opts?.showDaemonSetStrategy ? (
        <Row gutter={16}>
          <Col xs={24} md={7}>
            <Form.Item name="update_strategy_type" label="更新策略">
              <Select allowClear options={[{ label: "RollingUpdate", value: "RollingUpdate" }, { label: "OnDelete", value: "OnDelete" }]} />
            </Form.Item>
          </Col>
          <Col xs={24} md={5}>
            <Form.Item name="rolling_update_max_surge" label="maxSurge">
              <Input placeholder="25% / 1" />
            </Form.Item>
          </Col>
          <Col xs={24} md={5}>
            <Form.Item name="rolling_update_max_unavailable" label="maxUnavailable">
              <Input placeholder="25% / 0" />
            </Form.Item>
          </Col>
          <Col xs={24} md={7}>
            <Form.Item name="revision_history_limit" label="历史版本数">
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
      ) : null}
      {opts?.showCronJobPolicy ? (
        <Row gutter={16}>
          <Col xs={24} md={6}><Form.Item name="concurrency_policy" label="concurrencyPolicy"><Select allowClear options={[{ label: "Allow", value: "Allow" }, { label: "Forbid", value: "Forbid" }, { label: "Replace", value: "Replace" }]} /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="successful_jobs_history_limit" label="successfulJobsHistoryLimit"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="failed_jobs_history_limit" label="failedJobsHistoryLimit"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="starting_deadline_seconds" label="startingDeadlineSeconds"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
        </Row>
      ) : null}
      {opts?.showCronJobPolicy || opts?.showJobPolicy ? (
        <Row gutter={16}>
          <Col xs={24} md={5}><Form.Item name="parallelism" label="parallelism"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={5}><Form.Item name="completions" label="completions"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={5}><Form.Item name="backoff_limit" label="backoffLimit"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={5}><Form.Item name="active_deadline_seconds" label="activeDeadlineSeconds"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={4}><Form.Item name="ttl_seconds_after_finished" label="ttlSecondsAfterFinished"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
        </Row>
      ) : null}
    </Card>
  );
}

export function DeploymentHealthAndImagePullSecretsItems() {
  return (
    <Card size="small" title="探针与镜像拉取" styles={{ body: { paddingBottom: 8 } }}>
      <Form.Item label="镜像拉取 Secret">
        <Form.List name="image_pull_secrets">
          {(fields, { add, remove }) => (
            <Space direction="vertical" style={{ width: "100%" }}>
              {fields.map((f) => (
                <Space key={f.key} style={{ display: "flex" }} align="baseline">
                  <Form.Item name={f.name} rules={[{ required: true, message: "请输入 Secret 名称" }]} style={{ marginBottom: 0 }}>
                    <Input placeholder="my-image-pull-secret" style={{ width: 260 }} />
                  </Form.Item>
                  <Button onClick={() => remove(f.name)}>删除</Button>
                </Space>
              ))}
              <Button onClick={() => add("")}>新增 Secret</Button>
            </Space>
          )}
        </Form.List>
      </Form.Item>

      <SectionDivider>Liveness Probe（存活探针）</SectionDivider>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        K8s 探测动作有 `httpGet`、`tcpSocket`、`exec` 三种（与社区/源码一致）。`exec` 需要填写 JSON 数组形式的命令。
      </Typography.Paragraph>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_probe_type" label="探针类型" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "httpGet", value: "httpGet" },
              { label: "tcpSocket", value: "tcpSocket" },
              { label: "exec", value: "exec" },
            ]}
          />
        </Form.Item>
        <Form.Item name="liveness_http_path" label="HTTP path" style={{ flex: 1 }}>
          <Input placeholder="/health" />
        </Form.Item>
      </Space>
      <Form.Item name="liveness_exec_command" label="Exec 命令" extra='例如：["sh","-c","test -f /tmp/ready"]'>
        <Input placeholder='例如：["sh","-c","curl -sf http://127.0.0.1:8080/health"]' />
      </Form.Item>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_http_port" label="HTTP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
        <Form.Item name="liveness_http_scheme" label="HTTP scheme" style={{ width: 180 }}>
          <Select
            allowClear
            options={[
              { label: "HTTP", value: "HTTP" },
              { label: "HTTPS", value: "HTTPS" },
            ]}
          />
        </Form.Item>
        <Form.Item name="liveness_tcp_port" label="TCP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
      </Space>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_initial_delay_seconds" label="initialDelaySeconds（首次探测延迟秒）">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_period_seconds" label="periodSeconds（每次探测间隔秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_timeout_seconds" label="timeoutSeconds（单次超时秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_failure_threshold" label="failureThreshold（连续失败次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_success_threshold" label="successThreshold（连续成功次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>

      <SectionDivider>Readiness Probe（就绪探针）</SectionDivider>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_probe_type" label="探针类型" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "httpGet", value: "httpGet" },
              { label: "tcpSocket", value: "tcpSocket" },
              { label: "exec", value: "exec" },
            ]}
          />
        </Form.Item>
        <Form.Item name="readiness_http_path" label="HTTP path" style={{ flex: 1 }}>
          <Input placeholder="/ready" />
        </Form.Item>
      </Space>
      <Form.Item name="readiness_exec_command" label="Exec 命令" extra='例如：["sh","-c","test -f /tmp/ready"]'>
        <Input placeholder='例如：["sh","-c","cat /tmp/ready"]' />
      </Form.Item>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_http_port" label="HTTP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
        <Form.Item name="readiness_http_scheme" label="HTTP scheme" style={{ width: 180 }}>
          <Select
            allowClear
            options={[
              { label: "HTTP", value: "HTTP" },
              { label: "HTTPS", value: "HTTPS" },
            ]}
          />
        </Form.Item>
        <Form.Item name="readiness_tcp_port" label="TCP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
      </Space>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_initial_delay_seconds" label="initialDelaySeconds（首次探测延迟秒）">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_period_seconds" label="periodSeconds（每次探测间隔秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_timeout_seconds" label="timeoutSeconds（单次超时秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_failure_threshold" label="failureThreshold（连续失败次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_success_threshold" label="successThreshold（连续成功次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>

      <SectionDivider>Startup Probe（启动探针）</SectionDivider>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        启动探针通常用于启动慢的容器，避免应用还没真正启动完成时就被 liveness 误杀。
        当前先开放基础参数输入，后续我会把它连到完整的 YAML 构建与回填。
      </Typography.Paragraph>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="startup_probe_type" label="探针类型" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "httpGet", value: "httpGet" },
              { label: "tcpSocket", value: "tcpSocket" },
              { label: "exec", value: "exec" },
            ]}
          />
        </Form.Item>
        <Form.Item name="startup_http_path" label="HTTP path" style={{ flex: 1 }}>
          <Input placeholder="/startup" />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="startup_http_port" label="HTTP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
        <Form.Item name="startup_http_scheme" label="HTTP scheme" style={{ width: 180 }}>
          <Select allowClear options={[{ label: "HTTP", value: "HTTP" }, { label: "HTTPS", value: "HTTPS" }]} />
        </Form.Item>
        <Form.Item name="startup_tcp_port" label="TCP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
      </Space>
      <Form.Item name="startup_exec_command" label="Exec 命令" extra='例如：["sh","-c","test -f /tmp/ready"]'>
        <Input placeholder='例如：["sh","-c","cat /tmp/ready"]' />
      </Form.Item>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="startup_initial_delay_seconds" label="initialDelaySeconds（首次探测延迟秒）">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="startup_period_seconds" label="periodSeconds（每次探测间隔秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="startup_timeout_seconds" label="timeoutSeconds（单次超时秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="startup_failure_threshold" label="failureThreshold（连续失败次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="startup_success_threshold" label="successThreshold（连续成功次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
    </Card>
  );
}

