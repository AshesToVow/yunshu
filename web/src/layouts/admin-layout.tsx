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
import { GlobalSearchModal } from "../components/global-search-modal";
import { MenuAccessGate } from "../components/menu-access-gate";
import { useAuth } from "../contexts/auth-context";
import { useMenuTree } from "../hooks/use-menu-tree";
import { resolveAppLocale } from "../i18n";
import { buildSiderMenuItems, matchMenuSelectedKey, type AntdMenuItem } from "../utils/admin-menu";
import { resolveRouteTitle } from "../utils/i18n-labels";
import { usePlugins } from "../contexts/plugin-context";
import { filterAntdMenuItems } from "../modules/filter-menu";
import { useAdminThemeStore } from "../stores/admin-theme-store";

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
  darkSider: false,
  darkHeader: false,
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
  // 菜单树加载失败/为空时禁止回退到完整管理目录，避免无权限账号侧栏越权。
  const label = (path: string, fallback: string) => (
    <Link to={path}>{t(`menu.routes.${path}`, { defaultValue: fallback })}</Link>
  );
  return [{ key: "/", icon: <PieChartOutlined />, label: label("/", "总览页面") }];
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
  const { menus: menuTree } = useMenuTree();
  const [menuEpoch, setMenuEpoch] = useState(0);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [globalSearchOpen, setGlobalSearchOpen] = useState(false);
  const [settingsTab, setSettingsTab] = useState("appearance");
  const [uiPreferences, setUIPreferences] = useState<UIPreferences>(() => loadUIPreferences());
  const themeMode = useAdminThemeStore((s) => s.mode);
  const accent = useAdminThemeStore((s) => s.accent);
  const setThemeMode = useAdminThemeStore((s) => s.setMode);
  const setAccent = useAdminThemeStore((s) => s.setAccent);
  const toggleThemeMode = useAdminThemeStore((s) => s.toggleMode);

  const activeLocale = resolveAppLocale(i18n.language);

  const siderItems = useMemo(() => {
    if (menuTree?.length) {
      const items = buildSiderMenuItems(menuTree, t);
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
    "ops-platform",
    themeMode === "dark" ? "theme-dark" : "theme-light",
    uiPreferences.compactContent ? "layout-compact" : "",
    uiPreferences.darkSider ? "layout-dark-sider" : "layout-soft-sider",
    uiPreferences.darkHeader ? "layout-dark-header" : "layout-soft-header",
  ]
    .filter(Boolean)
    .join(" ");

  const pageTitle = useMemo(() => resolveRouteTitle(pathname, t), [pathname, t, activeLocale]);

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
  }

  function handleToggleMode() {
    toggleThemeMode();
  }

  function applyAccent(next: string) {
    setAccent(next);
  }

  function patchUIPreferences(patch: Partial<UIPreferences>) {
    setUIPreferences((prev) => {
      const next = { ...prev, ...patch };
      window.localStorage.setItem(UI_PREFS_KEY, JSON.stringify(next));
      return next;
    });
  }

  async function handleClearCacheAndLogout() {
    // 精确清理业务缓存：只删本应用写入的键，不用 localStorage.clear()。
    // clear() 会连带清掉同源下第三方/未来新增的持久化键（如 zustand persist、i18n 语言），
    // 且每加一个需要保留的键都得手动“读出-clear-回填”，极易漏。
    // 精确按前缀删除业务缓存；保留主题（zustand persist）、UI 偏好、语言。
    const preservedExact = new Set<string>(["admin-theme", UI_PREFS_KEY, "app-locale"]);
    const preservedPrefixes = ["admin-theme"]; // 含 legacy admin-theme-mode / admin-theme-accent
    const removable: string[] = [];
    for (let i = 0; i < window.localStorage.length; i += 1) {
      const key = window.localStorage.key(i);
      if (!key || preservedExact.has(key)) continue;
      if (preservedPrefixes.some((p) => key === p || key.startsWith(`${p}-`))) continue;
      if (key.startsWith("permission-system-") || key.startsWith("admin-") || key.startsWith("yunshu-")) {
        removable.push(key);
      }
    }
    removable.forEach((key) => window.localStorage.removeItem(key));
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

      <Layout className="admin-main">
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
              <Button type="text" className="admin-icon-btn" onClick={handleToggleMode} title={t("app.themeMode")}>
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

        <Content className="admin-content">
          {loading ? (
            <div className="page-loading">
              <Spin size="large" />
            </div>
          ) : (
            <MenuAccessGate>
              <Outlet />
            </MenuAccessGate>
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
                          {t("app.themeLight")}
                        </Button>
                        <Button
                          className={themeMode === "dark" ? "is-active" : ""}
                          onClick={() => applyThemeMode("dark")}
                          icon={<BulbOutlined />}
                        >
                          {t("app.themeDark")}
                        </Button>
                      </Space>
                    </div>

                    <div className="admin-settings-section">
                      <Typography.Text className="admin-settings-label">{t("app.accentColor")}</Typography.Text>
                      <div className="admin-accent-grid">
                        {["#0d9488", "#14b8a6", "#0ea5e9", "#10b981", "#eab308", "#6366f1", "#7c3aed", "#f97316", "#dc2626", "#4b5563", "#050505"].map((item) => (
                          <button
                            key={item}
                            type="button"
                            className={`admin-accent-dot ${accent === item ? "is-active" : ""}`}
                            style={{ background: item }}
                            onClick={() => applyAccent(item)}
                            aria-label={`${t("app.accentColor")} ${item}`}
                          />
                        ))}
                      </div>
                    </div>
                  </>
                ),
              },
              {
                key: "layout",
                label: t("app.layout"),
                children: (
                  <div className="admin-settings-section">
                    <Typography.Text className="admin-settings-label">{t("app.headerShortcuts")}</Typography.Text>
                    <div className="admin-setting-row">
                      <span>{t("app.showRefresh")}</span>
                      <Switch size="small" checked={uiPreferences.showRefresh} onChange={(checked) => patchUIPreferences({ showRefresh: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>{t("app.showFullscreen")}</span>
                      <Switch size="small" checked={uiPreferences.showFullscreen} onChange={(checked) => patchUIPreferences({ showFullscreen: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>{t("app.showThemeToggle")}</span>
                      <Switch size="small" checked={uiPreferences.showThemeToggle} onChange={(checked) => patchUIPreferences({ showThemeToggle: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>{t("app.compactContent")}</span>
                      <Switch size="small" checked={uiPreferences.compactContent} onChange={(checked) => patchUIPreferences({ compactContent: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>{t("app.darkSider")}</span>
                      <Switch size="small" checked={uiPreferences.darkSider} onChange={(checked) => patchUIPreferences({ darkSider: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>{t("app.darkHeader")}</span>
                      <Switch size="small" checked={uiPreferences.darkHeader} onChange={(checked) => patchUIPreferences({ darkHeader: checked })} />
                    </div>
                  </div>
                ),
              },
              {
                key: "shortcuts",
                label: t("app.shortcuts"),
                children: (
                  <div className="admin-settings-section">
                    <div className="admin-setting-row">
                      <span>{t("app.enableSearchHotkey")}</span>
                      <Switch size="small" checked disabled />
                    </div>
                    <div className="admin-setting-row">
                      <span>{t("app.hotkeyHintOverlay")}</span>
                      <Switch size="small" checked disabled />
                    </div>
                  </div>
                ),
              },
              {
                key: "general",
                label: t("app.general"),
                children: (
                  <div className="admin-settings-section">
                    <div className="admin-setting-row">
                      <span>{t("app.autoSaveTheme")}</span>
                      <Switch size="small" checked disabled />
                    </div>
                    <Space style={{ marginTop: 10 }}>
                      <Button type="primary" onClick={() => void handleCopyPreference()}>
                        {t("app.copyPreferences")}
                      </Button>
                      <Button danger onClick={() => void handleClearCacheAndLogout()}>
                        {t("app.clearCacheLogout")}
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
  );
}
