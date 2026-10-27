import { loader } from "@monaco-editor/react";
import * as monaco from "monaco-editor/esm/vs/editor/editor.api";
// 编辑器特性全集（不含任何语言包，语言按需引入以控制构建内存与包体）
import "monaco-editor/esm/vs/editor/editor.all.js";
import "monaco-editor/esm/vs/basic-languages/sql/sql.contribution.js";
import "monaco-editor/esm/vs/basic-languages/yaml/yaml.contribution.js";
import "monaco-editor/esm/vs/basic-languages/html/html.contribution.js";
import "monaco-editor/esm/vs/basic-languages/shell/shell.contribution.js";
import editorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker";

// 本地打包 monaco，避免 @monaco-editor/react 默认走 jsDelivr CDN（内网不可达且被 CSP script-src 'self' 拦截）。
self.MonacoEnvironment = {
  getWorker() {
    return new editorWorker();
  },
};

loader.config({ monaco });
