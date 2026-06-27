/**
 * One-off: extract alert-monitor tabs + modals from platform-provider.tsx
 */
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.join(__dirname, "..");
const providerPath = path.join(root, "web/src/pages/alert-monitor/platform-provider.tsx");
const lines = fs.readFileSync(providerPath, "utf8").split(/\r?\n/);

const tabKeys = [
  { key: "datasources", file: "datasources-tab", start: 2226, end: 2244 },
  { key: "policies", file: "policies-tab", start: 2249, end: 2301 },
  { key: "history", file: "history-tab", start: 2306, end: 2325 },
  { key: "inhibition", file: "inhibition-tab", start: 2330, end: 2334 },
  { key: "silences", file: "silences-tab", start: 2339, end: 2435 },
  { key: "rules", file: "rules-tab", start: 2440, end: 2491 },
  { key: "cloud-expiry", file: "cloud-expiry-tab", start: 2496, end: 2538 },
  { key: "promql", file: "promql-tab", start: 2543, end: 2622 },
];

const modalsBody = lines.slice(2626, 3421).join("\n");
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

const hookBody = lines.slice(381, 2193).join("\n");
const declared = new Set();
const declRe = /(?:const|let|function)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)/g;
while ((m = declRe.exec(hookBody)) !== null) declared.add(m[1]);
const destructureRe = /const\s+\[([a-zA-Z_$][a-zA-Z0-9_$]*)/g;
while ((m = destructureRe.exec(hookBody)) !== null) declared.add(m[1]);

const returnKeys = [...usedIds].filter((id) => declared.has(id)).sort();
console.log("return keys count", returnKeys.length);

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
fs.mkdirSync(tabsDir, { recursive: true });

const componentName = (file) =>
  file.split("-").map((p) => p.charAt(0).toUpperCase() + p.slice(1)).join("");

for (const t of tabKeys) {
  const rawBody = lines.slice(t.start - 1, t.end).join("\n");
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
  console.log("wrote tab", t.file);
}

const modalsJsx = prefixJsx(modalsBody, "c.", returnKeys);
const modalsPath = path.join(root, "web/src/pages/alert-monitor/modals.tsx");
fs.writeFileSync(
  modalsPath,
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
console.log("wrote modals.tsx");

const beforeReturn = lines.slice(0, 2194).join("\n");
const afterReturn = lines.slice(3424).join("\n");
const returnBlock = `  return {
${returnKeys.map((k) => `    ${k},`).join("\n")}
  };`;

let newProvider = `${beforeReturn}\n${returnBlock}\n${afterReturn}`;
if (!newProvider.includes('from "./modals"')) {
  newProvider = newProvider.replace(
    'import { AlertMonitorLayout } from "./layout";',
    'import { AlertMonitorLayout } from "./layout";\nimport { AlertMonitorModals } from "./modals";',
  );
}
fs.writeFileSync(providerPath, newProvider, "utf8");
console.log("updated platform-provider.tsx");
console.log("done");
