import { request } from '@umijs/max';

const PREFIX = '/api/v1';

export async function getMenuTree(options?: { skipErrorHandler?: boolean }) {
  return request<YunshuAPI.MenuItem[]>(`${PREFIX}/menus/tree`, {
    method: 'GET',
    skipErrorHandler: options?.skipErrorHandler,
    withCredentials: true,
  });
}
