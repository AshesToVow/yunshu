import {
  ApiOutlined,
  ApartmentOutlined,
  AuditOutlined,
  BgColorsOutlined,
  BulbFilled,
  BulbOutlined,
  CheckCircleOutlined,
  CloseOutlined,
  FullscreenOutlined,
  KubernetesOutlined,
  DatabaseOutlined,
  PieChartOutlined,
  HistoryOutlined,
  LoginOutlined,
  MenuOutlined,
  ReloadOutlined,
  SearchOutlined,
  SettingOutlined,
  TeamOutlined,
  UserOutlined,
  LogoutOutlined,
  DownOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Avatar, Button, Drawer, Dropdown, Layout, Menu, Select, Space, Spin, Switch, Tabs, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { BRAND_EN_NAME } from "../constants/brand";
import { LogStreamDockBar } from "../components/log-stream-dock-bar";
import { GlobalSearchModal } from "../components/global-search-modal";
import { useAuth } from "../contexts/auth-context";
import { LogStreamProvider } from "../contexts/log-stream-context";
import { getMenuTree } from "../services/menus";
import type { MenuItem } from "../services/menus";
import { resolveAppLocale } from "../i18n";
import { buildSiderMenuItems, matchMenuSelectedKey, type AntdMenuItem } from "../utils/admin-menu";
import { usePlugins } from "../contexts/plugin-context";
import { filterAntdMenuItems, filterMenuTreeByPlugins } from "../modules/filter-menu";

const { Content, Header, Sider } = Layout;
const UI_PREFS_KEY = "admin-ui-preferences";

type UIPreferences = {
  showRefresh: boolean;
  showFullscreen: boolean;
  showThemeToggle: boolean;
  compactContent: boolean;
  darkSider: boolean;
  darkHeader: boolean;
};

const defaultUIPreferences: UIPreferences = {
  showRefresh: true,
  showFullscreen: true,
  showThemeToggle: true,
  compactContent: false,
  darkSider: true,
  darkHeader: true,
};

function loadUIPreferences(): UIPreferences {
  try {
    const raw = window.localStorage.getItem(UI_PREFS_KEY);
    if (!raw) return defaultUIPreferences;
    const parsed = JSON.parse(raw) as Partial<UIPreferences>;
    return { ...defaultUIPreferences, ...parsed };
  } catch {
    return defaultUIPreferences;
  }
}

function buildFallbackMenuItems(t: (key: string, options?: { defaultValue?: string }) => string): MenuProps["items"] {
  const label = (path: string, fallback: string) => (
    <Link to={path}>{t(`menu.routes.${path}`, { defaultValue: fallback })}</Link>
  );
  return [
    { key: "/", icon: <PieChartOutlined />, label: label("/", "总览页面") },
    { key: "/clusters", icon: <KubernetesOutlined />, label: label("/clusters", "集群管理") },
    { key: "/pods", icon: <KubernetesOutlined />, label: label("/pods", "Pod 管理") },
    { key: "/users", icon: <TeamOutlined />, label: label("/users", "账号管理") },
    { key: "/roles", icon: <ApartmentOutlined />, label: label("/roles", "角色模板") },
    { key: "/permissions", icon: <ApiOutlined />, label: label("/permissions", "API管理") },
    { key: "/policies", icon: <AuditOutlined />, label: label("/policies", "授权管理") },
    { key: "/registrations", icon: <CheckCircleOutlined />, label: label("/registrations", "注册审核") },
    { key: "/menus", icon: <MenuOutlined />, label: label("/menus", "菜单管理") },
    {
      key: "/system",
      icon: <MenuOutlined />,
      label: t("menu.groups./system", { defaultValue: "系统管理" }),
      children: [
        { key: "/departments", icon: <ApartmentOutlined />, label: label("/departments", "组织架构") },
        { key: "/dict-entries", icon: <DatabaseOutlined />, label: label("/dict-entries", "数据字典") },
        { key: "/login-logs", icon: <LoginOutlined />, label: label("/login-logs", "登录日志") },
        { key: "/operation-logs", icon: <HistoryOutlined />, label: label("/operation-logs", "操作历史") },
        { key: "/banned-ips", icon: <ApiOutlined />, label: label("/banned-ips", "封禁 IP 管理") },
      ],
    },
  ];
}

function defaultOpenKeysFor(items: AntdMenuItem[]): string[] {
  const keys: string[] = [];
  function walk(list: AntdMenuItem[]) {
    for (const it of list) {
      if (it && typeof it === "object" && "children" in it && Array.isArray(it.children) && it.children.length > 0) {
        keys.push(String(it.key));
        walk(it.children as AntdMenuItem[]);
      }
    }
  }
  walk(items);
  return keys;
}

export function AdminLayout() {
  const { t, i18n } = useTranslation();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const { user, loading, logoutAction } = useAuth();
  const { isPluginEnabled } = usePlugins();
  const [menuTree, setMenuTree] = useState<MenuItem[] | null>(null);
  const [menuEpoch, setMenuEpoch] = useState(0);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [globalSearchOpen, setGlobalSearchOpen] = useState(false);
  const [settingsTab, setSettingsTab] = useState("appearance");
  const [uiPreferences, setUIPreferences] = useState<UIPreferences>(() => loadUIPreferences());
  const [themeMode, setThemeMode] = useState<"dark" | "light">(() => {
    const saved = window.localStorage.getItem("admin-theme-mode");
    return saved === "light" ? "light" : "dark";
  });
  const [accent, setAccent] = useState<string>(() => {
    return window.localStorage.getItem("admin-theme-accent") ?? "#e61919";
  });

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const tree: MenuItem[] = await getMenuTree();
        if (cancelled || !tree?.length) return;
        setMenuTree(tree);
      } catch {
        if (!cancelled) setMenuTree(null);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [isPluginEnabled]);

  const activeLocale = resolveAppLocale(i18n.language);

  const siderItems = useMemo(() => {
    if (menuTree?.length) {
      const filtered = filterMenuTreeByPlugins(menuTree, isPluginEnabled);
      const items = buildSiderMenuItems(filtered, t);
      if (items.length) return items;
    }
    return filterAntdMenuItems(buildFallbackMenuItems(t) as AntdMenuItem[], isPluginEnabled);
  }, [menuTree, isPluginEnabled, t, activeLocale]);

  const selectedKey = useMemo(() => {
    const items = (siderItems ?? []) as AntdMenuItem[];
    return matchMenuSelectedKey(pathname, items);
  }, [pathname, siderItems]);

  const defaultOpenKeys = useMemo(
    () => defaultOpenKeysFor((siderItems ?? []) as AntdMenuItem[]),
    [siderItems],
  );
  const menuTheme = themeMode === "dark" ? "dark" : "light";
  const layoutClassName = [
    "admin-shell",
    themeMode === "dark" ? "theme-dark" : "theme-light",
    uiPreferences.compactContent ? "layout-compact" : "",
    uiPreferences.darkSider ? "layout-dark-sider" : "layout-soft-sider",
    uiPreferences.darkHeader ? "layout-dark-header" : "layout-soft-header",
  ]
    .filter(Boolean)
    .join(" ");

  const pageTitle = useMemo(() => {
    const layoutTitle = t(`layout.routes.${pathname}`, { defaultValue: "" });
    if (layoutTitle) return layoutTitle;
    const menuTitle = t(`menu.routes.${pathname}`, { defaultValue: "" });
    if (menuTitle) return menuTitle;
    return t("app.console");
  }, [pathname, t, activeLocale]);

  async function handleLogout() {
    await logoutAction();
    navigate("/login", { replace: true });
  }

  function handleRefresh() {
    window.location.reload();
  }

  async function handleToggleFullscreen() {
    try {
      if (document.fullscreenElement) {
        await document.exitFullscreen();
        return;
      }
      await document.documentElement.requestFullscreen();
    } catch {
      // Ignore unsupported fullscreen APIs.
    }
  }

  function applyThemeMode(mode: "dark" | "light") {
    setThemeMode(mode);
    window.localStorage.setItem("admin-theme-mode", mode);
    window.dispatchEvent(new CustomEvent("admin-theme-mode-change", { detail: { mode } }));
  }

  function handleToggleMode() {
    applyThemeMode(themeMode === "dark" ? "light" : "dark");
  }

  function applyAccent(next: string) {
    setAccent(next);
    window.localStorage.setItem("admin-theme-accent", next);
    document.documentElement.style.setProperty("--admin-accent", next);
    window.dispatchEvent(new CustomEvent("admin-theme-accent-change", { detail: { accent: next } }));
  }

  function patchUIPreferences(patch: Partial<UIPreferences>) {
    setUIPreferences((prev) => {
      const next = { ...prev, ...patch };
      window.localStorage.setItem(UI_PREFS_KEY, JSON.stringify(next));
      return next;
    });
  }

  async function handleClearCacheAndLogout() {
    const mode = window.localStorage.getItem("admin-theme-mode");
    const accent = window.localStorage.getItem("admin-theme-accent");
    window.localStorage.clear();
    if (mode) window.localStorage.setItem("admin-theme-mode", mode);
    if (accent) window.localStorage.setItem("admin-theme-accent", accent);
    await handleLogout();
  }

  async function handleCopyPreference() {
    const payload = JSON.stringify({ themeMode, accent, uiPreferences }, null, 2);
    try {
      await navigator.clipboard.writeText(payload);
    } catch {
      // Ignore clipboard permission issues.
    }
  }

  useEffect(() => {
    document.documentElement.style.setProperty("--admin-accent", accent);
  }, [accent]);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setGlobalSearchOpen(true);
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const userMenuItems: MenuProps["items"] = [
    {
      key: "personal-settings",
      icon: <UserOutlined />,
      label: t("app.personalSettings"),
      onClick: () => navigate("/personal-settings"),
    },
    {
      key: "logout",
      icon: <LogoutOutlined />,
      label: t("app.logout"),
      onClick: handleLogout,
    },
  ];

  return (
    <LogStreamProvider>
    <Layout className={layoutClassName}>
      <Sider width={288} className="admin-sider" breakpoint="lg" collapsedWidth={0}>
        <div className="brand-block">
          <div className="brand-block__mark" aria-hidden="true">
            YS
          </div>
          <div>
            <Typography.Text className="brand-block__eyebrow">{t("brand.subtitle")}</Typography.Text>
            <Typography.Title level={4} className="brand-block__title">
              {t("brand.name")}
            </Typography.Title>
            <Typography.Text className="brand-block__subtitle">{t("brand.description")}</Typography.Text>
            <span className="brand-block__frame">&lt; {BRAND_EN_NAME} / OPS &gt;</span>
          </div>
        </div>

        <Menu
          key={menuEpoch}
          theme={menuTheme}
          mode="inline"
          selectedKeys={[selectedKey]}
          defaultOpenKeys={defaultOpenKeys}
          items={siderItems}
          className="admin-menu"
        />

        <div className="admin-sider__telemetry" aria-label="系统遥测">
          <strong>UNIT / YS-01</strong>
          <span>REV 2.6</span>
          <span>CTRL+K SEARCH</span>
        </div>
      </Sider>

      <Layout>
        <Header className="admin-header">
          <div className="admin-header__left">
            <div className="admin-header__title">{pageTitle}</div>
            <div className="admin-header__route-id">ROUTE / {pathname || "/"}</div>
          </div>
          <div className="admin-header__quick-actions">
            {uiPreferences.showRefresh ? (
              <Button type="text" className="admin-icon-btn" onClick={handleRefresh}>
                <ReloadOutlined />
              </Button>
            ) : null}
            {uiPreferences.showFullscreen ? (
              <Button type="text" className="admin-icon-btn" onClick={() => void handleToggleFullscreen()}>
                <FullscreenOutlined />
              </Button>
            ) : null}
            {uiPreferences.showThemeToggle ? (
              <Button type="text" className="admin-icon-btn" onClick={handleToggleMode} title="模式切换">
                {themeMode === "dark" ? <BulbOutlined /> : <BulbFilled />}
              </Button>
            ) : null}
            <Button type="text" className="admin-icon-btn" onClick={() => setGlobalSearchOpen(true)} title={`${t("app.search")} (Ctrl+K)`}>
              <SearchOutlined />
            </Button>
            <Select
              size="small"
              style={{ width: 96 }}
              value={activeLocale}
              options={[
                { label: "中文", value: "zh-CN" },
                { label: "EN", value: "en-US" },
              ]}
              onChange={(v: string) => {
                void i18n.changeLanguage(v);
                window.localStorage.setItem("app-locale", v);
                document.documentElement.lang = v.startsWith("en") ? "en" : "zh-CN";
                setMenuEpoch((n) => n + 1);
              }}
            />
            <Button type="text" className="admin-icon-btn" onClick={() => setSettingsOpen(true)} title={t("app.settings")}>
              <SettingOutlined />
            </Button>
          </div>
          <div className="admin-header__user-wrap">
            <Dropdown
              trigger={["click"]}
              placement="bottomRight"
              menu={{ items: userMenuItems }}
              dropdownRender={(menu) => (
                <div className="user-header-dropdown-panel">
                  <div className="user-header-dropdown-panel__head">
                    <Avatar className="user-header-dropdown-panel__avatar" size={40}>
                      {user?.nickname?.slice(0, 1) || user?.username?.slice(0, 1) || "Y"}
                    </Avatar>
                    <div className="user-header-dropdown-panel__text">
                      <div className="user-header-dropdown-panel__name">{user?.nickname || user?.username || t("app.notLoggedIn")}</div>
                      {user?.username ? (
                        <div className="user-header-dropdown-panel__account">{user.username}</div>
                      ) : null}
                    </div>
                  </div>
                  {user?.roles?.length ? (
                    <div className="user-header-dropdown-panel__roles">
                      {user.roles.map((r) => (
                        <Tag key={r.id} color="blue" bordered={false}>
                          {r.name}
                        </Tag>
                      ))}
                    </div>
                  ) : null}
                  <div className="user-header-dropdown-panel__menu">{menu}</div>
                </div>
              )}
            >
              <Button type="text" className="user-header-trigger">
                <span className="user-header-trigger__inner">
                  <Avatar className="user-header-trigger__avatar" size={30}>
                    {user?.nickname?.slice(0, 1) || user?.username?.slice(0, 1) || "Y"}
                  </Avatar>
                  <span className="user-header-trigger__name">{user?.nickname || user?.username || t("app.notLoggedIn")}</span>
                  <DownOutlined className="user-header-trigger__caret" />
                </span>
              </Button>
            </Dropdown>
          </div>
        </Header>

        <LogStreamDockBar />
        <Content className="admin-content">
          {loading ? (
            <div className="page-loading">
              <Spin size="large" />
            </div>
          ) : (
            <Outlet />
          )}
        </Content>

        <Drawer
          width={360}
          open={settingsOpen}
          onClose={() => setSettingsOpen(false)}
          className={`admin-settings-drawer ${themeMode === "dark" ? "is-dark" : "is-light"}`}
          closeIcon={<CloseOutlined />}
          title={t("app.preferences")}
          extra={
            <Button
              type="text"
              size="small"
              icon={themeMode === "dark" ? <BulbOutlined /> : <BulbFilled />}
              onClick={handleToggleMode}
            >
              {themeMode === "dark" ? t("app.themeDark") : t("app.themeLight")}
            </Button>
          }
        >
          <Tabs
            activeKey={settingsTab}
            onChange={setSettingsTab}
            className="admin-settings-tabs"
            items={[
              {
                key: "appearance",
                label: t("app.appearance"),
                children: (
                  <>
                    <div className="admin-settings-section">
                      <Typography.Text className="admin-settings-label">{t("app.themeMode")}</Typography.Text>
                      <Space size={8} style={{ marginTop: 10 }}>
                        <Button
                          className={themeMode === "light" ? "is-active" : ""}
                          onClick={() => applyThemeMode("light")}
                          icon={<BulbFilled />}
                        >
                          浅色
                        </Button>
                        <Button
                          className={themeMode === "dark" ? "is-active" : ""}
                          onClick={() => applyThemeMode("dark")}
                          icon={<BulbOutlined />}
                        >
                          深色
                        </Button>
                      </Space>
                    </div>

                    <div className="admin-settings-section">
                      <Typography.Text className="admin-settings-label">内置主题色</Typography.Text>
                      <div className="admin-accent-grid">
                        {["#e61919", "#ff2a2a", "#0ea5e9", "#14b8a6", "#eab308", "#10b981", "#6366f1", "#7c3aed", "#f97316", "#dc2626", "#4b5563", "#050505"].map((item) => (
                          <button
                            key={item}
                            type="button"
                            className={`admin-accent-dot ${accent === item ? "is-active" : ""}`}
                            style={{ background: item }}
                            onClick={() => applyAccent(item)}
                            aria-label={`切换主题色 ${item}`}
                          />
                        ))}
                      </div>
                    </div>
                  </>
                ),
              },
              {
                key: "layout",
                label: "布局",
                children: (
                  <div className="admin-settings-section">
                    <Typography.Text className="admin-settings-label">顶部快捷区</Typography.Text>
                    <div className="admin-setting-row">
                      <span>显示刷新按钮</span>
                      <Switch size="small" checked={uiPreferences.showRefresh} onChange={(checked) => patchUIPreferences({ showRefresh: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>显示全屏按钮</span>
                      <Switch size="small" checked={uiPreferences.showFullscreen} onChange={(checked) => patchUIPreferences({ showFullscreen: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>显示主题切换按钮</span>
                      <Switch size="small" checked={uiPreferences.showThemeToggle} onChange={(checked) => patchUIPreferences({ showThemeToggle: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>内容区紧凑模式</span>
                      <Switch size="small" checked={uiPreferences.compactContent} onChange={(checked) => patchUIPreferences({ compactContent: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>深色侧边栏</span>
                      <Switch size="small" checked={uiPreferences.darkSider} onChange={(checked) => patchUIPreferences({ darkSider: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>深色顶栏</span>
                      <Switch size="small" checked={uiPreferences.darkHeader} onChange={(checked) => patchUIPreferences({ darkHeader: checked })} />
                    </div>
                  </div>
                ),
              },
              {
                key: "shortcuts",
                label: "快捷键",
                children: (
                  <div className="admin-settings-section">
                    <div className="admin-setting-row">
                      <span>开启全局搜索快捷键</span>
                      <Switch size="small" checked disabled />
                    </div>
                    <div className="admin-setting-row">
                      <span>按键提示浮层</span>
                      <Switch size="small" checked disabled />
                    </div>
                  </div>
                ),
              },
              {
                key: "general",
                label: "通用",
                children: (
                  <div className="admin-settings-section">
                    <div className="admin-setting-row">
                      <span>自动保存主题偏好</span>
                      <Switch size="small" checked disabled />
                    </div>
                    <Space style={{ marginTop: 10 }}>
                      <Button type="primary" onClick={() => void handleCopyPreference()}>
                        复制偏好设置
                      </Button>
                      <Button danger onClick={() => void handleClearCacheAndLogout()}>
                        清空缓存 & 退出登录
                      </Button>
                    </Space>
                  </div>
                ),
              },
            ]}
          />

          <div className="admin-settings-tip">
            <BgColorsOutlined />
            <span>设置会自动保存，下次打开自动恢复。</span>
          </div>
        </Drawer>
        <GlobalSearchModal open={globalSearchOpen} onClose={() => setGlobalSearchOpen(false)} />
      </Layout>
    </Layout>
    </LogStreamProvider>
  );
}
