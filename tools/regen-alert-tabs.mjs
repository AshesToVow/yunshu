/**
 * Regenerate alert-monitor tabs + modals from original monolithic page.
 */
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.join(__dirname, "..");
const sourcePath = path.join(root, "web/src/pages/alert-monitor-platform-page.tsx");
const providerPath = path.join(root, "web/src/pages/alert-monitor/platform-provider.tsx");
const sourceLines = fs.readFileSync(sourcePath, "utf8").split(/\r?\n/);
const providerLines = fs.readFileSync(providerPath, "utf8").split(/\r?\n/);

const tabKeys = [
  { file: "datasources-tab", start: 2221, end: 2237 },
  { file: "policies-tab", start: 2244, end: 2295 },
  { file: "history-tab", start: 2301, end: 2318 },
  { file: "inhibition-tab", start: 2325, end: 2327 },
  { file: "silences-tab", start: 2334, end: 2429 },
  { file: "rules-tab", start: 2435, end: 2486 },
  { file: "cloud-expiry-tab", start: 2491, end: 2532 },
  { file: "promql-tab", start: 2538, end: 2617 },
];

const modalsBody = sourceLines.slice(2620, 3415).join("\n");
const jsxChunks = [
  ...tabKeys.map((t) => sourceLines.slice(t.start - 1, t.end).join("\n")),
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

const hookStart = providerLines.findIndex((l) => l.includes("function useAlertMonitorPlatformState"));
const hookEnd = providerLines.findIndex((l, i) => i > hookStart && l.trim() === "return {");
const hookBody = providerLines.slice(hookStart, hookEnd).join("\n");

const declared = new Set();
const declRe = /(?:const|let|function)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)/g;
while ((m = declRe.exec(hookBody)) !== null) declared.add(m[1]);
const destructureRe = /const\s+\[([a-zA-Z_$][a-zA-Z0-9_$]*)\s*,\s*([a-zA-Z_$][a-zA-Z0-9_$]*)/g;
while ((m = destructureRe.exec(hookBody)) !== null) {
  declared.add(m[1]);
  declared.add(m[2]);
}

const returnKeys = [...usedIds].filter((id) => declared.has(id)).sort();
console.log("return keys", returnKeys.length);

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
      const after = str[offset + match.length];
      if (before === "." || before === '"' || before === "'" || after === ":") return match;
      return `${prefix}${id}`;
    });
  }
  return out;
}

const tabsDir = path.join(root, "web/src/pages/alert-monitor/tabs");
const componentName = (file) =>
  file.split("-").map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join("");

for (const t of tabKeys) {
  const rawBody = sourceLines.slice(t.start - 1, t.end).join("\n");
  const newBody = prefixJsx(rawBody, "ctx.", returnKeys);
  const name = componentName(t.file);
  const needsInhibition = t.file === "inhibition-tab";
  const needsConfig = t.file === "policies-tab" || t.file === "history-tab";
  const content = `import {
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
`;
  fs.writeFileSync(path.join(tabsDir, `${t.file}.tsx`), content, "utf8");
}

const modalsJsx = prefixJsx(modalsBody, "c.", returnKeys);
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
${modalsJsx}
    </>
  );
}
`,
  "utf8",
);

// Merge any missing return keys into platform-provider
const returnStart = providerLines.findIndex((l) => l.trim() === "return {");
const returnEnd = providerLines.findIndex((l, i) => i > returnStart && l.trim() === "};");
const existingKeys = new Set();
for (let i = returnStart + 1; i < returnEnd; i++) {
  const km = providerLines[i].match(/^\s*([a-zA-Z0-9_]+),/);
  if (km) existingKeys.add(km[1]);
}
const missing = returnKeys.filter((k) => !existingKeys.has(k));
if (missing.length) {
  const insert = missing.map((k) => `    ${k},`).join("\n");
  providerLines.splice(returnEnd, 0, insert);
  fs.writeFileSync(providerPath, providerLines.join("\n"), "utf8");
  console.log("added missing return keys", missing.length);
}

console.log("regenerated tabs + modals");
