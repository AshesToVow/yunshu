// @ts-nocheck
import { Alert, Button, Drawer, Form, Input, Select, Switch } from "antd";
import type { FormInstance } from "antd/es/form";
import type { CicdPipelineTemplate, CicdServiceItem } from "../../services/cicd";

type DictOption = { label: string; value: string | number };

type Props = {
  open: boolean;
  ciService: CicdServiceItem | null;
  form: FormInstance;
  pipelineTemplates: CicdPipelineTemplate[];
  frontBuildTypes: DictOption[];
  backBuildTypes: DictOption[];
  npmInstallModes: DictOption[];
  onClose: () => void;
  onSave: () => void | Promise<void>;
};

export function CiConfigDrawer({
  open,
  ciService,
  form,
  pipelineTemplates,
  frontBuildTypes,
  backBuildTypes,
  npmInstallModes,
  onClose,
  onSave,
}: Props) {
  const selectedLanguageType = Form.useWatch("language_type", form) as string | undefined;
  const selectedTemplate = pipelineTemplates.find((t) => t.language_type === selectedLanguageType);
  const isFrontend = ciService?.service_type === "frontend";

  return (
    <Drawer
      title={`编辑 CI 配置 — ${ciService?.name ?? ""}`}
      width={560}
      open={open}
      onClose={onClose}
      extra={
        <Button type="primary" onClick={() => void onSave()}>
          确认
        </Button>
      }
    >
      <Alert
        type="warning"
        showIcon
        message="提示：填写业务仓库与构建参数后保存；Yunshu 将自动在 Jenkins 创建 Pipeline Job（Pipeline script from SCM），并引用数据字典中的 jenkinsfile-new。"
        style={{ marginBottom: 16 }}
      />
      <Form form={form} layout="vertical">
        <Form.Item name="git_url" label="仓库地址" rules={[{ required: true }]}>
          <Input placeholder="git@gitee.com:org/repo.git" />
        </Form.Item>
        <Form.Item name="ref_type" label="分支或 TAG">
          <Select options={[{ label: "branch", value: "branch" }, { label: "tag", value: "tag" }]} />
        </Form.Item>
        <Form.Item
          name="ref_name"
          label="分支或 TAG 名称"
          rules={[{ required: true }]}
          extra="须与远端仓库实际分支一致（Gitee 常见默认分支为 master，勿填 main 除非仓库确有 main）"
        >
          <Input placeholder="master" />
        </Form.Item>
        <Form.Item
          name="language_type"
          label="流水线语言模板"
          rules={[{ required: true }]}
          extra={
            selectedTemplate?.script_path
              ? `将使用 Script Path：${selectedTemplate.script_path}`
              : selectedLanguageType === "custom"
                ? "自定义：按服务类型选择 front/backend/k8s Jenkinsfile"
                : selectedTemplate?.description
          }
        >
          <Select
            options={(pipelineTemplates.length
              ? pipelineTemplates
              : [
                  { language_type: "go", name: "Go" },
                  { language_type: "java", name: "Java" },
                  { language_type: "frontend", name: "前端" },
                  { language_type: "python", name: "Python" },
                  { language_type: "custom", name: "自定义" },
                ]
            ).map((t) => ({
              label: t.name,
              value: t.language_type,
            }))}
          />
        </Form.Item>
        <Form.Item name="build_type" label="打包模板类型" rules={[{ required: true }]}>
          <Select
            options={(isFrontend ? frontBuildTypes : backBuildTypes).map((o) => ({
              label: o.label,
              value: o.value,
            }))}
          />
        </Form.Item>
        <Form.Item
          name="build_shell"
          label="打包参数"
          extra={isFrontend ? "如 run build:prod（不要写 npm/yarn 前缀）" : "如 clean package -DskipTests"}
        >
          <Input />
        </Form.Item>
        <Form.Item
          name="build_path"
          label={isFrontend ? "静态资源目录路径" : "服务包路径"}
          extra={isFrontend ? "前端构建输出目录，如 dist / build" : "JAR 搜索目录，默认 target"}
        >
          <Input />
        </Form.Item>
        {!isFrontend && (
          <>
            <Form.Item name="project_name" label="项目名(projectName)">
              <Input />
            </Form.Item>
            <Form.Item name="java_tool_name" label="JDK 工具名">
              <Input placeholder="jdk8" />
            </Form.Item>
            <Form.Item name="server_port" label="服务端口">
              <Input />
            </Form.Item>
          </>
        )}
        {isFrontend && (
          <>
            <Form.Item
              name="node_version"
              label="Node.js 工具"
              extra="与 Jenkins → Global Tool Configuration 中 Node 安装名称一致（如 node24、node20、node18）"
            >
              <Select
                options={[
                  { label: "Node 24 (node24)", value: "node24" },
                  { label: "Node 20 LTS (node20)", value: "node20" },
                  { label: "Node 18 LTS (node18)", value: "node18" },
                ]}
              />
            </Form.Item>
            <Form.Item name="npm_install_mode" label="依赖安装">
              <Select options={npmInstallModes.map((o) => ({ label: o.label, value: o.value }))} />
            </Form.Item>
            <Form.Item name="clean_npm_cache" label="清理缓存" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="clean_node_modules" label="删除 node_modules" valuePropName="checked">
              <Switch />
            </Form.Item>
          </>
        )}
        <Form.Item name="version" label="版本号">
          <Input />
        </Form.Item>
        <Form.Item name="description" label="描述信息" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
      </Form>
    </Drawer>
  );
}
