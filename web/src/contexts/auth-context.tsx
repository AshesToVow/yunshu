import { createContext, useContext, useEffect, useState } from "react";
import type { PropsWithChildren } from "react";
import type { EmailLoginPayload, LoginResult, PasswordLoginPayload, UserItem } from "../types/api";
import { emailLogin, getCurrentUser, logout as logoutRequest, passwordLogin } from "../services/auth";
import { clearAuthStorage, clearToken, getUser, setUser } from "../services/storage";

interface AuthContextValue {
  user: UserItem | null;
  /** @deprecated Cookie 会话下恒为空；保留字段避免调用方编译失败 */
  token: string;
  loading: boolean;
  isAuthenticated: boolean;
  passwordLoginAction: (payload: PasswordLoginPayload) => Promise<void>;
  emailLoginAction: (payload: EmailLoginPayload) => Promise<void>;
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
      } catch {
        if (!cancelled) {
          clearSessionMark();
          setUserState(null);
          setAuthenticated(false);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    void bootstrap();
    return () => {
      cancelled = true;
    };
  }, []);

  function applyLoginResult(result: LoginResult) {
    const nextUser = {
      ...result.user,
      must_change_password: Boolean(result.must_change_password || result.user?.must_change_password),
    };
    setUser(nextUser);
    setUserState(nextUser);
    setAuthenticated(true);
    markSession();
  }

  async function passwordLoginAction(payload: PasswordLoginPayload) {
    const result = await passwordLogin(payload);
    applyLoginResult(result);
  }

  async function emailLoginAction(payload: EmailLoginPayload) {
    const result = await emailLogin(payload);
    applyLoginResult(result);
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
    }
  }

  async function refreshUser() {
    if (!isAuthenticated) return;
    const profile = await getCurrentUser();
    setUser(profile);
    setUserState(profile);
  }

  return (
    <AuthContext.Provider
      value={{
        user,
        token: "",
        loading,
        isAuthenticated,
        passwordLoginAction,
        emailLoginAction,
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
