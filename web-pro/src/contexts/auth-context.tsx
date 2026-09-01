// @ts-nocheck
import { useModel } from '@umijs/max';
import { createContext, useContext, useMemo, type PropsWithChildren } from 'react';
import type { EmailLoginPayload, LoginResult, PasswordLoginPayload, UserItem } from '../types/api';
import { emailLogin, getCurrentUser, logout as logoutRequest, passwordLogin } from '../services/auth';
import { clearAuthStorage, setUser } from '../services/storage';

interface AuthContextValue {
  user: UserItem | null;
  token: string;
  loading: boolean;
  isAuthenticated: boolean;
  passwordLoginAction: (payload: PasswordLoginPayload) => Promise<void>;
  emailLoginAction: (payload: EmailLoginPayload) => Promise<void>;
  logoutAction: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

function mapInitialUser(raw: Record<string, unknown> | undefined): UserItem | null {
  if (!raw || typeof raw.id !== 'number') return null;
  return raw as unknown as UserItem;
}

/** Pro 壳层下兼容旧版 useAuth */
export function AuthProvider({ children }: PropsWithChildren) {
  const { initialState, setInitialState } = useModel('@@initialState');
  const user = mapInitialUser(initialState?.currentUser as Record<string, unknown> | undefined);
  const isAuthenticated = Boolean(user?.id);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      token: '',
      loading: false,
      isAuthenticated,
      passwordLoginAction: async (payload) => {
        const result = await passwordLogin(payload);
        const next = { ...result.user, must_change_password: Boolean(result.must_change_password) };
        setUser(next);
        await setInitialState((s) => ({
          ...s,
          currentUser: {
            name: next.nickname || next.username,
            userid: String(next.id),
            access: next.roles?.some((r) => r.code === 'super-admin') ? 'admin' : 'user',
            ...next,
          },
        }));
      },
      emailLoginAction: async (payload) => {
        const result = await emailLogin(payload);
        const next = result.user;
        setUser(next);
        await setInitialState((s) => ({ ...s, currentUser: { name: next.nickname || next.username, userid: String(next.id), ...next } }));
      },
      logoutAction: async () => {
        try {
          await logoutRequest();
        } finally {
          clearAuthStorage();
          await setInitialState((s) => ({ ...s, currentUser: undefined }));
        }
      },
      refreshUser: async () => {
        const profile = await getCurrentUser();
        setUser(profile);
        await setInitialState((s) => ({
          ...s,
          currentUser: { name: profile.nickname || profile.username, userid: String(profile.id), ...profile },
        }));
      },
    }),
    [user, isAuthenticated, setInitialState],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
