/**
 * Rebuild platform-provider.tsx from alert-monitor-platform-page.tsx
 */
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.join(__dirname, "..");
const tabsOnly = process.argv.includes("--tabs-only");
const sourcePath = path.join(root, "web/src/pages/.alert-monitor-platform-page.bak.tsx");
if (!fs.existsSync(sourcePath) || fs.readFileSync(sourcePath, "utf8").includes("\0")) {
  const content = execSync("git show HEAD:web/src/pages/alert-monitor-platform-page.tsx", {
    cwd: root,
    encoding: "utf8",
  });
  fs.writeFileSync(sourcePath, content, "utf8");
}
const outPath = path.join(root, "web/src/pages/alert-monitor/platform-provider.tsx");
const lines = fs.readFileSync(sourcePath, "utf8").split(/\r?\n/);

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

const jsxChunks = [
  ...tabKeys.map((t) => lines.slice(t.start - 1, t.end).join("\n")),
  modalsBody,
].join("\n");

const idRe = /\b([a-z][a-zA-Z0-9]*)\b/g;
const skip = new Set([
  "true", "false", "null", "undefined", "void", "return", "async", "await", "typeof", "string", "number",
  "object", "function", "import", "from", "export", "const", "let", "var", "if", "else", "new", "this",
  "default", "as", "type", "key", "value", "label", "children", "style", "width", "size", "small", "middle",
  "vertical", "primary", "secondary", "info", "warning", "strong", "code", "span", "ul", "li", "margin",
  "padding", "display", "flex", "wrap", "align", "baseline", "bottom", "top", "left", "right", "center",
  "loading", "disabled", "placeholder", "options", "columns", "dataSource", "pagination", "scroll", "rowKey",
  "title", "description", "message", "showIcon", "level", "marginBottom", "fontSize", "minWidth",
  "maxWidth", "readOnly", "rows", "bordered", "pageSize", "showSizeChanger", "locale", "emptyText", "mode",
  "allowClear", "onChange", "onClick", "onSearch", "onClose", "extra", "rules", "required", "name", "hidden",
  "checked", "filterOption", "optionFilterProp", "treeData", "treeCheckable", "showSearch", "treeDefaultExpandAll",
  "destroyOnClose", "placement", "open", "format", "showTime", "RangePicker", "TextArea", "Password",
  "embedded", "hideTabs", "activeTab", "onTabChange", "projectContextId", "initialEventCategory", "projectId",
  "Group", "Item", "List", "Paragraph", "Text", "Title", "result", "resultType", "data", "status", "error",
  "success", "replace", "includes", "map", "filter", "length", "toString", "trim", "String", "Number",
  "isFinite", "Array", "Object", "JSON", "Date", "Promise", "all", "then", "catch", "finally", "push",
  "some", "every", "find", "slice", "join", "split", "keys", "values", "entries", "set", "get", "delete",
  "has", "forEach", "reduce", "flat", "flatMap", "sort", "reverse", "toUpperCase", "toLowerCase", "match",
  "test", "exec", "indexOf", "lastIndexOf", "substring", "charAt", "concat", "startsWith", "endsWith",
  "ctx", "c", "e", "v", "k", "n", "r", "d", "p", "o", "x", "y", "z", "i", "j", "a", "b", "f", "g", "h",
  "w", "m", "s", "t", "u", "l", "id", "idx", "row", "col", "field", "fields", "add", "remove", "option",
  "input", "record", "text", "index", "prev", "next", "props", "state", "ref", "className",
  "it", "item", "items", "val", "key", "match", "labels", "name", "alertname", "parsed", "patch",
  "payload", "range", "userIds", "deptIds", "emailNew", "monitorRuleID", "rows", "parts", "body",
  "enabled", "selected", "targets", "merged", "allowed", "raw", "s", "u", "m", "d", "o", "t", "n",
  "Alert", "Button", "Space", "Table", "Typography", "Select", "Input", "Segmented", "Radio", "Form", "Drawer",
  "Switch", "AutoComplete", "TreeSelect", "DatePicker", "Card", "Suspense", "Collapse", "AlertConfigCenterPanel",
  "AlertInhibitionPanel", "MinusCircleOutlined", "PlusOutlined", "ReloadOutlined", "Link",
]);

const usedIds = new Set();
let m;
while ((m = idRe.exec(jsxChunks)) !== null) {
  const id = m[1];
  if (!skip.has(id) && id.length > 1) usedIds.add(id);
}

const hookLines = lines.slice(376, 2188);
let hookBody = hookLines.join("\n");
hookBody = hookBody.replace(
  "export function AlertMonitorPlatformPage() {\n  const [searchParams, setSearchParams] = useSearchParams();",
  `function useAlertMonitorPlatformState() {
  const navigate = useNavigate();
  const { tab: tabParam } = useParams();
  const [searchParams, setSearchParams] = useSearchParams();`,
);

const oldTabBlock = `  const tab: TabKey = useMemo(() => {
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

const newTabBlock = `  const tab: AlertMonitorTabKey = useMemo(() => normalizeAlertMonitorTab(tabParam), [tabParam]);

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

if (!hookBody.includes(oldTabBlock)) {
  console.error("Could not find old tab block");
  process.exit(1);
}
hookBody = hookBody.replace(oldTabBlock, newTabBlock);

const declared = new Set(["tab", "setTab", "projectContextId", "loading", "activeProjectName", "setProjectContext", "openHistoryTab"]);
const declRe = /(?:const|let|function)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)/g;
while ((m = declRe.exec(hookBody)) !== null) declared.add(m[1]);
const destructureRe = /const\s+\[([a-zA-Z_$][a-zA-Z0-9_$]*)\s*,\s*([a-zA-Z_$][a-zA-Z0-9_$]*)/g;
while ((m = destructureRe.exec(hookBody)) !== null) {
  declared.add(m[1]);
  declared.add(m[2]);
}

const returnKeys = [...new Set([...[...usedIds].filter((id) => declared.has(id)), "tab", "setTab", "projectContextId", "loading", "activeProjectName", "setProjectContext", "openHistoryTab"])].sort();

let helpersAndImports = lines.slice(0, 375).join("\n");
helpersAndImports = helpersAndImports
  .replace(/from "\.\.\//g, 'from "../../')
  .replace(/import "\.\.\//g, 'import "../../')
  .replace(
    'import { useCallback, useEffect, useMemo, useRef, useState, lazy, Suspense } from "react";',
    'import { useCallback, useEffect, useMemo, useRef, useState } from "react";',
  )
  .replace(
    'import { useSearchParams } from "react-router-dom";',
    "",
  )
  .replace(
    'import { PageTelemetryHeader } from "../components/page-telemetry-header";\n',
    "",
  )
  .replace(
    /const AlertInhibitionPanel = lazy[\s\S]*?\}\);\n\n/,
    "",
  )
  .replace(
    /const AlertConfigCenterPanel = lazy[\s\S]*?\}\);\n\n/,
    "",
  )
  .replace('type TabKey = "datasources"', 'type _TabKey = "datasources"');

helpersAndImports += `
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

const returnBlock = `  return {
${returnKeys.map((k) => `    ${k},`).join("\n")}
  };
}`;

const provider = `${helpersAndImports}\n${hookBody}\n\n${returnBlock}\n`;
if (!tabsOnly) {
  fs.writeFileSync(outPath, provider, "utf8");
  console.log("rebuilt platform-provider, return keys:", returnKeys.length);
} else {
  console.log("skip platform-provider (--tabs-only), return keys:", returnKeys.length);
}

// regen tabs + modals
const ANTD = new Set([
  "Alert", "Button", "Space", "Table", "Typography", "Select", "Input", "Segmented", "Radio", "Form", "Drawer",
  "Switch", "AutoComplete", "TreeSelect", "DatePicker", "Card", "Suspense", "Collapse", "message",
]);

function prefixJsx(body, prefix, keys) {
  let out = body;
  for (const id of [...keys].sort((a, b) => b.length - a.length)) {
    if (ANTD.has(id)) continue;
    out = out.replace(new RegExp(`\\b${id}\\b`, "g"), (match, offset, str) => {
      const before = str[offset - 1];
      if (before === "." || before === '"' || before === "'") return match;
      const afterSlice = str.slice(offset + match.length);
      if (/^\s*=/.test(afterSlice)) return match;
      return `${prefix}${id}`;
    });
  }
  return out;
}

function stripTabBody(raw, file) {
  let body = raw.trim();
  if (file === "inhibition-tab") {
    body = body.replace(/^children:\s*/, "").replace(/,\s*$/, "");
  }
  return body;
}

const tabsDir = path.join(root, "web/src/pages/alert-monitor/tabs");
fs.mkdirSync(tabsDir, { recursive: true });
const componentName = (file) =>
  file.split("-").map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join("");

for (const t of tabKeys) {
  const rawBody = stripTabBody(lines.slice(t.start - 1, t.end).join("\n"), t.file);
  const newBody = prefixJsx(rawBody, "ctx.", returnKeys);
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
  const ctx = useAlertMonitor() as Record<string, unknown>;
  return (
${newBody}
  );
}
`,
    "utf8",
  );
}

fs.writeFileSync(
  path.join(root, "web/src/pages/alert-monitor/modals.tsx"),
  `import {
  AutoComplete,
  Button,
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
import { useAlertMonitor } from "./context";

export function AlertMonitorModals() {
  const c = useAlertMonitor() as Record<string, unknown>;
  return (
    <>
${prefixJsx(modalsBody, "c.", returnKeys)}
    </>
  );
}
`,
  "utf8",
);

console.log("done");
