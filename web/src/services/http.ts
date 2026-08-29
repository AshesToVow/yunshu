import axios, { type AxiosError, type InternalAxiosRequestConfig } from "axios";
import { message } from "antd";
import type { ApiResponse } from "../types/api";
import { clearAuthStorage } from "./storage";
import { resolveApiErrorDisplayMessage } from "../utils/api-error-messages";

declare module "axios" {
  interface AxiosRequestConfig {
    silentErrorToast?: boolean;
    /** 跳过 401→refresh 重试（避免 refresh 自身死循环） */
    skipAuthRefresh?: boolean;
  }

  interface InternalAxiosRequestConfig {
    silentErrorToast?: boolean;
    skipAuthRefresh?: boolean;
  }
}

export const HTTP_TIMEOUT_DEFAULT = 30000;
/** K8s 列表/详情会聚合 metrics、全集群 Pod 等，后端耗时常超过 15s */
export const HTTP_TIMEOUT_K8S = 60000;
/** Loggie Agent 安装含二进制下载/上传，可能较慢 */
export const HTTP_TIMEOUT_LOGGIE_INSTALL = 300000;

export const http = axios.create({
  baseURL: "/api/v1",
  timeout: HTTP_TIMEOUT_DEFAULT,
  withCredentials: true,
});

/** 与 K8s API 代理相关的慢路径（自动延长 axios 超时） */
const K8S_SLOW_PATH =
  /^\/(pods|nodes|namespaces|configmaps|secrets|deployments|statefulsets|daemonsets|jobs|cronjobs|events|ingresses|k8s-services|network-policies|horizontal-pod-autoscalers|persistentvolumeclaims|persistentvolumes|storageclasses|crds|serviceaccounts|helm|rbac|crs|k8s)(\/|$)/;

const CLUSTER_SLOW_PATH = /\/clusters\/\d+\/(namespaces|status|component-statuses|api-resources)/;
const LOGGIE_INSTALL_PATH = /\/projects\/\d+\/loggie\/install/;

function resolveRequestTimeout(url: string, configured?: number): number | undefined {
  const path = url.split("?")[0] ?? url;
  if (LOGGIE_INSTALL_PATH.test(path)) {
    return HTTP_TIMEOUT_LOGGIE_INSTALL;
  }
  if (K8S_SLOW_PATH.test(path) || CLUSTER_SLOW_PATH.test(path)) {
    return HTTP_TIMEOUT_K8S;
  }
  return configured;
}

function toastOnce(key: string, content: string) {
  message.error({ content, key });
}

function nextRequestId(): string {
  try {
    return crypto.randomUUID();
  } catch {
    return `req-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
  }
}

const sessionExpiredCodes = new Set(["10002", "10008", "10009", "10010", "10011", "10014"]);

let refreshPromise: Promise<boolean> | null = null;

async function tryRefreshSession(): Promise<boolean> {
  if (!refreshPromise) {
    refreshPromise = http
      .post("/auth/refresh", null, { skipAuthRefresh: true, silentErrorToast: true })
      .then(() => true)
      .catch(() => false)
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

function forceLoginRedirect() {
  clearAuthStorage();
  window.sessionStorage.removeItem("yunshu-session");
  if (window.location.pathname !== "/login") {
    toastOnce("auth-expired", "登录已失效，请重新登录");
    window.location.href = "/login";
  }
}

http.interceptors.request.use((config) => {
  // Cookie 会话：不再附加 Authorization Bearer（HttpOnly，JS 不可读）
  if (!config.headers["X-Request-ID"]) {
    config.headers["X-Request-ID"] = nextRequestId();
  }
  const url = String(config.url ?? "");
  const resolvedTimeout = resolveRequestTimeout(url, config.timeout);
  if (resolvedTimeout != null) {
    config.timeout = resolvedTimeout;
  }
  return config;
});

http.interceptors.response.use(
  (response) => response.data,
  async (error: AxiosError<{ message?: string; error_code?: string }>) => {
    const status = error.response?.status;
    const rawData = error.response?.data;
    const resolved = resolveApiErrorDisplayMessage(rawData?.error_code, rawData?.message);
    const isTimeout = error.code === "ECONNABORTED" || /timeout of \d+ms exceeded/i.test(String(error.message ?? ""));
    const errorMessage = isTimeout
      ? "请求超时：Agent 安装较慢，请确认离线包 deploy/loggie/binary/loggie 存在且目标机 SSH 可达后重试"
      : resolved || error.message || "请求失败";
    const cfg = error.config as InternalAxiosRequestConfig | undefined;
    const silentErrorToast = Boolean(cfg?.silentErrorToast);
    const errorCode = rawData?.error_code ?? "";

    if (status === 401 && sessionExpiredCodes.has(errorCode) && cfg && !cfg.skipAuthRefresh) {
      const url = String(cfg.url ?? "");
      if (!url.includes("/auth/login") && !url.includes("/auth/email-login") && !url.includes("/auth/refresh")) {
        const ok = await tryRefreshSession();
        if (ok) {
          return http.request(cfg);
        }
        forceLoginRedirect();
        return Promise.reject(error);
      }
    }

    if (silentErrorToast) {
      return Promise.reject(error);
    }

    if (status === 401 && sessionExpiredCodes.has(errorCode)) {
      forceLoginRedirect();
    } else if (status === 401) {
      toastOnce("http-error", errorMessage);
    } else if (status === 403) {
      toastOnce("forbidden", typeof errorMessage === "string" ? errorMessage : "无访问权限");
    } else if (isTimeout) {
      toastOnce("http-timeout", errorMessage);
    } else {
      const requestUrl = String(cfg?.url ?? "");
      const isExistenceProbe =
        typeof errorMessage === "string" &&
        errorMessage.includes("不存在") &&
        (requestUrl.includes("/detail") || requestUrl.includes("detail"));
      if (!isExistenceProbe) {
        toastOnce("http-error", errorMessage);
      }
    }

    return Promise.reject(error);
  },
);

export async function getData<T>(promise: Promise<ApiResponse<T>>) {
  const result = await promise;
  return result.data;
}

/** 从 axios 错误中取出后端 Body.Message（与 response.Error 业务话术对齐）；非 axios 时回退 Error.message。 */
export function extractApiErrorMessage(error: unknown, fallback = "请求失败"): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as { message?: string; error_code?: string } | undefined;
    const resolved = resolveApiErrorDisplayMessage(data?.error_code, data?.message);
    if (resolved) {
      return resolved;
    }
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}
