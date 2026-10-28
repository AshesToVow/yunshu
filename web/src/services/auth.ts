import type {
  ChangePasswordPayload,
  EmailLoginPayload,
  LoginResult,
  MessageData,
  PasswordLoginPayload,
  RegisterPayload,
  RegisterResult,
  SendEmailCodePayload,
  SendEmailCodeResult,
  SendPasswordLoginCodePayload,
  SendPasswordLoginCodeResult,
  UpdateProfilePayload,
  UserItem,
  VerifyMFAPayload,
} from "../types/api";
import { getData, http } from "./http";

export function sendEmailCode(payload: SendEmailCodePayload) {
  return getData<SendEmailCodeResult>(http.post("/auth/verification-code", payload, { silentErrorToast: true }));
}

export function sendLoginCodeByUsername(username: string) {
  return getData<SendEmailCodeResult>(http.post("/auth/login-code", { username }));
}

export function sendPasswordLoginCode(payload: SendPasswordLoginCodePayload) {
  return getData<SendPasswordLoginCodeResult>(
    http.post("/auth/password-login-code", payload, { silentErrorToast: true }),
  );
}

export function passwordLogin(payload: PasswordLoginPayload) {
  return getData<LoginResult>(http.post("/auth/login", payload, { silentErrorToast: true }));
}

export function emailLogin(payload: EmailLoginPayload) {
  return getData<LoginResult>(http.post("/auth/email-login", payload, { silentErrorToast: true }));
}

export function verifyMfaLogin(payload: VerifyMFAPayload) {
  return getData<LoginResult>(http.post("/auth/mfa/verify", payload, { silentErrorToast: true }));
}

export interface MfaSetupResult {
  secret: string;
  otpauth_url: string;
}

export function setupMfa(password: string) {
  return getData<MfaSetupResult>(http.post("/auth/mfa/setup", { password }, { silentErrorToast: true }));
}

export function enableMfa(code: string) {
  return getData<{ recovery_codes: string[] }>(
    http.post("/auth/mfa/enable", { code }, { silentErrorToast: true }),
  );
}

export function disableMfa(payload: { password: string; code: string }) {
  return getData<MessageData>(http.post("/auth/mfa/disable", payload, { silentErrorToast: true }));
}

export function registerByEmail(payload: RegisterPayload) {
  // 注册失败提示由页面自行处理，避免拦截器通用文案覆盖后端细节。
  return getData<RegisterResult>(http.post("/auth/register", payload, { silentErrorToast: true }));
}

export function logout() {
  return getData<MessageData>(http.post("/auth/logout", null, { skipAuthRefresh: true, silentErrorToast: true }));
}

export function getCurrentUser() {
  return getData<UserItem>(http.get("/auth/me", { silentErrorToast: true }));
}

export interface MyPermissionCapability {
  resource: string;
  action: string;
}

export interface MyPermissions {
  super_admin: boolean;
  permissions: MyPermissionCapability[];
}

/** 当前用户经 Casbin 推导出的有效能力（含角色继承），用于前端按钮级门禁。 */
export function getMyPermissions() {
  return getData<MyPermissions>(http.get("/auth/permissions", { silentErrorToast: true }));
}

export function updateProfile(payload: UpdateProfilePayload) {
  return getData<UserItem>(http.put("/auth/me", payload));
}

export function changePassword(payload: ChangePasswordPayload) {
  return getData<MessageData>(http.put("/auth/password", payload));
}

export function getPasswordPolicy() {
  return getData<import("../types/api").PasswordPolicy>(http.get("/auth/password-policy", { silentErrorToast: true }));
}

export interface HealthData {
  status: string;
  version: string;
  uptime: number;
}

export function getHealth() {
  return getData<HealthData>(http.get("/health"));
}
