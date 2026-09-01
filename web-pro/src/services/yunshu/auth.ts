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

/** 图形验证码：后端返回 raw base64（无 data URL 前缀） */
export async function sendPasswordLoginCode(username: string) {
  return request<YunshuAPI.SendPasswordLoginCodeResult>(`${PREFIX}/auth/password-login-code`, {
    method: 'POST',
    data: { username },
    skipErrorHandler: true,
    withCredentials: true,
  });
}

export async function emailLogin(body: YunshuAPI.EmailLoginPayload) {
  return request<YunshuAPI.LoginResult>(`${PREFIX}/auth/email-login`, {
    method: 'POST',
    data: body,
    skipErrorHandler: true,
    withCredentials: true,
  });
}

export async function sendEmailCode(body: YunshuAPI.SendEmailCodePayload) {
  return request<YunshuAPI.SendEmailCodeResult>(`${PREFIX}/auth/verification-code`, {
    method: 'POST',
    data: body,
    skipErrorHandler: true,
    withCredentials: true,
  });
}

export async function registerByEmail(body: YunshuAPI.RegisterPayload) {
  return request<YunshuAPI.RegisterResult>(`${PREFIX}/auth/register`, {
    method: 'POST',
    data: body,
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

/** 将后端 raw base64 转为可用的 img src */
export function toCaptchaDataUrl(image: string | undefined | null): string | null {
  if (!image?.trim()) return null;
  const raw = image.trim();
  if (raw.startsWith('data:image/')) return raw;
  return `data:image/png;base64,${raw}`;
}
