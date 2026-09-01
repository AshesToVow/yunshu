import { request } from '@umijs/max';

const PREFIX = '/api/v1';

export async function getCurrentUser(options?: { skipErrorHandler?: boolean }) {
  return request<YunshuAPI.UserItem>(`${PREFIX}/auth/me`, {
    method: 'GET',
    skipErrorHandler: options?.skipErrorHandler,
    withCredentials: true,
  });
}

export async function passwordLogin(body: YunshuAPI.PasswordLoginPayload) {
  return request<YunshuAPI.LoginResult>(`${PREFIX}/auth/login`, {
    method: 'POST',
    data: body,
    skipErrorHandler: true,
    withCredentials: true,
  });
}

export async function sendPasswordLoginCode(username: string) {
  return request<YunshuAPI.SendPasswordLoginCodeResult>(`${PREFIX}/auth/password-login-code`, {
    method: 'POST',
    data: { username },
    skipErrorHandler: true,
    withCredentials: true,
  });
}

export async function logout() {
  return request<{ message?: string }>(`${PREFIX}/auth/logout`, {
    method: 'POST',
    skipErrorHandler: true,
    withCredentials: true,
  });
}

export async function refreshSession() {
  return request<unknown>(`${PREFIX}/auth/refresh`, {
    method: 'POST',
    skipErrorHandler: true,
    withCredentials: true,
  });
}
