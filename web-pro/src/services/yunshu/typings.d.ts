declare namespace YunshuAPI {
  type ApiResponse<T> = {
    code: number;
    message: string;
    error_code?: string;
    data: T;
  };

  type RoleItem = {
    id: number;
    name: string;
    code: string;
  };

  type UserItem = {
    id: number;
    username: string;
    email: string;
    nickname: string;
    phone?: string;
    status: number;
    roles: RoleItem[];
    must_change_password?: boolean;
  };

  type LoginResult = {
    user: UserItem;
    must_change_password?: boolean;
  };

  type PasswordLoginPayload = {
    username: string;
    password: string;
    captcha_key: string;
    code: string;
  };

  type SendPasswordLoginCodeResult = {
    captcha_key: string;
    image: string;
    expires_in: number;
    cooldown_in: number;
  };

  type MenuItem = {
    id: number;
    parent_id?: number;
    path: string;
    name: string;
    icon: string;
    sort: number;
    hidden: boolean;
    component: string;
    redirect: string;
    status: number;
    children?: MenuItem[];
  };

  type PluginManifest = {
    menu_path_prefixes?: string[];
    api_prefixes?: string[];
    depends_on?: string[];
    workers?: string[];
  };

  type PluginInfo = {
    name: string;
    description: string;
    enabled: boolean;
    manifest?: PluginManifest;
  };

  type PluginListResult = {
    plugins: PluginInfo[];
    enabled: string[];
    registered: string[];
  };
}
