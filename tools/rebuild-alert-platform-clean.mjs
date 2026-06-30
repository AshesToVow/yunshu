/**
 * Clean rebuild of alert-monitor split from git monolith.
 */
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.join(__dirname, "..");
const outProvider = path.join(root, "web/src/pages/alert-monitor/platform-provider.tsx");
const bakPath = path.join(root, "web/src/pages/.alert-monitor-platform-page.bak.tsx");

if (!fs.existsSync(bakPath)) {
  fs.writeFileSync(
    bakPath,
    execSync("git show HEAD:web/src/pages/alert-monitor-platform-page.tsx", { cwd: root, encoding: "utf8" }),
    "utf8",
  );
}

const lines = fs.readFileSync(bakPath, "utf8").split(/\r?\n/);
const exportIdx = lines.findIndex((l) => l.startsWith("export function AlertMonitorPlatformPage"));
const returnIdx = lines.findIndex(
  (l, i) => l.trim() === "return (" && String(lines[i + 1] || "").includes("page-stack"),
);
if (exportIdx < 0 || returnIdx < 0) throw new Error("cannot locate page function bounds");

const header = lines
  .slice(0, exportIdx)
  .join("\n")
  .replace(/from "\.\.\//g, 'from "../../')
  .replace(/import "\.\.\//g, 'import "../../')
  .replace(
    'import { useCallback, useEffect, useMemo, useRef, useState, lazy, Suspense } from "react";',
    'import { useCallback, useEffect, useMemo, useRef, useState } from "react";',
  )
  .replace('import { useSearchParams } from "react-router-dom";\n', "")
  .replace('import { PageTelemetryHeader } from "../../components/page-telemetry-header";\n', "")
  .replace(/const AlertInhibitionPanel = lazy[\s\S]*?\}\);\n\n/, "")
  .replace(/const AlertConfigCenterPanel = lazy[\s\S]*?\}\);\n\n/, "");

let hook = lines.slice(exportIdx, returnIdx).join("\n");
hook = hook.replace(
  "export function AlertMonitorPlatformPage() {\n  const [searchParams, setSearchParams] = useSearchParams();",
  "function useAlertMonitorPlatformState() {\n  const navigate = useNavigate();\n  const { tab: tabParam } = useParams();\n  const [searchParams, setSearchParams] = useSearchParams();",
);

const oldTab = `  const tab: TabKey = useMemo(() => {
    const t = searchParams.get("tab");
    if (t === "config") {
      return searchParams.get("cfg") === "history" ? "history" : "policies";
    }
    if (t === "policies" || t === "silences" || t === "inhibition" || t === "rules" || t === "history" || t === "cloud-expiry" || t === "promql") {
      return t;
    }
    return "datasources";
  }, [searchParams]);

  function setTab(key: TabKey) {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (key === "datasources") p.delete("tab");
        else p.set("tab", key);
        p.delete("cfg");
        return p;
      },
      { replace: true },
    );
  }`;

const newTab = `  const tab: AlertMonitorTabKey = useMemo(() => normalizeAlertMonitorTab(tabParam), [tabParam]);

  useEffect(() => {
    const legacyTab = searchParams.get("tab");
    if (!legacyTab || tabParam) return;
    let next = normalizeAlertMonitorTab(legacyTab);
    if (legacyTab === "config" && searchParams.get("cfg") === "history") {
      next = "history";
    }
    const qs = new URLSearchParams(searchParams);
    qs.delete("tab");
    qs.delete("cfg");
    const tail = qs.toString();
    const path = tabPathForKey(next);
    navigate(tail ? \`\${path}?\${tail}\` : path, { replace: true });
  }, [navigate, searchParams, tabParam]);

  function setTab(key: AlertMonitorTabKey) {
    const qs = searchParams.toString();
    const path = tabPathForKey(key);
    navigate(qs ? \`\${path}?\${qs}\` : path, { replace: true });
  }`;

if (!hook.includes(oldTab)) throw new Error("old tab block not found");
hook = hook.replace(oldTab, newTab);

const tabKeys = [
  { file: "datasources-tab", start: 2217, end: 2233 },
  { file: "policies-tab", start: 2240, end: 2290 },
  { file: "history-tab", start: 2297, end: 2314 },
  { file: "inhibition-tab", start: 2320, end: 2320 },
  { file: "silences-tab", start: 2326, end: 2420 },
  { file: "rules-tab", start: 2427, end: 2476 },
  { file: "cloud-expiry-tab", start: 2483, end: 2523 },
  { file: "promql-tab", start: 2530, end: 2607 },
];
const modalsBody = lines.slice(2612, 3407).join("\n");
const jsxChunks = [...tabKeys.map((t) => lines.slice(t.start - 1, t.end).join("\n")), modalsBody].join("\n");

const skip = new Set([
  "true", "false", "null", "undefined", "void", "return", "async", "await", "typeof", "string", "number",
  "object", "function", "import", "from", "export", "const", "let", "var", "if", "else", "new", "this",
  "default", "as", "type", "key", "value", "label", "children", "style", "width", "size", "small", "middle",
  "vertical", "primary", "secondary", "info", "warning", "strong", "code", "span", "ul", "li",
  "loading", "disabled", "placeholder", "options", "columns", "dataSource", "pagination", "scroll", "rowKey",
  "title", "description", "message", "showIcon",   "embedded", "hideTabs", "activeTab", "onTabChange",
  "initialEventCategory", "projectId", "destroyOnClose", "placement", "open", "extra",
  "form", "checked", "required", "name", "hidden", "mode", "allowClear", "onChange", "onClick", "onClose",
  "Alert", "Button", "Space", "Table", "Typography", "Select", "Input", "Segmented", "Radio", "Form", "Drawer",
  "Switch", "AutoComplete", "TreeSelect", "DatePicker", "Card", "Suspense", "Collapse", "message", "Link",
  "ctx", "c", "e", "v", "k", "n", "r", "d", "p", "o", "x", "y", "z", "i", "j", "a", "b", "f", "g", "h",
  "w", "m", "s", "t", "u", "l", "id", "idx", "row", "col", "field", "fields", "add", "remove", "option",
  "input", "record", "text", "index", "prev", "next", "props", "state", "ref", "className",
  "it", "item", "items", "val", "match", "labels", "alertname", "parsed", "patch", "payload", "range",
  "userIds", "deptIds", "emailNew", "monitorRuleID", "rows", "parts", "body", "enabled", "selected", "targets",
  "metric", "comparator", "threshold", "op", "startsAt", "endsAt", "key", "name", "comment", "raw",
]);

const usedIds = new Set();
const idRe = /\b([a-z][a-zA-Z0-9]*)\b/g;
let m;
while ((m = idRe.exec(jsxChunks)) !== null) {
  const id = m[1];
  if (!skip.has(id) && id.length > 1) usedIds.add(id);
}

const declared = new Set(["tab", "setTab", "projectContextId", "loading", "activeProjectName", "setProjectContext", "openHistoryTab"]);
const declRe = /(?:const|let|function)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)/g;
while ((m = declRe.exec(hook)) !== null) declared.add(m[1]);
const destructureRe = /const\s+\[([a-zA-Z_$][a-zA-Z0-9_$]*)(?:\s*,\s*([a-zA-Z_$][a-zA-Z0-9_$]*))?\]/g;
while ((m = destructureRe.exec(hook)) !== null) {
  declared.add(m[1]);
  if (m[2]) declared.add(m[2]);
}

const returnKeys = [...new Set([...[...usedIds].filter((id) => declared.has(id)), "tab", "setTab", "projectContextId", "loading", "activeProjectName", "setProjectContext", "openHistoryTab", "dsForm", "silForm", "ruleForm", "assignForm", "blkForm", "cloudExpiryForm"])].sort();

const returnBlock = `  return {\n${returnKeys.map((k) => `    ${k},`).join("\n")}\n  };\n}`;

const bridge = `
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import { AlertMonitorProvider } from "./context";
import { AlertMonitorLayout } from "./layout";
import { AlertMonitorModals } from "./modals";
import { normalizeAlertMonitorTab, tabPathForKey, type AlertMonitorTabKey } from "./tab-config";

export type { AlertMonitorTabKey };

export function AlertMonitorPlatformRoot() {
  const state = useAlertMonitorPlatformState();
  return (
    <AlertMonitorProvider value={state as never}>
      <AlertMonitorLayout />
      <AlertMonitorModals />
    </AlertMonitorProvider>
  );
}
`;

fs.writeFileSync(outProvider, `${header}\n${bridge}\n${hook}\n\n${returnBlock}\n`, "utf8");
console.log("provider ok", returnKeys.length);

// tabs + modals (same as before with prefix fix)
function prefixJsx(body, prefix, keys) {
  let out = body;
  for (const id of [...keys].sort((a, b) => b.length - a.length)) {
    if (skip.has(id)) continue;
    out = out.replace(new RegExp(`\\b${id}\\b`, "g"), (match, offset, str) => {
      const before = str[offset - 1];
      if (before === "." || before === '"' || before === "'") return match;
      if (/^\s*=/.test(str.slice(offset + match.length))) return match;
      return `${prefix}${id}`;
    });
  }
  return out;
}

function postProcessTabJsx(body) {
  return body
    .replace(/projectContextId=\{projectContextId\}/g, "projectContextId={ctx.projectContextId}")
    .replace(/projectId=\{projectContextId\}/g, "projectId={ctx.projectContextId}")
    .replace(/\(projectContextId\)/g, "(ctx.projectContextId)")
    .replace(/\{projectContextId \?/g, "{ctx.projectContextId ?")
    .replace(/#\$\{projectContextId\}/g, "#${ctx.projectContextId}")
    .replace(/\bpromMode\b/g, (m, offset, str) => {
      const before = str[offset - 1];
      if (before === "." || before === "'") return m;
      if (/^\s*=/.test(str.slice(offset + m.length))) return m;
      return "ctx.promMode";
    })
    .replace(/\bpromViewMode\b/g, (m, offset, str) => {
      const before = str[offset - 1];
      if (before === "." || before === "'") return m;
      if (/^\s*=/.test(str.slice(offset + m.length))) return m;
      return "ctx.promViewMode";
    });
}

function stripTabBody(raw, file) {
  let body = raw.trim();
  if (file === "inhibition-tab") body = body.replace(/^children:\s*/, "").replace(/,\s*$/, "");
  return body;
}

const tabsDir = path.join(root, "web/src/pages/alert-monitor/tabs");
fs.mkdirSync(tabsDir, { recursive: true });
const componentName = (file) => file.split("-").map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join("");

for (const t of tabKeys) {
  const rawBody = stripTabBody(lines.slice(t.start - 1, t.end).join("\n"), t.file);
  const newBody = postProcessTabJsx(prefixJsx(rawBody, "ctx.", returnKeys));
  const name = componentName(t.file);
  const needsInhibition = t.file === "inhibition-tab";
  const needsConfig = t.file === "policies-tab" || t.file === "history-tab";
  fs.writeFileSync(
    path.join(tabsDir, `${t.file}.tsx`),
    `import {
  Alert,
  Button,
  Card,
  Collapse,
  Input,
  Radio,
  Segmented,
  Select,
  Space,
  Table,
  Typography,
} from "antd";
import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { lazy, Suspense } from "react";
import { useAlertMonitor } from "../context";
${needsConfig ? `
const AlertConfigCenterPanel = lazy(async () => {
  const mod = await import("../../alert-config-center-panel");
  return { default: mod.AlertConfigCenterPanel };
});` : ""}${needsInhibition ? `
const AlertInhibitionPanel = lazy(async () => {
  const mod = await import("../../alert-inhibition-panel");
  return { default: mod.AlertInhibitionPanel };
});` : ""}

export function ${name}() {
  const ctx = useAlertMonitor();
  return (
${newBody}
  );
}
`,
    "utf8",
  );
}

let modalsJsx = prefixJsx(modalsBody, "c.", returnKeys)
  .replace(/projectContextId=\{projectContextId\}/g, "projectContextId={c.projectContextId}")
  .replace(/\(projectContextId\)/g, "(c.projectContextId)");
modalsJsx = modalsJsx
  .replace(/form=\{dsForm\}/g, "form={c.dsForm}")
  .replace(/form=\{silForm\}/g, "form={c.silForm}")
  .replace(/form=\{cloudExpiryForm\}/g, "form={c.cloudExpiryForm}")
  .replace(/form=\{ruleForm\}/g, "form={c.ruleForm}")
  .replace(/form=\{assignForm\}/g, "form={c.assignForm}")
  .replace(/form=\{blkForm\}/g, "form={c.blkForm}")
  .replace(/\{ \.\.\.it, c\.metric:/g, "{ ...it, metric:")
  .replace(/\{ c\.metric:/g, "{ metric:");

fs.writeFileSync(
  path.join(root, "web/src/pages/alert-monitor/modals.tsx"),
  `import {
  AutoComplete,
  Button,
  Card,
  DatePicker,
  Drawer,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  Switch,
  Table,
  TreeSelect,
  Typography,
  message,
} from "antd";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import type { MetricLabelFilter } from "../platform-provider-types";
import { useAlertMonitor } from "./context";

type QuickSilenceTarget = {
  key: string;
  name: string;
  labels: Record<string, string>;
  startsAt: import("dayjs").Dayjs;
  endsAt: import("dayjs").Dayjs;
};

export function AlertMonitorModals() {
  const c = useAlertMonitor();
  return (
    <>
${modalsJsx}
    </>
  );
}
`,
  "utf8",
);

console.log("done");
