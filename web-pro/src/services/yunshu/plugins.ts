import { request } from '@umijs/max';

const PREFIX = '/api/v1';

export async function listPlugins(options?: { skipErrorHandler?: boolean }) {
  return request<YunshuAPI.PluginListResult>(`${PREFIX}/plugins`, {
    method: 'GET',
    skipErrorHandler: options?.skipErrorHandler,
    withCredentials: true,
  });
}
