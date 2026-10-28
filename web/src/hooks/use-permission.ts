import { useAuth } from "../contexts/auth-context";

/**
 * 判断当前用户是否具备某个接口动作（如 can("POST", "/api/v1/users")）。
 * 能力未加载或超管时返回 true；后端 Authorize 始终是最终拦截点。
 */
export function usePermission(action: string, resource: string) {
  const { can } = useAuth();
  return can(action, resource);
}
