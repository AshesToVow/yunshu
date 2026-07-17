import { getData, http } from "./http";

export interface MenuItem {
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
  created_at: string;
  updated_at: string;
  children?: MenuItem[];
}

export interface MenuCreatePayload {
  parent_id?: number;
  path?: string;
  name: string;
  icon?: string;
  sort?: number;
  hidden?: boolean;
  component?: string;
  redirect?: string;
  status: number;
}

export interface MenuUpdatePayload {
  parent_id?: number;
  path?: string;
  name?: string;
  icon?: string;
  sort?: number;
  hidden?: boolean;
  component?: string;
  redirect?: string;
  status?: number;
}

export interface MenuBatchStatusPayload {
  ids: number[];
  status: number;
}

export function getMenuTree() {
  return getData<MenuItem[]>(http.get("/menus/tree"));
}

export function createMenu(payload: MenuCreatePayload) {
  return getData<MenuItem>(http.post("/menus", payload));
}

export function updateMenu(id: number, payload: MenuUpdatePayload) {
  return getData<MenuItem>(http.put(`/menus/${id}`, payload));
}

export function deleteMenu(id: number) {
  return getData<void>(http.delete(`/menus/${id}`));
}

export function batchUpdateMenuStatus(payload: MenuBatchStatusPayload) {
  return getData<void>(http.put("/menus/status", payload));
}

export interface MenuPermissionBindingItem {
  id?: number;
  resource: string;
  action: string;
  mode?: string;
}

export interface MenuEntryPermission {
  resource: string;
  action: string;
}

export interface MenuPermissionBindingsResponse {
  menu_id: number;
  menu_path: string;
  custom: MenuPermissionBindingItem[];
  default: MenuEntryPermission[];
  effective: MenuEntryPermission[];
}

export function getMenuBindings(menuId: number) {
  return getData<MenuPermissionBindingsResponse>(http.get(`/menus/${menuId}/bindings`));
}

export function replaceMenuBindings(menuId: number, bindings: MenuPermissionBindingItem[]) {
  return getData<{ message: string }>(http.put(`/menus/${menuId}/bindings`, { bindings }));
}
