import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes("node_modules")) {
            if (id.includes("/react-dom/") || id.includes("/react-router-dom/") || id.includes("/react/")) {
              return "react-vendor";
            }
            if (id.includes("antd") || id.includes("@ant-design/icons")) {
              return "antd-vendor";
            }
            if (id.includes("axios") || id.includes("dayjs")) {
              return "shared-vendor";
            }
            if (id.includes("xterm")) {
              return "xterm-vendor";
            }
            if (id.includes("monaco-editor") || id.includes("@monaco-editor")) {
              return "monaco-vendor";
            }
            return;
          }
          if (id.includes("/components/k8s/yaml-crud-page")) {
            return "yaml-crud";
          }
          if (id.includes("alert-monitor-platform-page") || id.includes("alert-config-center-panel")) {
            return "alert-platform";
          }
        },
      },
    },
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
