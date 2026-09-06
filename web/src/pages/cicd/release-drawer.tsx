import { Alert, Form, Input, Modal, Select } from "antd";
import type { FormInstance } from "antd/es/form";
import type {
  CicdArtifactItem,
  CicdBuildRun,
  CicdDeployConfig,
  CicdServiceItem,
} from "../../services/cicd";
import type { UserItem } from "../../types/api";
import { ownerEmailPreview } from "./display";
import {
  BACKEND_RELEASE_OPS,
  CONTAINER_RELEASE_OPS,
  FRONTEND_RELEASE_OPS,
  releaseOpLabel,
} from "./release-ops";

type Props = {
  open: boolean;
  releaseService: CicdServiceItem | null;
  releaseDeployConfig: CicdDeployConfig | null;
  releaseArtifacts: CicdArtifactItem[];
  releaseArtifactsLoading: boolean;
  releaseBuildRuns: CicdBuildRun[];
  releaseBuildRunsLoading: boolean;
  expandedDeploys: Record<number, CicdDeployConfig[]>;
  form: FormInstance;
  userOptions: UserItem[];
  onCancel: () => void;
  onOk: () => void | Promise<void>;
};

export function ReleaseDrawer({
  open,
  releaseService,
  releaseDeployConfig,
  releaseArtifacts,
  releaseArtifactsLoading,
  releaseBuildRuns,
  releaseBuildRunsLoading,
  expandedDeploys,
  form,
  userOptions,
  onCancel,
  onOk,
}: Props) {
  return (
    <Modal title={`发布 — ${releaseService?.name}`} open={open} onCancel={onCancel} onOk={() => void onOk()}>
      <Form form={form} layout="vertical">
        {releaseDeployConfig?.audit_enabled ? (
          <Alert
            type="warning"
            showIcon
            message="该环境已开启发布审核"
            description="提交后进入「工单中心 → 我的待办」审批；全部通过后由提交人在「CD 历史工单」详情中点击「执行发布」触发 Jenkins。"
            style={{ marginBottom: 16 }}
          />
        ) : null}
        <Form.Item name="deploy_kind" hidden>
          <Input />
        </Form.Item>
        <Form.Item name="title" label="任务名称" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="deploy_config_id" label="发布配置" rules={[{ required: true }]}>
          <Select
            options={(expandedDeploys[releaseService?.id ?? 0] ?? []).map((c) => ({
              label: `${c.name} (${c.tenv})`,
              value: c.id,
            }))}
          />
        </Form.Item>
        <Form.Item noStyle shouldUpdate={(prev, cur) => prev.deploy_kind !== cur.deploy_kind}>
          {({ getFieldValue, setFieldsValue }) => {
            const kind = getFieldValue("deploy_kind");
            const svcType = releaseService?.service_type;
            if (kind === "container") {
              const op = getFieldValue("release_operation");
              const isRollback = op === "container_rollback";
              return (
                <>
                  <Form.Item
                    name="release_operation"
                    label="操作类型"
                    rules={[{ required: true, message: "请选择操作类型" }]}
                  >
                    <Select
                      options={CONTAINER_RELEASE_OPS.map((o) => ({ label: o.label, value: o.value }))}
                      onChange={(v) => {
                        const name = releaseService?.name ?? "应用";
                        setFieldsValue({
                          title: `${name}-${releaseOpLabel(String(v))}`,
                          publish_mode: v === "container_rollback" ? "回滚" : "制品发布",
                        });
                      }}
                    />
                  </Form.Item>
                  {!isRollback ? (
                    <>
                      <Form.Item name="publish_mode" hidden initialValue="制品发布">
                        <Input />
                      </Form.Item>
                      <Form.Item
                        name="build_run_id"
                        label="CI 构建镜像"
                        rules={[{ required: true, message: "请选择已成功构建的镜像" }]}
                        extra="须先执行 CI 打包并成功推送 Harbor"
                      >
                        <Select
                          showSearch
                          loading={releaseBuildRunsLoading}
                          placeholder={releaseBuildRunsLoading ? "加载构建记录…" : releaseBuildRuns.length ? "选择镜像" : "暂无可用镜像，请先 CI 打包"}
                          options={releaseBuildRuns.map((r) => ({
                            value: r.id,
                            label: `#${r.build_number} ${r.image_address}${r.branch_name ? ` · ${r.branch_name}` : ""}`,
                          }))}
                          onChange={(id) => {
                            const run = releaseBuildRuns.find((r) => r.id === id);
                            setFieldsValue({ image_address: run?.image_address });
                          }}
                        />
                      </Form.Item>
                      <Form.Item name="image_address" hidden>
                        <Input />
                      </Form.Item>
                    </>
                  ) : (
                    <Alert type="warning" showIcon message="将回滚到 K8s 上一版本（Helm/kubectl）" />
                  )}
                </>
              );
            }
            const opOptions = svcType === "frontend" ? FRONTEND_RELEASE_OPS : BACKEND_RELEASE_OPS;
            const op = getFieldValue("release_operation");
            const opExtra =
              op === "frontend_rollback"
                ? "选择 MinIO 中的历史制品包进行回滚"
                : op === "backend_update"
                  ? "部署所选制品；选最新包为上线，选历史包即为回滚"
                  : op === "backend_initial"
                    ? "首次部署将清空目标目录后解压制品"
                    : "部署所选 MinIO 制品到目标服务器";
            return (
              <>
                <Form.Item
                  name="release_operation"
                  label="操作类型"
                  rules={[{ required: true, message: "请选择操作类型" }]}
                >
                  <Select
                    options={opOptions.map((o) => ({ label: o.label, value: o.value }))}
                    onChange={(v) => {
                      const name = releaseService?.name ?? "应用";
                      setFieldsValue({ title: `${name}-${releaseOpLabel(String(v))}` });
                    }}
                  />
                </Form.Item>
                <Form.Item
                  name="artifact_name"
                  label="MinIO 制品包"
                  rules={[{ required: true, message: "请选择要部署的制品" }]}
                  extra={opExtra}
                >
                  <Select
                    showSearch
                    loading={releaseArtifactsLoading}
                    placeholder={releaseArtifactsLoading ? "加载制品列表…" : releaseArtifacts.length ? "选择制品" : "暂无制品，请先 CI 打包"}
                    options={releaseArtifacts.map((a) => ({
                      value: a.name,
                      label: `${a.name}${a.last_modified ? ` · ${a.last_modified}` : ""}`,
                    }))}
                    filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
                  />
                </Form.Item>
              </>
            );
          }}
        </Form.Item>
        <Alert
          type="info"
          showIcon
          message="构建通知邮件"
          description={
            ownerEmailPreview(releaseService?.owner, userOptions)
              ? `Jenkins 构建/部署结果将发送至 Owner 邮箱：${ownerEmailPreview(releaseService?.owner, userOptions)}（可在用户管理维护 email；须配置 Jenkins 邮件扩展与 SMTP）`
              : "未找到 Owner 邮箱：请在用户管理为应用 Owner 填写 email，并在数据字典配置 mail_* 与 Jenkins 邮件插件"
          }
        />
      </Form>
    </Modal>
  );
}
