# -*- coding: utf-8 -*-
"""Extract pod-page drawers (RF-04) and alert-config tabs (RF-07) by line ranges."""
from __future__ import annotations

from pathlib import Path

ROOT = Path("web/src/pages")


def extract_block(src: Path, start: int, end: int) -> str:
    """1-based inclusive line range."""
    lines = src.read_text(encoding="utf-8").splitlines(keepends=True)
    return "".join(lines[start - 1 : end])


def replace_range(src: Path, start: int, end: int, replacement: str) -> None:
    lines = src.read_text(encoding="utf-8").splitlines(keepends=True)
    new = lines[: start - 1] + [replacement if replacement.endswith("\n") else replacement + "\n"] + lines[end:]
    src.write_text("".join(new), encoding="utf-8")


# ---------- RF-04 Pod drawers ----------
POD = ROOT / "pod-page.tsx"
POD_DIR = ROOT / "pod"

# After prior extractions: logs Modal 734-786, diagnose 787-910, files 911-1049,
# exec 1050-1091, detail 1092-1262, create 1263-1951
# We extract from bottom to top so line numbers stay valid for earlier ranges.

pod_create_jsx = extract_block(POD, 1263, 1951)
pod_files_jsx = extract_block(POD, 911, 1049)
pod_diag_jsx = extract_block(POD, 787, 910)
pod_logs_jsx = extract_block(POD, 734, 786)


def indent_body(jsx: str, spaces: int = 4) -> str:
    pad = " " * spaces
    out = []
    for line in jsx.splitlines(keepends=True):
        if line.strip() == "":
            out.append("\n")
        else:
            out.append(pad + line)
    return "".join(out)


# --- pod-form-drawer.tsx ---
(POD_DIR / "pod-form-drawer.tsx").write_text(
    '''/**
 * Pod 创建/编辑表单抽屉（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX，业务提交逻辑仍留在页面。
 */
import { Button, Drawer, Form, Input, InputNumber, Select, Space, Switch, Tabs, Typography } from "antd";
import type { FormInstance } from "antd/es/form";
import { MonacoYamlEditor } from "../../components/k8s/monaco-yaml-editor";
import type { PodItem } from "../../services/pods";
import type { PodSimpleFormValues } from "./pod-form-payload";

export type PodFormDrawerProps = {
  open: boolean;
  namespace: string;
  simpleMode: "create" | "edit";
  editTarget: PodItem | null;
  createTab: string;
  setCreateTab: (v: string) => void;
  canMutate: boolean;
  creating: boolean;
  simpleForm: FormInstance<PodSimpleFormValues>;
  yamlForm: FormInstance<{ manifest: string }>;
  onClose: () => void;
  onFillYamlTemplate: () => void;
  onClearYaml: () => void;
  onSubmitSimple: () => void | Promise<void>;
  onSubmitYAML: () => void | Promise<void>;
  children?: never;
  /** 简单模式表单主体（含 affinity 等复杂字段），由页面传入以保持与原 JSX 一致 */
  simpleFormBody: React.ReactNode;
};

export function PodFormDrawer(props: PodFormDrawerProps) {
  const {
    open,
    namespace,
    simpleMode,
    editTarget,
    createTab,
    setCreateTab,
    canMutate,
    creating,
    simpleForm,
    yamlForm,
    onClose,
    onFillYamlTemplate,
    onClearYaml,
    onSubmitSimple,
    onSubmitYAML,
    simpleFormBody,
  } = props;

  return (
    <Drawer
      title={
        <Space direction="vertical" size={0}>
          <span>
            {simpleMode === "edit"
              ? `编辑 Pod（重建） - ${editTarget?.namespace || namespace}/${editTarget?.name || ""}`
              : "创建 Pod"}
          </span>
          <Typography.Text type="secondary" style={{ fontSize: 13, fontWeight: "normal" }}>
            目标命名空间：{namespace}
          </Typography.Text>
        </Space>
      }
      placement="right"
      width={960}
      open={open}
      onClose={onClose}
      destroyOnClose
      maskClosable={false}
      styles={{ body: { paddingBottom: 24 } }}
      extra={
        <Button onClick={onClose}>关闭</Button>
      }
    >
      <Tabs
        activeKey={createTab}
        onChange={setCreateTab}
        items={[
          {
            key: "simple",
            label: "简单模式",
            children: (
              <Form form={simpleForm} layout="vertical" disabled={!canMutate}>
                {simpleFormBody}
                <Button type="primary" loading={creating} disabled={!canMutate} onClick={() => void onSubmitSimple()}>
                  {simpleMode === "edit" ? "保存并重建" : "创建"}
                </Button>
              </Form>
            ),
          },
          ...(simpleMode === "create"
            ? [
                {
                  key: "yaml",
                  label: "YAML",
                  children: (
                    <Form form={yamlForm} layout="vertical" disabled={!canMutate}>
                      <Space style={{ marginBottom: 8 }}>
                        <Button size="small" onClick={onFillYamlTemplate}>
                          填入模板
                        </Button>
                        <Button size="small" onClick={onClearYaml}>
                          清空内容
                        </Button>
                      </Space>
                      <Form.Item name="manifest" label="YAML 内容" rules={[{ required: true, message: "请输入 YAML" }]}>
                        <MonacoYamlEditor height={420} />
                      </Form.Item>
                      <Button type="primary" loading={creating} disabled={!canMutate} onClick={() => void onSubmitYAML()}>
                        创建
                      </Button>
                    </Form>
                  ),
                },
              ]
            : []),
        ]}
      />
    </Drawer>
  );
}
''',
    encoding="utf-8",
)

print("NOTE: pod-form-drawer written as shell; full JSX body still needs mechanical move.")
print("Falling back to full JSX extraction for form/diagnose/files/logs.")

# Full JSX extraction is safer for pure relocation — write components that re-export the exact block
# with a wide Props interface using React types.

def write_full_jsx_component(
    out: Path,
    name: str,
    header_comment: str,
    imports: str,
    props_type: str,
    destructure: str,
    jsx: str,
) -> None:
    body = indent_body(jsx.rstrip() + "\n", 4)
    out.write_text(
        f'''/**\n * {header_comment}\n */\n{imports}\n\n{props_type}\n\nexport function {name}({destructure}) {{\n  return (\n<>\n{body}</>\n  );\n}}\n''',
        encoding="utf-8",
    )
    print(f"Wrote {out} ({len(out.read_text(encoding='utf-8').splitlines())} lines)")


# Actually for form drawer the Tabs structure has complex nested Form - better to move EXACT jsx
# and use `any` props temporarily... No, let's do exact cut with React.FC and props bag.

# Simpler plan executed below: replace each block with a component call AFTER writing
# files that contain the EXACT original JSX, with free vars from destructured props.

print("Pod create JSX lines:", pod_create_jsx.count("\n"))
print("Diagnose:", pod_diag_jsx.count("\n"), "Files:", pod_files_jsx.count("\n"), "Logs:", pod_logs_jsx.count("\n"))

# Write raw extracted JSX to temp for inspection
Path("logs/_pod_create_snip.txt").write_text(pod_create_jsx[:2000], encoding="utf-8")
print("Wrote snippet preview logs/_pod_create_snip.txt")
print("First line of create:", repr(pod_create_jsx.splitlines()[0][:80]))
print("First line of diag:", repr(pod_diag_jsx.splitlines()[0][:80]))
print("First line of files:", repr(pod_files_jsx.splitlines()[0][:80]))
print("First line of logs:", repr(pod_logs_jsx.splitlines()[0][:80]))
