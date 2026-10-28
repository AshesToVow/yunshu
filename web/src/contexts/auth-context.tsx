import { createContext, useCallback, useContext, useEffect, useState } from "react";
import type { PropsWithChildren } from "react";
import type { EmailLoginPayload, LoginResult, PasswordLoginPayload, UserItem } from "../types/api";
import {
  emailLogin,
  getCurrentUser,
  getMyPermissions,
  logout as logoutRequest,
  passwordLogin,
  verifyMfaLogin,
} from "../services/auth";
import { clearAuthStorage, clearToken, getUser, setUser } from "../services/storage";
import {
  buildCapabilities,
  EMPTY_CAPABILITIES,
  hasCapability,
} from "../modules/permission-capability";
import type { PermissionCapabilities } from "../modules/permission-capability";

interface AuthContextValue {
  user: UserItem | null;
  loading: boolean;
  isAuthenticated: boolean;
  /** 当前用户的有效能力集合；未加载时按放行处理，后端 Authorize 仍是最终闸门。 */
  capabilities: PermissionCapabilities;
  can: (action: string, resource: string) => boolean;
  passwordLoginAction: (payload: PasswordLoginPayload) => Promise<LoginResult>;
  emailLoginAction: (payload: EmailLoginPayload) => Promise<LoginResult>;
  completeMfaAction: (challenge: string, code: string) => Promise<LoginResult>;
  logoutAction: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

const SESSION_FLAG = "yunshu-session";

function markSession() {
  window.sessionStorage.setItem(SESSION_FLAG, "1");
}

function clearSessionMark() {
  window.sessionStorage.removeItem(SESSION_FLAG);
}

export function AuthProvider({ children }: PropsWithChildren) {
  const [user, setUserState] = useState<UserItem | null>(() => getUser());
  const [loading, setLoading] = useState(true);
  const [isAuthenticated, setAuthenticated] = useState(false);
  const [capabilities, setCapabilities] = useState<PermissionCapabilities>(EMPTY_CAPABILITIES);

  const loadCapabilities = useCallback(async () => {
    try {
      const data = await getMyPermissions();
      setCapabilities(
        buildCapabilities({ superAdmin: data.super_admin, permissions: data.permissions }),
      );
    } catch {
      // 能力接口失败不阻塞登录：退化为放行，真正的越权由后端拦截。
      setCapabilities(EMPTY_CAPABILITIES);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      // 迁移：清历史 localStorage JWT；用户资料仍以 /auth/me 为准
      clearToken();
      try {
        const profile = await getCurrentUser();
        if (cancelled) return;
        setUser(profile);
        setUserState(profile);
        setAuthenticated(true);
        markSession();
        void loadCapabilities();
      } catch {
        if (!cancelled) {
          clearSessionMark();
          setUserState(null);
          setAuthenticated(false);
          setCapabilities(EMPTY_CAPABILITIES);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void bootstrap();
    return () => {
      cancelled = true;
    };
  }, [loadCapabilities]);

  function applyLoginResult(result: LoginResult) {
    const resultUser = result.user;
    if (!resultUser) return;
    const nextUser = {
      ...resultUser,
      must_change_password: Boolean(result.must_change_password || resultUser.must_change_password),
    };
    setUser(nextUser);
    setUserState(nextUser);
    setAuthenticated(true);
    markSession();
    void loadCapabilities();
  }

  async function passwordLoginAction(payload: PasswordLoginPayload) {
    const result = await passwordLogin(payload);
    if (!result.mfa_required) applyLoginResult(result);
    return result;
  }

  async function emailLoginAction(payload: EmailLoginPayload) {
    const result = await emailLogin(payload);
    if (!result.mfa_required) applyLoginResult(result);
    return result;
  }

  async function completeMfaAction(challenge: string, code: string) {
    const result = await verifyMfaLogin({ challenge, code });
    applyLoginResult(result);
    return result;
  }

  async function logoutAction() {
    try {
      await logoutRequest();
    } catch {
      // keep local cleanup even if the backend token has already expired
    } finally {
      clearAuthStorage();
      clearSessionMark();
      setUserState(null);
      setAuthenticated(false);
      setCapabilities(EMPTY_CAPABILITIES);
    }
  }

  async function refreshUser() {
    if (!isAuthenticated) return;
    const profile = await getCurrentUser();
    setUser(profile);
    setUserState(profile);
  }

  const can = useCallback(
    (action: string, resource: string) => hasCapability(capabilities, action, resource),
    [capabilities],
  );

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        isAuthenticated,
        capabilities,
        can,
        passwordLoginAction,
        emailLoginAction,
        completeMfaAction,
        logoutAction,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
