import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

/** 判断 id 是否落在某个 npm 包路径下（避免误伤含 react 字样的其它包）。 */
function isPkg(id: string, pkg: string): boolean {
  const norm = id.replace(/\\/g, "/");
  return (
    norm.includes(`/node_modules/${pkg}/`) ||
    norm.includes(`/node_modules/${pkg}@`) ||
    // pnpm
    norm.includes(`/node_modules/.pnpm/${pkg}@`)
  );
}

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes("node_modules")) {
            if (id.includes("/components/k8s/yaml-crud-page")) {
              return "yaml-crud";
            }
            if (id.includes("alert-monitor-platform-page") || id.includes("alert-config-center-panel")) {
              return "alert-platform";
            }
            return;
          }
          // React 核心必须单独且优先加载；勿再把其余依赖塞进统一 vendor（易造成 createContext undefined）
          if (
            isPkg(id, "react") ||
            isPkg(id, "react-dom") ||
            isPkg(id, "react-router") ||
            isPkg(id, "react-router-dom") ||
            isPkg(id, "scheduler")
          ) {
            return "react-vendor";
          }
          if (isPkg(id, "antd") || isPkg(id, "@ant-design/icons") || id.includes("/@ant-design/")) {
            return "antd-vendor";
          }
          if (isPkg(id, "axios") || isPkg(id, "dayjs")) {
            return "shared-vendor";
          }
          if (id.includes("xterm")) {
            return "xterm-vendor";
          }
          if (id.includes("monaco-editor") || id.includes("@monaco-editor")) {
            return "monaco-vendor";
          }
          // 其余依赖交给 Rollup 按引用关系切分，保证对 react-vendor 的依赖顺序正确
        },
      },
    },
    chunkSizeWarningLimit: 1600,
  },
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
        ws: true,
      },
      "/swagger": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: "0.0.0.0",
    port: 4173,
  },
});
