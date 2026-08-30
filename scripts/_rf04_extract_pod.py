# -*- coding: utf-8 -*-
"""RF-04: exact JSX relocation of pod drawers into pod/*.tsx components."""
from __future__ import annotations

from pathlib import Path

POD = Path("web/src/pages/pod-page.tsx")
OUT = Path("web/src/pages/pod")
text = POD.read_text(encoding="utf-8")
lines = text.splitlines(keepends=True)


def get(a: int, b: int) -> list[str]:
    return lines[a - 1 : b]


def dedent(ls: list[str], n: int = 2) -> str:
    out = []
    for line in ls:
        if line.startswith(" " * n):
            out.append(line[n:])
        else:
            out.append(line)
    return "".join(out)


def indent(s: str, n: int = 4) -> str:
    pad = " " * n
    parts = []
    for line in s.splitlines(keepends=True):
        parts.append("\n" if line.strip() == "" else pad + line)
    return "".join(parts)


# --- ranges ---
LOGS = get(734, 786)  # includes comment
DIAG = get(787, 910)
FILES = get(911, 1049)
CREATE = get(1263, 1951)

# Strip leading comment from logs for component (keep comment in file header)
logs_body = dedent(LOGS)
if logs_body.lstrip().startswith("{/*"):
    # drop first comment line
    lb = logs_body.splitlines(keepends=True)
    # find first <Modal
    i = next(i for i, l in enumerate(lb) if "<Modal" in l)
    logs_body = "".join(lb[i:])

diag_body = dedent(DIAG)
files_body = dedent(FILES)
create_body = dedent(CREATE)

# Write pod-logs-modal.tsx
(OUT / "pod-logs-modal.tsx").write_text(
    '''/**
 * Pod 日志高级筛选对话框（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX；state/handler 仍由页面持有，经同名 props 传入。
 */
import { DownloadOutlined, FileSearchOutlined } from "@ant-design/icons";
import { Button, Checkbox, Input, Modal, Select, Space, Switch, Typography } from "antd";
import type { PodItem } from "../../services/pods";

export type PodLogsModalProps = {
  logsOpen: boolean;
  logsTitle: string;
  logsLoading: boolean;
  logsText: string;
  logsKeyword: string;
  logsStartTime: string;
  logsEndTime: string;
  logsPrevious: boolean;
  logsTimestamps: boolean;
  logsSinceSeconds: number | undefined;
  logsSinceTime: string;
  logsContainer: string | undefined;
  logContainerOptions: string[];
  streaming: boolean;
  selected: PodItem | null;
  setLogsOpen: (v: boolean) => void;
  setLogsKeyword: (v: string) => void;
  setLogsStartTime: (v: string) => void;
  setLogsEndTime: (v: string) => void;
  setLogsPrevious: (v: boolean) => void;
  setLogsTimestamps: (v: boolean) => void;
  setLogsSinceSeconds: (v: number | undefined) => void;
  setLogsSinceTime: (v: string) => void;
  setLogsContainer: (v: string | undefined) => void;
  stopLogStream: () => void;
  startLogStream: () => void | Promise<void>;
  handleFilterLogs: () => void | Promise<void>;
  handleDownloadLogs: () => void | Promise<void>;
  handleViewLogs: (pod: PodItem, mode?: string) => void | Promise<void>;
};

export function PodLogsModal({
  logsOpen,
  logsTitle,
  logsLoading,
  logsText,
  logsKeyword,
  logsStartTime,
  logsEndTime,
  logsPrevious,
  logsTimestamps,
  logsSinceSeconds,
  logsSinceTime,
  logsContainer,
  logContainerOptions,
  streaming,
  selected,
  setLogsOpen,
  setLogsKeyword,
  setLogsStartTime,
  setLogsEndTime,
  setLogsPrevious,
  setLogsTimestamps,
  setLogsSinceSeconds,
  setLogsSinceTime,
  setLogsContainer,
  stopLogStream,
  startLogStream,
  handleFilterLogs,
  handleDownloadLogs,
  handleViewLogs,
}: PodLogsModalProps) {
  return (
'''
    + indent(logs_body.rstrip() + "\n")
    + "  );\n}\n",
    encoding="utf-8",
)

# diagnose
(OUT / "pod-diagnose-drawer.tsx").write_text(
    '''/**
 * Pod 排障诊断抽屉（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX。
 */
import { Alert, Button, Drawer, Space, Table, Tag, Typography } from "antd";
import type { AIPodDiagnoseResult } from "../../services/ai";
import type { PodDiagnoseResult, PodEventItem, PodItem } from "../../services/pods";
import { formatDateTime } from "../../utils/format";

export type PodDiagnoseDrawerProps = {
  diagnoseOpen: boolean;
  setDiagnoseOpen: (v: boolean) => void;
  selected: PodItem | null;
  diagnoseLoading: boolean;
  diagnoseResult: PodDiagnoseResult | null;
  aiDiagnoseLoading: boolean;
  aiDiagnoseResult: AIPodDiagnoseResult | null;
  events: PodEventItem[];
  handleAIDiagnose: () => void | Promise<void>;
};

export function PodDiagnoseDrawer({
  diagnoseOpen,
  setDiagnoseOpen,
  selected,
  diagnoseLoading,
  diagnoseResult,
  aiDiagnoseLoading,
  aiDiagnoseResult,
  events,
  handleAIDiagnose,
}: PodDiagnoseDrawerProps) {
  return (
'''
    + indent(diag_body.rstrip() + "\n")
    + "  );\n}\n",
    encoding="utf-8",
)

# files
(OUT / "pod-files-drawer.tsx").write_text(
    '''/**
 * Pod 文件管理抽屉（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX。
 */
import { DeleteOutlined, DownloadOutlined, ReloadOutlined, UploadOutlined } from "@ant-design/icons";
import { Button, Divider, Drawer, Input, Popconfirm, Space, Table, Typography, message } from "antd";
import type { RefObject } from "react";
import { deletePodFile, downloadPodFile, readPodFile, uploadPodFile, type PodFileItem, type PodItem } from "../../services/pods";

export type PodFilesDrawerProps = {
  fileOpen: boolean;
  setFileOpen: (v: boolean) => void;
  selected: PodItem | null;
  clusterId: number | undefined;
  filePath: string;
  setFilePath: (v: string) => void;
  fileList: PodFileItem[];
  fileLoading: boolean;
  fileContent: string;
  setFileContent: (v: string) => void;
  fileInputRef: RefObject<HTMLInputElement | null>;
  loadFiles: (pod: PodItem, path: string) => void | Promise<void>;
};

export function PodFilesDrawer({
  fileOpen,
  setFileOpen,
  selected,
  clusterId,
  filePath,
  setFilePath,
  fileList,
  fileLoading,
  fileContent,
  setFileContent,
  fileInputRef,
  loadFiles,
}: PodFilesDrawerProps) {
  return (
'''
    + indent(files_body.rstrip() + "\n")
    + "  );\n}\n",
    encoding="utf-8",
)

# form / create
(OUT / "pod-form-drawer.tsx").write_text(
    '''/**
 * Pod 创建/编辑表单抽屉（RF-04 拆分产物）
 * 从 pod-page.tsx 原地搬迁 JSX；提交逻辑仍由页面持有。
 */
import { Button, Drawer, Form, Input, InputNumber, Select, Space, Tabs, Typography } from "antd";
import type { FormInstance } from "antd/es/form";
import { MonacoYamlEditor } from "../../components/k8s/monaco-yaml-editor";
import type { PodItem } from "../../services/pods";
import { POD_CREATE_YAML_TEMPLATE } from "./pod-create-template";
import type { PodSimpleFormValues } from "./pod-form-payload";

export type PodFormDrawerProps = {
  createOpen: boolean;
  setCreateOpen: (v: boolean) => void;
  namespace: string;
  simpleMode: "create" | "edit";
  setSimpleMode: (v: "create" | "edit") => void;
  editTarget: PodItem | null;
  setEditTarget: (v: PodItem | null) => void;
  creating: boolean;
  simpleForm: FormInstance<PodSimpleFormValues>;
  yamlForm: FormInstance<{ manifest: string }>;
  rfc1123Subdomain: RegExp;
  rfc1123Label: RegExp;
  submitCreateSimple: () => void | Promise<void>;
  submitCreateYAML: () => void | Promise<void>;
};

export function PodFormDrawer({
  createOpen,
  setCreateOpen,
  namespace,
  simpleMode,
  setSimpleMode,
  editTarget,
  setEditTarget,
  creating,
  simpleForm,
  yamlForm,
  rfc1123Subdomain,
  rfc1123Label,
  submitCreateSimple,
  submitCreateYAML,
}: PodFormDrawerProps) {
  return (
'''
    + indent(create_body.rstrip() + "\n")
    + "  );\n}\n",
    encoding="utf-8",
)

# Patch pod-page: replace ranges bottom-up with component usage
# Build replacement snippets

logs_call = '''      <PodLogsModal
        logsOpen={logsOpen}
        logsTitle={logsTitle}
        logsLoading={logsLoading}
        logsText={logsText}
        logsKeyword={logsKeyword}
        logsStartTime={logsStartTime}
        logsEndTime={logsEndTime}
        logsPrevious={logsPrevious}
        logsTimestamps={logsTimestamps}
        logsSinceSeconds={logsSinceSeconds}
        logsSinceTime={logsSinceTime}
        logsContainer={logsContainer}
        logContainerOptions={logContainerOptions}
        streaming={streaming}
        selected={selected}
        setLogsOpen={setLogsOpen}
        setLogsKeyword={setLogsKeyword}
        setLogsStartTime={setLogsStartTime}
        setLogsEndTime={setLogsEndTime}
        setLogsPrevious={setLogsPrevious}
        setLogsTimestamps={setLogsTimestamps}
        setLogsSinceSeconds={setLogsSinceSeconds}
        setLogsSinceTime={setLogsSinceTime}
        setLogsContainer={setLogsContainer}
        stopLogStream={stopLogStream}
        startLogStream={startLogStream}
        handleFilterLogs={handleFilterLogs}
        handleDownloadLogs={handleDownloadLogs}
        handleViewLogs={handleViewLogs}
      />
'''

diag_call = '''      <PodDiagnoseDrawer
        diagnoseOpen={diagnoseOpen}
        setDiagnoseOpen={setDiagnoseOpen}
        selected={selected}
        diagnoseLoading={diagnoseLoading}
        diagnoseResult={diagnoseResult}
        aiDiagnoseLoading={aiDiagnoseLoading}
        aiDiagnoseResult={aiDiagnoseResult}
        events={events}
        handleAIDiagnose={handleAIDiagnose}
      />
'''

files_call = '''      <PodFilesDrawer
        fileOpen={fileOpen}
        setFileOpen={setFileOpen}
        selected={selected}
        clusterId={clusterId}
        filePath={filePath}
        setFilePath={setFilePath}
        fileList={fileList}
        fileLoading={fileLoading}
        fileContent={fileContent}
        setFileContent={setFileContent}
        fileInputRef={fileInputRef}
        loadFiles={loadFiles}
      />
'''

create_call = '''      <PodFormDrawer
        createOpen={createOpen}
        setCreateOpen={setCreateOpen}
        namespace={namespace}
        simpleMode={simpleMode}
        setSimpleMode={setSimpleMode}
        editTarget={editTarget}
        setEditTarget={setEditTarget}
        creating={creating}
        simpleForm={simpleForm}
        yamlForm={yamlForm}
        rfc1123Subdomain={rfc1123Subdomain}
        rfc1123Label={rfc1123Label}
        submitCreateSimple={submitCreateSimple}
        submitCreateYAML={submitCreateYAML}
      />
'''

# Replace bottom-up
new_lines = lines[:]
# CREATE 1263-1951
new_lines = new_lines[:1262] + [create_call if create_call.endswith("\n") else create_call + "\n"] + new_lines[1951:]
# FILES was 911-1049 — but line numbers unchanged for content before 1263
new_lines = new_lines[:910] + [files_call + "\n"] + new_lines[1049:]
# DIAG 787-910
new_lines = new_lines[:786] + [diag_call + "\n"] + new_lines[910:]
# LOGS 734-786
new_lines = new_lines[:733] + [logs_call + "\n"] + new_lines[786:]

content = "".join(new_lines)

# Add imports after usePodExec import
needle = 'import { usePodExec } from "./pod/use-pod-exec";\n'
insert = (
    needle
    + 'import { PodLogsModal } from "./pod/pod-logs-modal";\n'
    + 'import { PodDiagnoseDrawer } from "./pod/pod-diagnose-drawer";\n'
    + 'import { PodFilesDrawer } from "./pod/pod-files-drawer";\n'
    + 'import { PodFormDrawer } from "./pod/pod-form-drawer";\n'
)
if needle not in content:
    raise SystemExit("import needle missing")
content = content.replace(needle, insert, 1)

# POD_CREATE_YAML_TEMPLATE may become unused in page — leave for tsc/eslint
POD.write_text(content, encoding="utf-8")
print(f"pod-page now {len(content.splitlines())} lines")
for f in ["pod-logs-modal.tsx", "pod-diagnose-drawer.tsx", "pod-files-drawer.tsx", "pod-form-drawer.tsx"]:
    p = OUT / f
    print(f"  {f}: {len(p.read_text(encoding='utf-8').splitlines())} lines")
