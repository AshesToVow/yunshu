import type { Settings as LayoutSettings } from '@ant-design/pro-components';
import { SettingDrawer } from '@ant-design/pro-components';
import type { RequestConfig, RunTimeLayoutConfig } from '@umijs/max';
import { history, Link } from '@umijs/max';
import React from 'react';
import { AvatarDropdown, ErrorBoundary } from '@/components';
import { BRAND_NAME, BRAND_PRIMARY } from '@/constants/brand';
import { getCurrentUser } from '@/services/yunshu/auth';
import { fetchProLayoutMenu } from '@/utils/yunshu-menu';
import defaultSettings from '../config/defaultSettings';
import { errorConfig } from './requestErrorConfig';

const loginPath = '/user/login';

function mapUser(user: YunshuAPI.UserItem): API.CurrentUser {
  return {
    name: user.nickname || user.username,
    userid: String(user.id),
    access: user.roles?.some((r: YunshuAPI.RoleItem) => r.code === 'super-admin') ? 'admin' : 'user',
    ...user,
  };
}

export async function getInitialState(): Promise<{
  settings?: Partial<LayoutSettings>;
  currentUser?: API.CurrentUser;
  loading?: boolean;
  fetchUserInfo?: () => Promise<API.CurrentUser | undefined>;
  settingDrawerOpen?: boolean;
}> {
  const fetchUserInfo = async () => {
    try {
      const user = await getCurrentUser({ skipErrorHandler: true });
      return mapUser(user);
    } catch {
      const { pathname, search, hash } = history.location;
      if (pathname !== loginPath) {
        history.replace(`${loginPath}?redirect=${encodeURIComponent(pathname + search + hash)}`);
      }
    }
    return undefined;
  };

  const { location } = history;
  if (location.pathname !== loginPath) {
    const currentUser = await fetchUserInfo();
    return {
      fetchUserInfo,
      currentUser,
      settings: defaultSettings as Partial<LayoutSettings>,
      settingDrawerOpen: false,
    };
  }
  return {
    fetchUserInfo,
    settings: defaultSettings as Partial<LayoutSettings>,
    settingDrawerOpen: false,
  };
}

export const layout: RunTimeLayoutConfig = ({ initialState, setInitialState }) => {
  return {
    logo: false,
    title: BRAND_NAME,
    menu: {
      locale: false,
      request: async () => {
        try {
          return await fetchProLayoutMenu();
        } catch {
          return [];
        }
      },
    },
    menuItemRender: (item, dom) => {
      if (item.path) {
        return (
          <Link to={item.path} prefetch>
            {dom}
          </Link>
        );
      }
      return dom;
    },
    actionsRender: () => [],
    avatarProps: {
      title: initialState?.currentUser?.name,
      render: (_, avatarChildren) => <AvatarDropdown>{avatarChildren}</AvatarDropdown>,
    },
    footerRender: false,
    onPageChange: () => {
      const { location } = history;
      if (!initialState?.currentUser && location.pathname !== loginPath) {
        history.replace(
          `${loginPath}?redirect=${encodeURIComponent(location.pathname + location.search + location.hash)}`,
        );
      }
    },
    bgLayoutImgList: [],
    links: [],
    ErrorBoundary,
    menuHeaderRender: undefined,
    childrenRender: (children) => (
      <>
        {children}
        <SettingDrawer
          disableUrlParams
          enableDarkTheme
          collapse={initialState?.settingDrawerOpen}
          onCollapseChange={(open) => {
            setInitialState((s) => ({ ...s, settingDrawerOpen: open }));
          }}
          settings={initialState?.settings}
          onSettingChange={(settings) => {
            setInitialState((s) => ({ ...s, settings }));
          }}
        />
      </>
    ),
    token: {
      colorPrimary: BRAND_PRIMARY,
    },
    ...initialState?.settings,
  };
};

export const request: RequestConfig = {
  ...errorConfig,
};

export function rootContainer(container: React.ReactNode) {
  return <ErrorBoundary>{container}</ErrorBoundary>;
}
