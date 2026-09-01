import type { RequestOptions } from '@@/plugin-request/request';
import type { RequestConfig } from '@umijs/max';
import { message } from 'antd';

type YunshuBody<T = unknown> = {
  code: number;
  message: string;
  error_code?: string;
  data: T;
};

const sessionExpiredCodes = new Set(['10002', '10008', '10009', '10010', '10011', '10014']);

function forceLoginRedirect() {
  if (window.location.pathname !== '/user/login') {
    message.error({ content: '登录已失效，请重新登录', key: 'auth-expired' });
    window.location.href = `/user/login?redirect=${encodeURIComponent(window.location.pathname)}`;
  }
}

function isYunshuSuccess(code: number) {
  return code === 200 || code === 201;
}

export const errorConfig: RequestConfig = {
  withCredentials: true,
  errorConfig: {
    errorHandler: (error: any, opts: any) => {
      if (opts?.skipErrorHandler) throw error;
      if (error.name === 'BizError') {
        message.error(error.message || '请求失败');
        return;
      }
      if (error.response) {
        const data = error.response.data as YunshuBody | undefined;
        const errorCode = data?.error_code ?? '';
        const status = error.response.status;
        if (status === 401 && sessionExpiredCodes.has(errorCode)) {
          forceLoginRedirect();
          return;
        }
        message.error(data?.message || `请求失败 (${status})`);
        return;
      }
      message.error(error.message || '网络异常，请稍后重试');
    },
  },
  requestInterceptors: [
    (config: RequestOptions) => {
      if (!config.headers) config.headers = {};
      if (!config.headers['X-Request-ID']) {
        try {
          config.headers['X-Request-ID'] = crypto.randomUUID();
        } catch {
          config.headers['X-Request-ID'] = `req-${Date.now()}`;
        }
      }
      return config;
    },
  ],
  responseInterceptors: [
    (response) => {
      const body = response.data as YunshuBody;
      if (body && typeof body.code === 'number') {
        if (!isYunshuSuccess(body.code)) {
          const err = new Error(body.message || '请求失败') as Error & { name: string };
          err.name = 'BizError';
          throw err;
        }
        return { ...response, data: body.data } as typeof response;
      }
      return response;
    },
  ],
};
