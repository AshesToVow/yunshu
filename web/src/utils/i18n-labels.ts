import type { TFunction } from "i18next";
import i18n from "../i18n";

function translateOrEmpty(t: TFunction, key: string): string {
  if (!i18n.exists(key)) return "";
  const value = t(key);
  return typeof value === "string" ? value.trim() : "";
}

/** Resolve menu/group label: path key first, then Chinese name map, else fallback. */
export function resolveTranslatedLabel(
  path: string | undefined,
  fallback: string,
  t?: TFunction,
): string {
  if (!t) return fallback;
  const p = path?.trim();
  if (p) {
    const byPath = translateOrEmpty(t, `menu.routes.${p}`);
    if (byPath) return byPath;
    const byGroup = translateOrEmpty(t, `menu.groups.${p}`);
    if (byGroup) return byGroup;
  }
  const name = fallback.trim();
  if (name) {
    const byName = translateOrEmpty(t, `menu.names.${name}`);
    if (byName) return byName;
  }
  return fallback;
}

/** Longest-prefix match against menu.routes / layout.routes for header titles. */
export function resolveRouteTitle(pathname: string, t: TFunction): string {
  const exactLayout = translateOrEmpty(t, `layout.routes.${pathname}`);
  if (exactLayout) return exactLayout;
  const exactMenu = translateOrEmpty(t, `menu.routes.${pathname}`);
  if (exactMenu) return exactMenu;

  const bundle = i18n.getResourceBundle(i18n.language, "translation") as
    | { menu?: { routes?: Record<string, string> }; layout?: { routes?: Record<string, string> } }
    | undefined;
  const routeMap: Record<string, string> = {
    ...(bundle?.layout?.routes ?? {}),
    ...(bundle?.menu?.routes ?? {}),
  };

  let bestPath = "";
  let bestLabel = "";
  for (const [path, label] of Object.entries(routeMap)) {
    if (!path || path === "/" || typeof label !== "string" || !label.trim()) continue;
    if (pathname === path || pathname.startsWith(`${path}/`)) {
      if (path.length > bestPath.length) {
        bestPath = path;
        bestLabel = label.trim();
      }
    }
  }
  if (bestLabel) return bestLabel;
  return translateOrEmpty(t, "app.console") || "Console";
}
